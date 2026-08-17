package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHealthzContract asserts ADR-0007:
//   - GET /healthz returns HTTP 200
//   - response body equals the literal string "ok"
func TestHealthzContract(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()

	healthzHandler(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
	if got, want := rr.Body.String(), "ok"; got != want {
		t.Errorf("body: got %q, want %q", got, want)
	}
}

// TestRootReturns200 is a smoke test that the homepage route is wired up
// (catches "container started but mux wasn't built" regressions).
func TestRootReturns200(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	rootHandler(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
	if rr.Body.Len() == 0 {
		t.Error("body: got empty, want non-empty")
	}
}