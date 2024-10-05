package natssse

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
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
	ch      chan nats.Msg
}

type Response struct {
	Subject string `json:"subject"`
	Data    []byte `json:"data"`
}

type sseFunc func(context.Context, http.Flusher, options)

// AuthFunc does external authentication for the HTTP connections
type AuthFunc func(authorization string, subject string) bool

// replacer replaces the slash in a URL with a period to convert it to a NATS subject
func replacer(r *http.Request, name string) string {
	trimmed := r.PathValue(name)
	return strings.ReplaceAll(trimmed, "/", ".")
}

// newSSEHandler is a wrapper around an sseFunc. It handles creating the goroutines around the keepalive and context cancellations
func newSSEHandler(w http.ResponseWriter, r *http.Request, nc NatsContext, f sseFunc) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	subject := replacer(r, "subject")

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

	ch := make(chan nats.Msg)

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
func ping(ctx context.Context, ch chan<- nats.Msg) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			time.Sleep(30 * time.Second)
			msg := nats.Msg{
				Subject: "natssse.system",
				Data:    []byte(`{"system": "ping"}`),
			}
			ch <- msg
		}
	}
}

func writeAndFlushResponse(w http.ResponseWriter, flusher http.Flusher, msg nats.Msg) {

	if err := writeResponse(w, msg); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), 500)
		return
	}

	flusher.Flush()
}

func writeResponse(w http.ResponseWriter, msg nats.Msg) error {
	for k, v := range msg.Header {
		if strings.ToLower(k) == "content-length" {
			continue
		}
		for _, h := range v {
			w.Header().Add(k, h)
		}
	}

	resp := Response{
		Subject: msg.Subject,
		Data:    msg.Data,
	}

	return json.NewEncoder(w).Encode(resp)
}
