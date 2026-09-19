package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/hydradns/hydra-cli/selfupdate"
)

// TestUpdateCheckDoesNotModify verifies that `hydra update --check` reports an
// available update but never downloads an asset or touches the binary.
func TestUpdateCheckDoesNotModify(t *testing.T) {
	var downloadHits int32
	var srv *httptest.Server

	assetName := "hydra_linux_amd64"
	binContent := []byte("new binary")

	mux := http.NewServeMux()
	mux.HandleFunc("/download/bin", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&downloadHits, 1)
		w.Write(binContent)
	})
	mux.HandleFunc("/download/checksums", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&downloadHits, 1)
		fmt.Fprintf(w, "%s  %s\n", selfupdate.Sum256(binContent), assetName)
	})
	mux.HandleFunc("/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		rel := selfupdate.Release{
			TagName: "v99.0.0", // strictly newer than the built-in Version
			Assets: []selfupdate.Asset{
				{Name: assetName, URL: srv.URL + "/download/bin"},
				{Name: "checksums.txt", URL: srv.URL + "/download/checksums"},
			},
		}
		json.NewEncoder(w).Encode(rel)
	})
	srv = httptest.NewServer(mux)
	defer srv.Close()

	// Reset command flag state and run: hydra update --check --url <server>.
	updateCheckOnly = false
	updateURL = ""
	rootCmd.SetOut(io.Discard)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"update", "--check", "--url", srv.URL + "/releases/latest"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("update --check returned error: %v", err)
	}

	if n := atomic.LoadInt32(&downloadHits); n != 0 {
		t.Fatalf("--check must not download anything, but saw %d download(s)", n)
	}
}
