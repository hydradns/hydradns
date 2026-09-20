// SPDX-License-Identifier: GPL-3.0-or-later
package utils

import "testing"

func TestAnonymizeIP_MasksLastOctetWithoutSecret(t *testing.T) {
	InitSecret("")
	secret = nil // no secret configured

	got := AnonymizeIP("192.168.1.42")
	want := "192.168.1.0"
	if got != want {
		t.Fatalf("AnonymizeIP() = %q, want %q", got, want)
	}
}

func TestAnonymizeIP_SameSecretIsStableAndDeterministic(t *testing.T) {
	InitSecret("test-secret-one")
	defer func() { secret = nil }()

	a := AnonymizeIP("203.0.113.7")
	b := AnonymizeIP("203.0.113.7")
	if a == "" || a != b {
		t.Fatalf("expected stable non-empty hash, got %q and %q", a, b)
	}
}

func TestAnonymizeIP_DifferentSecretsProduceDifferentHashes(t *testing.T) {
	InitSecret("test-secret-one")
	first := AnonymizeIP("203.0.113.7")

	InitSecret("test-secret-two")
	second := AnonymizeIP("203.0.113.7")
	defer func() { secret = nil }()

	if first == second {
		t.Fatalf("expected different secrets to produce different hashes, both were %q", first)
	}
}

func TestInitSecret_UsesKeyBytesVerbatimNotBase64(t *testing.T) {
	// "not-base64!!" is not valid base64; InitSecret must still accept it
	// since it does not base64-decode its input.
	InitSecret("not-base64!!")
	defer func() { secret = nil }()

	if got := AnonymizeIP("10.0.0.1"); got == "" {
		t.Fatalf("expected a hashed value, got empty string")
	}
}

// TestAnonymizeIP_WithSecretDistinguishesDevicesOnSameLAN guards against
// masking the address before hashing: zeroing the last IPv4 octet (or last
// 80 IPv6 bits) before hashing would map every device on the same /24 (in
// practice, every device on one home/office LAN) to the identical hash, a
// 100% collision rate, not merely a truncation-length risk. With a secret
// configured, two different hosts on the same /24 must produce different
// anonymized values.
func TestAnonymizeIP_WithSecretDistinguishesDevicesOnSameLAN(t *testing.T) {
	InitSecret("household-secret")
	defer func() { secret = nil }()

	a := AnonymizeIP("192.168.1.42")
	b := AnonymizeIP("192.168.1.43")
	if a == "" || b == "" {
		t.Fatalf("expected non-empty hashes, got %q and %q", a, b)
	}
	if a == b {
		t.Fatalf("two different devices on the same /24 hashed to the same value %q — per-device visibility is broken", a)
	}
}

// TestAnonymizeIP_WithoutSecretStillMasksPerDeviceBits documents the
// no-secret fallback path: it intentionally keeps the coarser masking
// behavior (since there's no HMAC providing one-wayness in that case), so
// two devices on the same /24 DO collapse to the same masked value. This
// path should only ever be hit if AnonymizeIP runs before InitSecret.
func TestAnonymizeIP_WithoutSecretStillMasksPerDeviceBits(t *testing.T) {
	InitSecret("")
	secret = nil

	a := AnonymizeIP("192.168.1.42")
	b := AnonymizeIP("192.168.1.43")
	if a != b {
		t.Fatalf("expected masked (unhashed) fallback to collapse same-/24 addresses, got %q and %q", a, b)
	}
	if a != "192.168.1.0" {
		t.Fatalf("got %q, want masked 192.168.1.0", a)
	}
}

func TestAnonymizeIP_IPv6WithSecret(t *testing.T) {
	InitSecret("household-secret")
	defer func() { secret = nil }()

	a := AnonymizeIP("2001:db8::1")
	b := AnonymizeIP("2001:db8::2")
	if a == "" || b == "" {
		t.Fatalf("expected non-empty hashes for IPv6 addresses, got %q and %q", a, b)
	}
	if a == b {
		t.Fatalf("two different IPv6 hosts hashed to the same value %q", a)
	}
	// stable
	if again := AnonymizeIP("2001:db8::1"); again != a {
		t.Fatalf("expected stable hash for repeated calls, got %q then %q", a, again)
	}
}

func TestAnonymizeIP_IPv6WithoutSecretMasksLast80Bits(t *testing.T) {
	InitSecret("")
	secret = nil

	got := AnonymizeIP("2001:db8::1")
	want := "2001:db8::"
	if got != want {
		t.Fatalf("AnonymizeIP() = %q, want %q", got, want)
	}
}

func TestAnonymizeIP_InvalidInputReturnsEmpty(t *testing.T) {
	InitSecret("household-secret")
	defer func() { secret = nil }()

	if got := AnonymizeIP("not-an-ip"); got != "" {
		t.Fatalf("expected empty string for invalid input, got %q", got)
	}
	if got := AnonymizeIP(""); got != "" {
		t.Fatalf("expected empty string for empty input, got %q", got)
	}
}
