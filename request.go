package natssse

import (
	"io"
	"net/http"
	"time"

	"github.com/nats-io/nats.go"
)

func NewReqHandler(conn *nats.Conn, authFunc AuthFunc) http.HandlerFunc {
	n := NatsContext{
		Conn: conn,
		Auth: authFunc,
	}

	return func(w http.ResponseWriter, r *http.Request) {
		newReqHandler(w, r, n)
	}
}

func newReqHandler(w http.ResponseWriter, r *http.Request, nc NatsContext) {
	subject := r.URL.Query().Get("subject")

	ok := nc.Auth(r.Header.Get("Authorization"), subject)
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return
	}

	msg := &nats.Msg{
		Subject: subject,
		Header:  nats.Header(r.Header.Clone()),
		Data:    body,
	}

	resp, err := nc.Conn.RequestMsg(msg, 3*time.Second)
	if err != nil && err == nats.ErrNoResponders {
		http.Error(w, http.StatusText(http.StatusNotFound), 404)
		return
	}
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), 500)
		return
	}

	w.Write(resp.Data)
}
