package natssse

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/nats-io/nats.go"
)

// NewSubHandler creates a handler that does server side event subscribing
func NewSubHandler(conn *nats.Conn, authFunc AuthFunc) http.HandlerFunc {
	n := NatsContext{
		Conn: conn,
		Auth: authFunc,
	}

	return func(w http.ResponseWriter, r *http.Request) {
		newSSEHandler(w, r, n, Subscribe)
	}
}

// Subscribe wraps handleSubscription and handles flushing the writer.
func Subscribe(ctx context.Context, flusher http.Flusher, opts options) {
	go handleSubscription(ctx, opts)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			data := fmt.Sprintf("%s\n", <-opts.ch)
			_, err := fmt.Fprintf(opts.writer, data)
			if err != nil {
				http.Error(opts.writer, err.Error(), 500)
				return
			}

			flusher.Flush()
		}
	}
}

// handleSubscription creates the NATS subscription and iterates over the messages
func handleSubscription(ctx context.Context, opts options) {
	sub, err := opts.nc.Conn.SubscribeSync(opts.subject)
	if err != nil {
		opts.ch <- err.Error()
		opts.cancel()
		return
	}
	defer sub.Unsubscribe()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			msg, err := sub.NextMsg(10 * time.Second)
			if err != nil && errors.Is(err, nats.ErrTimeout) {
				continue
			}
			if err != nil {
				opts.ch <- err.Error()
				continue
			}
			opts.ch <- string(msg.Data)

		}
	}
}
