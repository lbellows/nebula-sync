package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lovelaze/nebula-sync/internal/sync"
)

// The cron goroutine writes state while the health handler reads it. Run with
// -race to catch unguarded access.
func TestHealthHandler_concurrentWithSyncOutcomes(t *testing.T) {
	state := sync.NewState()
	server := NewServer(state)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for range 2000 {
			state.OnSuccess()
		}
	}()

	for range 2000 {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		resp := httptest.NewRecorder()
		server.router.ServeHTTP(resp, req)
	}

	<-done
}
