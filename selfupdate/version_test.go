package selfupdate

import "testing"

func TestCompare(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.1", -1},    // older
		{"1.0.1", "1.0.0", 1},     // newer
		{"1.0.0", "1.0.0", 0},     // equal
		{"v1.2.3", "1.2.3", 0},    // leading v tolerated
		{"2.0.0", "1.9.9", 1},     // major dominates
		{"1.2", "1.2.0", 0},       // missing patch defaults to 0
		{"1.10.0", "1.9.0", 1},    // numeric, not lexical
		{"1.0.0-rc1", "1.0.0", 0}, // pre-release stripped in first cut
	}
	for _, c := range cases {
		got, err := Compare(c.a, c.b)
		if err != nil {
			t.Fatalf("Compare(%q,%q) unexpected error: %v", c.a, c.b, err)
		}
		if got != c.want {
			t.Errorf("Compare(%q,%q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestIsNewer(t *testing.T) {
	if n, err := IsNewer("1.0.0", "1.0.1"); err != nil || !n {
		t.Errorf("1.0.1 should be newer than 1.0.0 (got %v, err %v)", n, err)
	}
	if n, err := IsNewer("1.0.1", "1.0.0"); err != nil || n {
		t.Errorf("1.0.0 should not be newer than 1.0.1 (got %v, err %v)", n, err)
	}
	if n, err := IsNewer("1.0.0", "1.0.0"); err != nil || n {
		t.Errorf("equal versions should not be newer (got %v, err %v)", n, err)
	}
}

func TestParseVersionInvalid(t *testing.T) {
	for _, s := range []string{"", "abc", "1.x.0", "1.2.3.4", "-1.0.0"} {
		if _, err := parseVersion(s); err == nil {
			t.Errorf("parseVersion(%q) expected error, got nil", s)
		}
	}
}
