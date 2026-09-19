// SPDX-License-Identifier: GPL-3.0-or-later
package policy

import (
	"strings"
	"testing"
)

// shippedPoliciesPath points at the seed policy file baked into the image
// (configs/policies.json, two levels up from this package).
const shippedPoliciesPath = "../../configs/policies.json"

// criticalPlatformDomains is a small denylist of major first-party
// platforms that a first-time, non-technical user would not expect (and
// cannot easily diagnose) to be broken by a demo default. Blocking any of
// these breaks core device functionality (App Store/iMessage, Google
// sign-in/Play Services, Windows Update, Amazon purchases/Alexa, WhatsApp,
// Cloudflare-fronted sites, GitHub, etc.) on first boot.
var criticalPlatformDomains = []string{
	"apple.com",
	"icloud.com",
	"google.com",
	"gmail.com",
	"microsoft.com",
	"windowsupdate.com",
	"live.com",
	"amazon.com",
	"amazonaws.com",
	"meta.com",
	"facebook.com",
	"whatsapp.com",
	"instagram.com",
	"cloudflare.com",
	"github.com",
}

// TestShippedPolicies_DoNotBlockCriticalPlatformDomains loads the actual
// seed file shipped in configs/policies.json and fails if any BLOCK policy
// contains (or is a subdomain match for) a critical first-party platform
// domain. This is a regression guard for the "block-shopping" policy that
// used to block apple.com out of the box, breaking iCloud/App
// Store/iMessage for any first-time Apple device user.
func TestShippedPolicies_DoNotBlockCriticalPlatformDomains(t *testing.T) {
	policies, err := LoadPoliciesFromFile(shippedPoliciesPath)
	if err != nil {
		t.Fatalf("failed to load shipped policies.json: %v", err)
	}
	if len(policies) == 0 {
		t.Fatalf("shipped policies.json produced no policies")
	}

	for _, p := range policies {
		if strings.ToUpper(p.Action) != "BLOCK" {
			continue
		}
		for _, domain := range p.Domains {
			for _, critical := range criticalPlatformDomains {
				if domain == critical || strings.HasSuffix(domain, "."+critical) {
					t.Errorf("policy %q (%s) blocks critical platform domain %q — this breaks first-boot connectivity for a common device/platform", p.ID, p.Name, domain)
				}
			}
		}
	}
}

// TestShippedPolicies_NoShoppingPolicy guards specifically against the
// removed demo policy resurfacing under any name/ID.
func TestShippedPolicies_NoShoppingPolicy(t *testing.T) {
	policies, err := LoadPoliciesFromFile(shippedPoliciesPath)
	if err != nil {
		t.Fatalf("failed to load shipped policies.json: %v", err)
	}
	for _, p := range policies {
		if p.ID == "block-shopping" {
			t.Errorf("shipped policies.json still contains the removed %q policy", p.ID)
		}
	}
}

// TestShippedPolicies_StillHasAnAdBlockDemo makes sure the fix didn't
// over-correct into blocking nothing — the first `dig` demo should still
// show a block against a safe, non-platform ad/tracker domain.
func TestShippedPolicies_StillHasAnAdBlockDemo(t *testing.T) {
	policies, err := LoadPoliciesFromFile(shippedPoliciesPath)
	if err != nil {
		t.Fatalf("failed to load shipped policies.json: %v", err)
	}

	found := false
	for _, p := range policies {
		if strings.ToUpper(p.Action) == "BLOCK" && len(p.Domains) > 0 {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected at least one BLOCK policy with domains so the default install still demonstrates blocking")
	}
}
