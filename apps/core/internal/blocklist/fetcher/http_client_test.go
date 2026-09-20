package fetcher

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hydradns/hydradns/apps/core/internal/blocklist/parser"
)

func TestCheckDialAddress(t *testing.T) {
	cases := []struct {
		addr    string
		blocked bool
	}{
		{"93.184.216.34:443", false},
		{"192.168.1.10:80", false}, // LAN-hosted lists stay allowed
		{"10.0.0.5:8080", false},
		{"[2001:db8::1]:443", false},
		{"127.0.0.1:80", true},
		{"[::1]:80", true},
		{"169.254.169.254:80", true}, // cloud metadata
		{"[fe80::1]:80", true},
		{"0.0.0.0:80", true},
		{"224.0.0.1:80", true},
		{"not-an-ip:80", true},
		{"no-port", true},
	}
	for _, c := range cases {
		err := checkDialAddress(c.addr, false)
		if c.blocked && !errors.Is(err, errBlockedTarget) {
			t.Errorf("%s: want blocked, got %v", c.addr, err)
		}
		if !c.blocked && err != nil {
			t.Errorf("%s: want allowed, got %v", c.addr, err)
		}
	}
	if err := checkDialAddress("127.0.0.1:80", true); err != nil {
		t.Errorf("loopback with allowLoopback: %v", err)
	}
}

func TestFetchRejectsBadURLsWithoutRetrying(t *testing.T) {
	f := NewHTTPFetcher()
	for _, u := range []string{"file:///etc/passwd", "ftp://example.com/list", "gopher://x/", "http://", "://bad", ""} {
		start := time.Now()
		_, _, err := f.Fetch(context.Background(), parser.SourceConfig{URL: u}, "")
		if !errors.Is(err, errBlockedTarget) {
			t.Errorf("%q: want errBlockedTarget, got %v", u, err)
		}
		if time.Since(start) > 2*time.Second {
			t.Errorf("%q: rejection took %s, looks retried", u, time.Since(start))
		}
	}
}

func TestFetchRefusesLoopbackQuickly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ads.example.com\n"))
	}))
	defer srv.Close()

	start := time.Now()
	_, _, err := NewHTTPFetcher().Fetch(context.Background(), parser.SourceConfig{URL: srv.URL}, "")
	if !errors.Is(err, errBlockedTarget) {
		t.Fatalf("want errBlockedTarget for loopback server, got %v", err)
	}
	if time.Since(start) > 5*time.Second {
		t.Fatalf("blocked target was retried for %s", time.Since(start))
	}
}

func TestFetchRefusesRedirectToBlockedTarget(t *testing.T) {
	for name, location := range map[string]string{
		"metadata address": "http://169.254.169.254/latest/meta-data/",
		"file scheme":      "file:///etc/passwd",
	} {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, location, http.StatusFound)
			}))
			defer srv.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			_, _, err := newHTTPFetcher(true).Fetch(ctx, parser.SourceConfig{URL: srv.URL}, "")
			if !errors.Is(err, errBlockedTarget) {
				t.Fatalf("want errBlockedTarget, got %v", err)
			}
		})
	}
}

func TestFetchStopsRedirectLoops(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, srv.URL+"/again", http.StatusFound)
	}))
	defer srv.Close()

	_, _, err := newHTTPFetcher(true).Fetch(context.Background(), parser.SourceConfig{URL: srv.URL}, "")
	if !errors.Is(err, errBlockedTarget) {
		t.Fatalf("want errBlockedTarget after too many redirects, got %v", err)
	}
}

func TestFetchHappyPathAndETag(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("If-None-Match") == `"v1"` {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", `"v1"`)
		_, _ = w.Write([]byte("ads.example.com\n"))
	}))
	defer srv.Close()

	f := newHTTPFetcher(true)
	body, etag, err := f.Fetch(context.Background(), parser.SourceConfig{URL: srv.URL}, "")
	if err != nil || etag != `"v1"` || !strings.Contains(string(body), "ads.example.com") {
		t.Fatalf("first fetch: body=%q etag=%q err=%v", body, etag, err)
	}
	body, etag, err = f.Fetch(context.Background(), parser.SourceConfig{URL: srv.URL}, `"v1"`)
	if err != nil || body != nil || etag != `"v1"` {
		t.Fatalf("304 fetch: body=%q etag=%q err=%v", body, etag, err)
	}
}
