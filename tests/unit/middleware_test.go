//go:build unit

package unit_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	appmw "pipeline/apps/server/middleware"
)

func TestAPIKeyAuth_MissingHeader(t *testing.T) {
	handler := appmw.APIKeyAuth("secret")(okHandler())
	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPIKeyAuth_WrongKey(t *testing.T) {
	handler := appmw.APIKeyAuth("secret")(okHandler())
	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	req.Header.Set("X-API-Key", "wrong")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPIKeyAuth_CorrectKey(t *testing.T) {
	handler := appmw.APIKeyAuth("secret")(okHandler())
	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	req.Header.Set("X-API-Key", "secret")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRateLimit_AllowsUnderLimit(t *testing.T) {
	handler := appmw.RateLimit(10, time.Minute)(okHandler())
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}
}

func TestRateLimit_BlocksOverLimit(t *testing.T) {
	handler := appmw.RateLimit(3, time.Hour)(okHandler())
	var lastCode int
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "9.9.9.9:5555"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		lastCode = w.Code
	}
	assert.Equal(t, http.StatusTooManyRequests, lastCode)
}

func TestCORS_SetsHeaders(t *testing.T) {
	handler := appmw.CORS(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "X-API-Key")
}

func TestCORS_PreflightReturns204(t *testing.T) {
	handler := appmw.CORS(okHandler())
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}
