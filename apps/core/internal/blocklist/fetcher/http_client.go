package fetcher

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"syscall"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/hydradns/hydra-core/internal/blocklist/parser"
)

// maxBlocklistBytes caps a single blocklist download. The largest public
// lists are a few tens of MB; an unbounded read lets one bad URL exhaust
// memory on a Pi.
const maxBlocklistBytes = 128 << 20

// maxRedirects bounds the redirect chain followed for one fetch.
const maxRedirects = 5

// errBlockedTarget marks a URL or address the fetcher refuses on purpose.
// Retrying cannot fix it, so Fetch treats it as permanent.
var errBlockedTarget = errors.New("blocklist fetch: target not allowed")

type HTTPFetcher struct {
	client *http.Client
}

// NewHTTPFetcher returns a fetcher for operator-supplied blocklist URLs.
//
// Fetching a URL an operator typed is the feature, so this cannot refuse
// "user-provided URLs". What it does refuse is the part of server-side
// request forgery that has no legitimate use here: non-http(s) schemes, and
// connections to loopback, link-local (including the 169.254.169.254 cloud
// metadata address), unspecified and multicast addresses. Private LAN
// addresses stay allowed because people host lists on their own network.
//
// The address check runs in the dialer, at connect time, on the IP actually
// being connected to. That covers every redirect hop and cannot be bypassed
// by a hostname that resolves to a blocked address (DNS rebinding).
func NewHTTPFetcher() *HTTPFetcher {
	return newHTTPFetcher(false)
}

// newHTTPFetcher's allowLoopback exists for tests, which serve from
// 127.0.0.1 via httptest.
func newHTTPFetcher(allowLoopback bool) *HTTPFetcher {
	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
		Control: func(_, address string, _ syscall.RawConn) error {
			return checkDialAddress(address, allowLoopback)
		},
	}
	transport := &http.Transport{
		// No proxy: with an environment proxy the dialer would check the
		// proxy's address instead of the real target.
		Proxy:                 nil,
		DialContext:           dialer.DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 20 * time.Second,
		MaxIdleConns:          4,
		IdleConnTimeout:       60 * time.Second,
	}
	return &HTTPFetcher{
		client: &http.Client{
			Timeout:   60 * time.Second,
			Transport: transport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= maxRedirects {
					return fmt.Errorf("%w: stopped after %d redirects", errBlockedTarget, maxRedirects)
				}
				return validateFetchURL(req.URL)
			},
		},
	}
}

// validateFetchURL accepts only absolute http(s) URLs with a host.
func validateFetchURL(u *url.URL) error {
	if u == nil {
		return fmt.Errorf("%w: URL is empty", errBlockedTarget)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%w: scheme %q (use http or https)", errBlockedTarget, u.Scheme)
	}
	if u.Hostname() == "" {
		return fmt.Errorf("%w: URL has no host", errBlockedTarget)
	}
	return nil
}

// checkDialAddress rejects connection targets that a blocklist download
// never needs. address is the "ip:port" the dialer is about to connect to.
func checkDialAddress(address string, allowLoopback bool) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("%w: bad dial address %q", errBlockedTarget, address)
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return fmt.Errorf("%w: dial address %q is not an IP", errBlockedTarget, host)
	}
	switch {
	case ip.IsLoopback():
		if allowLoopback {
			return nil
		}
		return fmt.Errorf("%w: loopback address %s", errBlockedTarget, ip)
	case ip.IsUnspecified():
		return fmt.Errorf("%w: unspecified address %s", errBlockedTarget, ip)
	case ip.IsLinkLocalUnicast(), ip.IsLinkLocalMulticast(), ip.IsInterfaceLocalMulticast():
		return fmt.Errorf("%w: link-local address %s", errBlockedTarget, ip)
	case ip.IsMulticast():
		return fmt.Errorf("%w: multicast address %s", errBlockedTarget, ip)
	}
	return nil
}

// Fetch returns body bytes, etag string, and error.
func (h *HTTPFetcher) Fetch(ctx context.Context, src parser.SourceConfig, knownETag string) ([]byte, string, error) {
	target, err := url.Parse(src.URL)
	if err != nil {
		return nil, "", fmt.Errorf("%w: %v", errBlockedTarget, err)
	}
	if err := validateFetchURL(target); err != nil {
		return nil, "", err
	}

	var body []byte
	var etag string

	op := func() error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
		if err != nil {
			return backoff.Permanent(err)
		}
		if knownETag != "" {
			req.Header.Set("If-None-Match", knownETag)
		}
		resp, err := h.client.Do(req)
		if err != nil {
			if errors.Is(err, errBlockedTarget) {
				return backoff.Permanent(err)
			}
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotModified {
			// nothing changed
			etag = knownETag
			body = nil
			return nil
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("http status %d", resp.StatusCode)
		}
		b, err := io.ReadAll(io.LimitReader(resp.Body, maxBlocklistBytes+1))
		if err != nil {
			return err
		}
		if len(b) > maxBlocklistBytes {
			return backoff.Permanent(fmt.Errorf("blocklist is larger than %d MiB", maxBlocklistBytes>>20))
		}
		etag = resp.Header.Get("ETag")
		body = b
		return nil
	}

	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 2 * time.Minute
	if err := backoff.Retry(op, backoff.WithContext(bo, ctx)); err != nil {
		return nil, "", err
	}
	return body, etag, nil
}
