package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func startTestServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	var srv *httptest.Server
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Skipf("httptest server unavailable: %v", r)
			}
		}()
		srv = httptest.NewServer(handler)
	}()
	return srv
}
