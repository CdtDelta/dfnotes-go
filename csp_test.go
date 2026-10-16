package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func parseCSP(t *testing.T, policy string) map[string][]string {
	t.Helper()
	directives := map[string][]string{}
	for _, part := range strings.Split(policy, ";") {
		fields := strings.Fields(part)
		if len(fields) == 0 {
			continue
		}
		if _, dup := directives[fields[0]]; dup {
			t.Fatalf("duplicate directive %q", fields[0])
		}
		directives[fields[0]] = fields[1:]
	}
	return directives
}

func TestCSPScriptSrcIsSelfOnly(t *testing.T) {
	d := parseCSP(t, contentSecurityPolicy)
	got := strings.Join(d["script-src"], " ")
	if got != "'self'" {
		t.Fatalf("script-src = %q, want 'self' only", got)
	}
}

func TestCSPLocksDownDefaults(t *testing.T) {
	d := parseCSP(t, contentSecurityPolicy)
	want := map[string]string{
		"default-src":     "'none'",
		"connect-src":     "'self'",
		"object-src":      "'none'",
		"base-uri":        "'none'",
		"form-action":     "'none'",
		"frame-src":       "'none'",
		"frame-ancestors": "'none'",
	}
	for name, value := range want {
		if got := strings.Join(d[name], " "); got != value {
			t.Errorf("%s = %q, want %q", name, got, value)
		}
	}
	for name, sources := range d {
		for _, s := range sources {
			if s == "*" || strings.HasPrefix(s, "http") || s == "'unsafe-eval'" {
				t.Errorf("%s allows %s", name, s)
			}
		}
	}
}

func TestWithCSPSetsHeader(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html></html>"))
	})
	rec := httptest.NewRecorder()
	withCSP(next).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if got := rec.Header().Get("Content-Security-Policy"); got != contentSecurityPolicy {
		t.Fatalf("header = %q, want policy", got)
	}
}
