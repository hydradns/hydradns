package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net"
)

var secret []byte

// InitSecret sets the HMAC key used by AnonymizeIP. key is used verbatim as
// key material (its raw bytes, not base64-decoded); this accepts anything
// from an operator-supplied HYDRA_ANON_SECRET value to a generated hex
// secret from config.ResolveAnonymizationSecret without requiring either to
// be valid base64.
func InitSecret(key string) {
	secret = []byte(key)
}

// AnonymizeIP turns a client IP into a value safe to persist in the query
// log: with a secret configured (the normal case whenever anonymization is
// enabled; see config.ResolveAnonymizationSecret), it's an HMAC-SHA256 of
// the exact address, truncated to 16 hex chars (64 bits); without one, a
// coarser, unhashed mask (IPv4: last octet zeroed; IPv6: last 80 bits
// zeroed).
//
// The two modes intentionally do NOT share the masking step. Masking
// zeroes device-specific bits before hashing would deterministically
// collapse every device on the same /24 (IPv4) or /48 (IPv6), in
// practice every device on one home/office LAN since that's almost
// always a single /24, onto the identical hash. That doesn't anonymize
// the per-client query log, it erases it: 100% of devices on a typical
// install become indistinguishable, not just harder to correlate. Hashing
// the address as-is avoids that: the secret (unique per install, see
// ResolveAnonymizationSecret) makes the stored value non-reversible and
// non-correlatable across installs, while still letting a device's own
// queries be grouped together in the dashboard, the actual point of the
// feature. 64 bits of HMAC output is not the weak link here: the birthday
// bound for even thousands of devices sharing one install's secret is
// astronomically below any realistic collision risk.
//
// The unhashed masking fallback only matters if AnonymizeIP is ever called
// before InitSecret (e.g. a test that builds an Engine directly); it is a
// deliberately conservative default, never a live production path, since
// callers only invoke AnonymizeIP when anonymization is enabled, and
// enabling it always resolves (or generates) a secret first.
func AnonymizeIP(ipStr string) string {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return ""
	}
	ip = ip.To16()

	if len(secret) == 0 {
		return maskIP(ip).String()
	}

	h := hmac.New(sha256.New, secret)
	h.Write(ip)
	return hex.EncodeToString(h.Sum(nil))[:16] // first 16 hex chars = 64 bits
}

// maskIP zeroes the device-specific portion of ip (IPv4: last octet, IPv6:
// last 80 bits), returning a coarser network-level address. Operates on a
// copy so callers' slices are never mutated in place.
func maskIP(ip net.IP) net.IP {
	out := make(net.IP, len(ip))
	copy(out, ip)
	if v4 := out.To4(); v4 != nil {
		v4[3] = 0
		return v4
	}
	for i := 6; i < len(out); i++ {
		out[i] = 0
	}
	return out
}
