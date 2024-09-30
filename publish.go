package natssse

import (
	"io"
	"net/http"

	"github.com/nats-io/nats.go"
)

// NewPubHandler creates a handler that can publish on a subject
func NewPubHandler(conn *nats.Conn, authFunc AuthFunc) http.HandlerFunc {
	n := NatsContext{
		Conn: conn,
		Auth: authFunc,
	}

	return func(w http.ResponseWriter, r *http.Request) {
		newPubHandler(w, r, n)
	}
}

// newPubHandler is a handler that will publish on a subject passed as a query parameter
func newPubHandler(w http.ResponseWriter, r *http.Request, nc NatsContext) {
	subject := replacer(r, "subject")

	ok := nc.Auth(r.Header.Get("Authorization"), subject)
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return
	}

	if err := nc.Conn.Publish(subject, body); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), 500)
		return
	}

	return
}
