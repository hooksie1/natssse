package natssse

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/nats-io/nats.go"
)

// NatsContext holds a NATS connection and an authorization function
type NatsContext struct {
	Conn *nats.Conn
	Auth AuthFunc
}

type options struct {
	writer  http.ResponseWriter
	cancel  context.CancelFunc
	subject string
	nc      NatsContext
	ch      chan string
}

type sseFunc func(context.Context, http.Flusher, options)

// AuthFunc does external authentication for the HTTP connections
type AuthFunc func(authorization string, subject string) bool

// newSSEHandler is a wrapper around an sseFunc. It handles creating the goroutines around the keepalive and context cancellations
func newSSEHandler(w http.ResponseWriter, r *http.Request, nc NatsContext, f sseFunc) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	subject := r.URL.Query().Get("subject")

	ok := nc.Auth(r.Header.Get("Authorization"), subject)
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, http.StatusText(http.StatusInternalServerError), 500)
		return
	}

	ch := make(chan string)

	go func() {
		<-r.Context().Done()
		return
	}()

	go ping(ctx, ch)

	opts := options{
		cancel:  cancel,
		subject: subject,
		nc:      nc,
		ch:      ch,
		writer:  w,
	}

	f(ctx, flusher, opts)
}

// ping handles sending server side keep alive events
func ping(ctx context.Context, ch chan<- string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			time.Sleep(30 * time.Second)
			ch <- fmt.Sprint(`{"system": "ping"}`)
		}
	}
}
