package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/hydradns/hydra-cli/api"
)

// setupTestServer builds an httptest server that speaks the same
// envelope as apps/core cmd/controlplane/handlers/auth.go, tracking
// whether /auth/setup was ever hit so tests can assert "no password
// sent" paths really send nothing.
func setupTestServer(t *testing.T, alreadyComplete bool) (*httptest.Server, *int32) {
	t.Helper()
	var setupHits int32

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/status", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"status": "success",
			"data":   map[string]any{"setup_complete": alreadyComplete},
		})
	})
	mux.HandleFunc("/api/v1/auth/setup", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&setupHits, 1)

		if alreadyComplete {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]any{
				"status": "error",
				"error":  "setup already completed",
			})
			return
		}

		var req api.SetupRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{"status": "error", "error": "bad request"})
			return
		}
		if len(req.Password) < 8 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"status": "error",
				"error":  "password is required (minimum 8 characters)",
			})
			return
		}

		json.NewEncoder(w).Encode(map[string]any{
			"status": "success",
			"data":   map[string]any{"token": "tok-" + req.Password},
		})
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, &setupHits
}

// resetSetupState restores every package-level var setupCmd touches so
// tests don't leak state into each other or into other test files.
func resetSetupState(t *testing.T) (homeDir string) {
	t.Helper()
	setupEmail = ""
	setupPasswordStdin = false
	setupShowToken = false
	setupStdin = os.Stdin
	setupIsTerminal = func() bool { return false }
	setupReadPassword = func() ([]byte, error) { return nil, fmt.Errorf("not stubbed") }

	homeDir = t.TempDir()
	t.Setenv("HOME", homeDir)
	return homeDir
}

func TestSetup_HappyPath_StoresToken(t *testing.T) {
	home := resetSetupState(t)
	srv, hits := setupTestServer(t, false)

	client = api.New(srv.URL, "")
	setupPasswordStdin = true
	setupStdin = strings.NewReader("changeme123\n")

	if err := runSetup(setupCmd, nil); err != nil {
		t.Fatalf("runSetup returned error: %v", err)
	}
	if got := atomic.LoadInt32(hits); got != 1 {
		t.Fatalf("expected exactly 1 call to /auth/setup, got %d", got)
	}

	tokenPath := filepath.Join(home, ".hydra", "token")
	b, err := os.ReadFile(tokenPath)
	if err != nil {
		t.Fatalf("token file not written: %v", err)
	}
	if strings.TrimSpace(string(b)) != "tok-changeme123" {
		t.Fatalf("token content = %q, want %q", strings.TrimSpace(string(b)), "tok-changeme123")
	}
}

func TestSetup_AlreadyComplete_SendsNoPassword(t *testing.T) {
	resetSetupState(t)
	srv, hits := setupTestServer(t, true)

	client = api.New(srv.URL, "")
	setupPasswordStdin = true
	setupStdin = strings.NewReader("shouldnotmatter\n")

	err := runSetup(setupCmd, nil)
	if err == nil {
		t.Fatal("expected an error when setup is already complete")
	}
	if !strings.Contains(err.Error(), "hydra login") {
		t.Errorf("error should point to `hydra login`, got: %v", err)
	}
	if got := atomic.LoadInt32(hits); got != 0 {
		t.Fatalf("expected /auth/setup to never be called, got %d hit(s)", got)
	}
}

func TestSetup_InteractiveMismatch_NoRequestSent(t *testing.T) {
	resetSetupState(t)
	srv, hits := setupTestServer(t, false)
	client = api.New(srv.URL, "")

	setupIsTerminal = func() bool { return true }
	var call int
	setupReadPassword = func() ([]byte, error) {
		call++
		if call == 1 {
			return []byte("firstpassword"), nil
		}
		return []byte("secondpassword"), nil
	}

	err := runSetup(setupCmd, nil)
	if err == nil {
		t.Fatal("expected an error on password mismatch")
	}
	if !strings.Contains(err.Error(), "do not match") {
		t.Errorf("error = %v, want a mismatch message", err)
	}
	if got := atomic.LoadInt32(hits); got != 0 {
		t.Fatalf("mismatched passwords must never reach the server, got %d hit(s)", got)
	}
}

func TestSetup_InteractiveMatch_Succeeds(t *testing.T) {
	home := resetSetupState(t)
	srv, hits := setupTestServer(t, false)
	client = api.New(srv.URL, "")

	setupIsTerminal = func() bool { return true }
	setupReadPassword = func() ([]byte, error) { return []byte("samepassword"), nil }

	if err := runSetup(setupCmd, nil); err != nil {
		t.Fatalf("runSetup returned error: %v", err)
	}
	if got := atomic.LoadInt32(hits); got != 1 {
		t.Fatalf("expected exactly 1 call to /auth/setup, got %d", got)
	}
	if _, err := os.Stat(filepath.Join(home, ".hydra", "token")); err != nil {
		t.Fatalf("token file missing: %v", err)
	}
}

func TestSetup_WeakPasswordRejectedLocally(t *testing.T) {
	resetSetupState(t)
	srv, hits := setupTestServer(t, false)
	client = api.New(srv.URL, "")

	setupPasswordStdin = true
	setupStdin = strings.NewReader("short\n")

	err := runSetup(setupCmd, nil)
	if err == nil {
		t.Fatal("expected an error for a too-short password")
	}
	if !strings.Contains(err.Error(), "8 characters") {
		t.Errorf("error = %v, want a message about the 8-char minimum", err)
	}
	if got := atomic.LoadInt32(hits); got != 0 {
		t.Fatalf("a locally-rejected password must never reach the server, got %d hit(s)", got)
	}
}

func TestSetup_ServerErrorSurfaced(t *testing.T) {
	resetSetupState(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/status", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"status": "success",
			"data":   map[string]any{"setup_complete": false},
		})
	})
	mux.HandleFunc("/api/v1/auth/setup", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
			"status": "error",
			"error":  "failed to create admin",
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	client = api.New(srv.URL, "")
	setupPasswordStdin = true
	setupStdin = strings.NewReader("changeme123\n")

	err := runSetup(setupCmd, nil)
	if err == nil {
		t.Fatal("expected the server error to surface")
	}
	if !strings.Contains(err.Error(), "failed to create admin") {
		t.Errorf("error = %v, want it to contain the server's message", err)
	}
}

func TestSetup_APIUnreachable_FriendlyError(t *testing.T) {
	resetSetupState(t)
	// Point at a closed port so the connection fails outright.
	closed := httptest.NewServer(http.NotFoundHandler())
	closedURL := closed.URL
	closed.Close()

	client = api.New(closedURL, "")
	err := runSetup(setupCmd, nil)
	if err == nil {
		t.Fatal("expected an error when the API is unreachable")
	}
	if !strings.Contains(err.Error(), "docker compose ps") {
		t.Errorf("error = %v, want a hint about docker compose ps", err)
	}
}

func TestSetup_TokenFilePermissions(t *testing.T) {
	home := resetSetupState(t)
	srv, _ := setupTestServer(t, false)
	client = api.New(srv.URL, "")

	setupPasswordStdin = true
	setupStdin = strings.NewReader("changeme123\n")

	if err := runSetup(setupCmd, nil); err != nil {
		t.Fatalf("runSetup returned error: %v", err)
	}

	dir := filepath.Join(home, ".hydra")
	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if perm := dirInfo.Mode().Perm(); perm != 0700 {
		t.Errorf("dir perm = %o, want 0700", perm)
	}

	fileInfo, err := os.Stat(filepath.Join(dir, "token"))
	if err != nil {
		t.Fatalf("stat token file: %v", err)
	}
	if perm := fileInfo.Mode().Perm(); perm != 0600 {
		t.Errorf("token file perm = %o, want 0600", perm)
	}
}

func TestSetup_ShowTokenFlag(t *testing.T) {
	resetSetupState(t)
	srv, _ := setupTestServer(t, false)

	for _, tt := range []struct {
		name      string
		show      bool
		wantToken bool
	}{
		{"hidden by default", false, false},
		{"shown with --show-token", true, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			resetSetupState(t)
			client = api.New(srv.URL, "")
			setupPasswordStdin = true
			setupShowToken = tt.show
			setupStdin = strings.NewReader("changeme123\n")

			r, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("os.Pipe: %v", err)
			}
			origStdout := os.Stdout
			os.Stdout = w
			runErr := runSetup(setupCmd, nil)
			w.Close()
			os.Stdout = origStdout

			var buf bytes.Buffer
			if _, err := buf.ReadFrom(r); err != nil {
				t.Fatalf("read captured stdout: %v", err)
			}

			if runErr != nil {
				t.Fatalf("runSetup returned error: %v", runErr)
			}
			gotToken := strings.Contains(buf.String(), "tok-changeme123")
			if gotToken != tt.wantToken {
				t.Errorf("token in output = %v, want %v (output: %q)", gotToken, tt.wantToken, buf.String())
			}
		})
	}
}
