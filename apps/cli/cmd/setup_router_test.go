// SPDX-License-Identifier: GPL-3.0-or-later
package cmd

import (
	"bytes"
	"strings"
	"testing"
	"text/template"
	"time"
)

// renderTemplate is a thin wrapper around template.Execute that lets us
// exercise each router template directly without spinning up the cobra
// flag parser.
func renderTemplate(t *testing.T, kind string, data routerTmplData) string {
	t.Helper()
	tmplStr, ok := routerTemplates[kind]
	if !ok {
		t.Fatalf("unknown template kind %q", kind)
	}
	tmpl := template.Must(template.New(kind).Funcs(template.FuncMap{
		"now": func() string { return "2026-01-01" },
	}).Parse(tmplStr))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("execute %s: %v", kind, err)
	}
	return buf.String()
}

func TestRouterTemplate_AllKindsRender(t *testing.T) {
	data := routerTmplData{
		HydraIP:     "192.168.1.50",
		BlockDoHIPs: true,
		DoHIPs:      dohProviderIPs,
	}
	for _, kind := range []string{"pfsense", "mikrotik", "openwrt", "asus"} {
		kind := kind
		t.Run(kind, func(t *testing.T) {
			out := renderTemplate(t, kind, data)
			if !strings.Contains(out, "192.168.1.50") {
				t.Errorf("HydraIP not substituted in %s output", kind)
			}
			if !strings.Contains(out, "1.1.1.1") {
				t.Errorf("expected DoH-blackhole rule for Cloudflare in %s", kind)
			}
			if !strings.Contains(out, "8.8.8.8") {
				t.Errorf("expected DoH-blackhole rule for Google in %s", kind)
			}
			// Date stub from the test helper should appear so we know
			// `now` is wired correctly.
			if !strings.Contains(out, "2026-01-01") {
				t.Errorf("template did not call the now func in %s", kind)
			}
		})
	}
}

func TestRouterTemplate_BlockDoHIPsToggle(t *testing.T) {
	withBlock := renderTemplate(t, "pfsense", routerTmplData{
		HydraIP:     "192.168.1.50",
		BlockDoHIPs: true,
		DoHIPs:      dohProviderIPs,
	})
	withoutBlock := renderTemplate(t, "pfsense", routerTmplData{
		HydraIP:     "192.168.1.50",
		BlockDoHIPs: false,
	})
	if !strings.Contains(withBlock, "1.1.1.1") {
		t.Error("expected Cloudflare rule when BlockDoHIPs=true")
	}
	if strings.Contains(withoutBlock, "1.1.1.1") {
		t.Error("expected NO Cloudflare rule when BlockDoHIPs=false")
	}
	if len(withoutBlock) >= len(withBlock) {
		t.Errorf("output should shrink when BlockDoHIPs=false (with=%d, without=%d)",
			len(withBlock), len(withoutBlock))
	}
}

func TestRouterTemplate_HydraIPAppearsAtUsageSites(t *testing.T) {
	out := renderTemplate(t, "openwrt", routerTmplData{
		HydraIP:     "10.0.0.5",
		BlockDoHIPs: false,
	})
	// At minimum the LAN -> HydraDNS allow rule and the DNAT redirect
	// should both reference the IP. If the template forgets one of these
	// the lock-down is incomplete.
	if !strings.Contains(out, "-d 10.0.0.5") {
		t.Error("expected `-d 10.0.0.5` ACCEPT rule for HydraDNS itself")
	}
	if !strings.Contains(out, "DNAT --to-destination 10.0.0.5:53") {
		t.Error("expected DNAT redirect rule to 10.0.0.5:53")
	}
}

func TestDoHProviderIPs_Wellformed(t *testing.T) {
	if len(dohProviderIPs) == 0 {
		t.Fatal("dohProviderIPs is empty")
	}
	seen4 := make(map[string]bool, len(dohProviderIPs))
	seen6 := make(map[string]bool, len(dohProviderIPs))
	for _, p := range dohProviderIPs {
		if p.Owner == "" {
			t.Errorf("entry has no Owner: %+v", p)
		}
		if p.IP4 == "" || p.IP6 == "" {
			t.Errorf("entry %s missing v4 or v6: %+v", p.Owner, p)
		}
		if seen4[p.IP4] {
			t.Errorf("duplicate IPv4: %s", p.IP4)
		}
		if seen6[p.IP6] {
			t.Errorf("duplicate IPv6: %s", p.IP6)
		}
		seen4[p.IP4] = true
		seen6[p.IP6] = true
	}
}

func TestCurrentDateString_UsesTimeNowSeam(t *testing.T) {
	original := timeNow
	t.Cleanup(func() { timeNow = original })

	timeNow = func() time.Time {
		return time.Date(2030, 6, 15, 0, 0, 0, 0, time.UTC)
	}
	if got := currentDateString(); got != "2030-06-15" {
		t.Errorf("currentDateString: got %q, want 2030-06-15", got)
	}
}
