package panel

import (
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
)

func TestEnJSRouteAndInjection(t *testing.T) {
	p := newTestPanel()

	// 1. /panel/en.js must be publicly accessible (200 OK) without API key
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, httptest.NewRequest("GET", "/panel/en.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /panel/en.js code=%d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/javascript") {
		t.Errorf("GET /panel/en.js Content-Type=%q, want text/javascript", ct)
	}
	if csp := rec.Header().Get("Content-Security-Policy"); csp == "" {
		t.Error("GET /panel/en.js missing CSP security header")
	}

	// 2. /panel/ HTML must inject en.js with cache-busting fingerprint right after app.js
	pageRec := httptest.NewRecorder()
	p.ServeHTTP(pageRec, httptest.NewRequest("GET", "/panel/", nil))
	body := pageRec.Body.String()
	if !strings.Contains(body, `<script src="en.js?v=`) {
		t.Errorf("GET /panel/ HTML does not contain <script src=\"en.js?v=...\">")
	}
}

func TestZeroModificationsToUpstream(t *testing.T) {
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not available; skipping diff check")
	}

	// Verify app.js and index.html are completely untouched against git tracking
	for _, file := range []string{"app.js", "index.html"} {
		cmd := exec.Command(git, "diff", "--name-only", "HEAD", file)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git diff failed: %v\n%s", err, out)
		}
		if len(strings.TrimSpace(string(out))) > 0 {
			t.Fatalf("CRITICAL CONSTRAINT VIOLATION: upstream file %s has been modified! Changes:\n%s", file, out)
		}
	}
}
