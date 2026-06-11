// SPDX-License-Identifier: GPL-3.0-or-later
package dnsengine

import (
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/lopster568/phantomDNS/internal/config"
	"github.com/lopster568/phantomDNS/internal/logger"
	"github.com/lopster568/phantomDNS/internal/metrics"
	"github.com/lopster568/phantomDNS/internal/policy"
	"github.com/lopster568/phantomDNS/internal/storage/models"
	"github.com/lopster568/phantomDNS/internal/storage/repositories"
	"github.com/lopster568/phantomDNS/internal/threat"
	"github.com/miekg/dns"
)

type BlocklistChecker interface {
	IsBlocked(domain string) (bool, error)
}

type RuntimeState struct {
	acceptQueries atomic.Bool
	// policyEnabled atomic.Bool
	lastError atomic.Value
}

type Engine struct {
	upstreamManager *UpstreamManager
	policyEngine    *policy.Engine
	blocklist       BlocklistChecker
	state           *RuntimeState
	metrics         *metrics.QueryMetrics
	queryLog        repositories.QueryLogRepository
	statistics      repositories.StatisticsRepository
	threatDetector  *threat.Detector
	cache           *ResponseCache
}

// Upstream exchange budget: short per-attempt timeout with retries beats
// one long wait — a lost UDP packet costs 1.5s, not 5s, before failover.
const (
	upstreamTimeout    = 1500 * time.Millisecond
	upstreamMaxRetries = 2
)

func (e *Engine) AttachBlocklistChecker(b BlocklistChecker) {
	e.blocklist = b
}

func NewDNSEngine(cfg config.DataPlaneConfig, repos *repositories.Store, pE *policy.Engine) (*Engine, error) {
	mgr, err := NewUpstreamManager(cfg.UpstreamResolvers, 4)
	state := &RuntimeState{}
	state.acceptQueries.Store(false)
	qm := metrics.NewQueryMetrics()

	if err != nil {
		return nil, err
	}
	return &Engine{
		upstreamManager: mgr,
		policyEngine:    pE,
		state:           state,
		metrics:         qm,
		queryLog:        repos.QueryLogs,
		statistics:      repos.Statistics,
		threatDetector:  threat.NewDetector(),
		cache:           NewResponseCache(defaultCacheMaxEntries),
	}, nil
}

func (e *Engine) SetAcceptQueries(enabled bool) {
	e.state.acceptQueries.Store(enabled)
}

// Cleanup the resources used by the Engine
func (e *Engine) Shutdown() {
	if e.upstreamManager != nil {
		e.upstreamManager.Close()
	}
}

// blockResponseKind controls how respondBlocked answers a blocked query.
// Configurable via the BLOCK_RESPONSE env var so we can A/B test:
//
//	zero      (default) - return A 0.0.0.0 / AAAA ::. Linux clients fail
//	            connect immediately with ECONNREFUSED. Windows is slower
//	            because the TCP stack will try connect to 0.0.0.0 and wait
//	            for the OS connect timeout.
//	nxdomain            - return RcodeNameError (NXDOMAIN). Definitive
//	            "this name does not exist" answer per RFC 1034. Browsers
//	            on every OS give up immediately with DNS_PROBE_FINISHED_
//	            NXDOMAIN. Windows DNS Client caches this and does NOT fall
//	            back to secondary (unlike REFUSED).
//	refused             - return RcodeRefused. Original v0 behaviour.
//	            Windows + many home routers fall back to a secondary DNS
//	            on REFUSED, which silently bypasses every block. Do not
//	            use unless you have verified there is no secondary DNS.
//
// Default stays "zero" to preserve existing behaviour. Operators flip
// via env to test in their own browser before we change the default.
var blockResponseKind = func() string {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("BLOCK_RESPONSE")))
	switch v {
	case "nxdomain", "refused", "zero":
		return v
	default:
		return "zero"
	}
}()

// respondNXDomain writes an unconditional NXDOMAIN reply. Used by the
// DoH bootstrap interception path where we always want the browser to
// fall back to system DNS, regardless of the operator's BLOCK_RESPONSE
// preference for user-defined blocks.
func respondNXDomain(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetRcode(r, dns.RcodeNameError)
	if err := w.WriteMsg(m); err != nil {
		logger.Log.Error("Failed to write NXDOMAIN response: " + err.Error())
	}
}

func (e *Engine) respondBlocked(w dns.ResponseWriter, r *dns.Msg, domain, reason string) {
	m := new(dns.Msg)
	switch blockResponseKind {
	case "nxdomain":
		m.SetRcode(r, dns.RcodeNameError)
	case "refused":
		m.SetRcode(r, dns.RcodeRefused)
	default: // "zero"
		m.SetReply(r)
		qtype := r.Question[0].Qtype
		name := r.Question[0].Name
		switch qtype {
		case dns.TypeAAAA:
			rr, err := dns.NewRR(name + " 60 IN AAAA ::")
			if err == nil {
				m.Answer = append(m.Answer, rr)
			}
		default: // TypeA and everything else
			rr, err := dns.NewRR(name + " 60 IN A 0.0.0.0")
			if err == nil {
				m.Answer = append(m.Answer, rr)
			}
		}
	}
	if err := w.WriteMsg(m); err != nil {
		logger.Log.Error("Failed to write DNS block response: " + err.Error())
	}
}

func (e *Engine) respondRedirect(w dns.ResponseWriter, r *dns.Msg, domain, ip string) {
	m := new(dns.Msg)
	m.SetReply(r)
	rr, err := dns.NewRR(domain + " 60 IN A " + ip)
	if err != nil {
		logger.Log.Error("Failed to create redirect RR: " + err.Error())
		m.SetRcode(r, dns.RcodeServerFailure)
		_ = w.WriteMsg(m)
		return
	}
	m.Answer = append(m.Answer, rr)
	if err := w.WriteMsg(m); err != nil {
		logger.Log.Error("Failed to write DNS redirect response: " + err.Error())
	}
}

func (e *Engine) forwardUpstream(w dns.ResponseWriter, r *dns.Msg, domain string) {
	if cached := e.cache.Get(r); cached != nil {
		if err := w.WriteMsg(cached); err != nil {
			logger.Log.Error("Failed to write cached DNS response: " + err.Error())
		}
		return
	}

	resp, err := e.upstreamManager.Exchange(r, upstreamTimeout, upstreamMaxRetries)
	if err != nil {
		logger.Log.Error("Upstream query failed: " + err.Error())
		m := new(dns.Msg)
		m.SetRcode(r, dns.RcodeServerFailure)
		_ = w.WriteMsg(m)
		return
	}
	if resp == nil {
		logger.Log.Error("Upstream returned nil response for: " + domain)
		m := new(dns.Msg)
		m.SetRcode(r, dns.RcodeServerFailure)
		_ = w.WriteMsg(m)
		return
	}
	e.cache.Set(r, resp)
	if err := w.WriteMsg(resp); err != nil {
		logger.Log.Error("Failed to write DNS response: " + err.Error())
	}
}

// normalizeDomain lowercases and strips the trailing dot from a DNS FQDN.
func normalizeDomain(d string) string {
	return strings.TrimSuffix(strings.ToLower(d), ".")
}

// Known private/local DNS suffixes appended by routers via DHCP search domains.
// When a router advertises a search domain (e.g., "hgu_lan", "home", "local"),
// Windows/macOS append it to bare hostnames AND sometimes to FQDNs.
// We strip these before blocklist/policy checks so "godaddy.com.hgu_lan"
// still matches the "godaddy.com" block rule.
var localSuffixes = map[string]bool{
	"lan": true, "local": true, "home": true, "internal": true,
	"localdomain": true, "domain.name": true, "hgu_lan": true,
	"fritz.box": true, "mynetwork": true, "belkin": true,
	"router": true, "gateway": true, "dlink": true,
}

// stripSearchDomain removes known router-appended search domains from a query.
// e.g., "godaddy.com.hgu_lan" → "godaddy.com"
func stripSearchDomain(domain string) string {
	parts := strings.Split(domain, ".")
	if len(parts) < 3 {
		return domain
	}

	// Check last 1 and last 2 labels against known suffixes
	lastOne := parts[len(parts)-1]
	lastTwo := strings.Join(parts[len(parts)-2:], ".")

	if localSuffixes[lastTwo] {
		return strings.Join(parts[:len(parts)-2], ".")
	}
	if localSuffixes[lastOne] {
		return strings.Join(parts[:len(parts)-1], ".")
	}

	return domain
}

// ProcessDNSQuery processes the DNS query and returns a response
func (e *Engine) ProcessDNSQuery(w dns.ResponseWriter, r *dns.Msg) {
	if r == nil || len(r.Question) == 0 {
		return
	}

	if !e.state.acceptQueries.Load() {
		m := new(dns.Msg)
		m.SetRcode(r, dns.RcodeRefused)
		_ = w.WriteMsg(m)
		return
	}

	start := time.Now()
	success := false

	defer func() {
		elapsed := time.Since(start)
		e.metrics.Record(elapsed, success)
	}()

	domainName := stripSearchDomain(normalizeDomain(r.Question[0].Name))
	clientIP := ""
	if w.RemoteAddr() != nil {
		clientIP = w.RemoteAddr().String()
	}

	// Run threat detection on every query (non-blocking, just scoring)
	var threatResult threat.Result
	if e.threatDetector != nil {
		threatResult = e.threatDetector.Analyze(domainName)
	}

	// --- Step 0: DoH/DoT bootstrap interception ---
	// If a client is trying to reach a known encrypted-DNS provider, we
	// answer NXDOMAIN so the browser falls back to system DNS (which is
	// us). We force NXDOMAIN regardless of the operator's BLOCK_RESPONSE
	// setting because the alternatives (0.0.0.0 / REFUSED) defeat the
	// purpose: the browser would either hang on the connect or roll over
	// to its hardcoded fallback, both of which leak DNS off-net.
	//
	// This list is invisible to the dashboard and not user-editable; see
	// internal/dnsengine/doh_bootstrap.go.
	if IsDoHBootstrap(domainName) {
		logger.Log.Infof("Blocked DoH bootstrap: %s", domainName)
		e.logQuery(domainName, clientIP, "block", threatResult)
		respondNXDomain(w, r)
		success = true
		return
	}

	// --- Step 1: Check blocklist first ---
	if e.blocklist != nil {
		blocked, err := e.blocklist.IsBlocked(domainName)
		if err != nil {
			logger.Log.Error("Blocklist check failed: " + err.Error())
		} else if blocked {
			logger.Log.Infof("Blocked by blocklist: %s", domainName)
			e.logQuery(domainName, clientIP, "block", threatResult)
			e.respondBlocked(w, r, domainName, "blocklist")
			success = true
			return
		}
	}

	// --- Step 2: Evaluate policy ---
	decision, err := e.policyEngine.Evaluate(domainName)
	if err != nil {
		logger.Log.Error("Failed to evaluate policy: " + err.Error())
		e.logQuery(domainName, clientIP, "error", threatResult)
		m := new(dns.Msg)
		m.SetRcode(r, dns.RcodeServerFailure)
		_ = w.WriteMsg(m)
		return
	}

	switch decision.Action {
	case policy.ActionDeny:
		logger.Log.Infof("Blocking via policy %s", decision.PolicyID)
		e.logQuery(domainName, clientIP, "block", threatResult)
		e.respondBlocked(w, r, domainName, decision.PolicyID)
		success = true

	case policy.ActionRedirect:
		e.logQuery(domainName, clientIP, "redirect", threatResult)
		e.respondRedirect(w, r, domainName, decision.RedirectIP)
		success = true

	default: // policy.ActionAllow
		action := "allow"
		if threatResult.IsSuspicious {
			action = "flagged"
			logger.Log.Warnf("Suspicious domain allowed: %s (score=%.2f, method=%s)", domainName, threatResult.ThreatScore, threatResult.DetectionMethod)
		}
		e.logQuery(domainName, clientIP, action, threatResult)
		e.forwardUpstream(w, r, domainName)
		success = true
	}
}

func (e *Engine) logQuery(domain, clientIP, action string, tr threat.Result) {
	if e.queryLog == nil {
		return
	}
	q := &models.DNSQuery{
		Domain:          domain,
		ClientIP:        clientIP,
		Action:          action,
		IsSuspicious:    tr.IsSuspicious,
		ThreatScore:     tr.ThreatScore,
		DetectionMethod: tr.DetectionMethod,
		ThreatReason:    tr.Reason,
	}
	// Map "flagged" to "allow" for statistics (flagged domains are still forwarded)
	statsAction := action
	if statsAction == "flagged" {
		statsAction = "allow"
	}

	go func() {
		if err := e.queryLog.Save(q); err != nil {
			logger.Log.Errorf("Failed to log query: %v", err)
		}
		if e.statistics != nil {
			if err := e.statistics.IncrementCounter(statsAction); err != nil {
				logger.Log.Errorf("Failed to increment stats: %v", err)
			}
		}
	}()
}

func (e *Engine) Metrics() *metrics.QueryMetrics {
	return e.metrics
}
