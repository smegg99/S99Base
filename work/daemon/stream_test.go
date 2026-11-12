// work/daemon/stream_test.go

package daemon

import (
	"net/http"
	"sync"
	"testing"
)

func testStream(t *testing.T) (http.Handler, func()) {
	t.Helper()
	stopped := make(chan struct{})
	stop := sync.OnceFunc(func() { close(stopped) })
	t.Cleanup(stop)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		select {
		case <-stopped:
		case <-r.Context().Done():
		}
	})
	return handler, stop
}
