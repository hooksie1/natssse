package natssse

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/nats-io/nats.go"
)

const (
	PUT Method = "PUT"
	GET Method = "GET"
)

type Method string

func NewKVHandler(conn *nats.Conn, authFunc AuthFunc) http.HandlerFunc {
	n := NatsContext{
		Conn: conn,
		Auth: authFunc,
	}

	return func(w http.ResponseWriter, r *http.Request) {
		err := newKVHandler(w, r, n)
		if err == nil {
			return
		}

		e, ok := err.(ClientError)
		if !ok {
			log.Println(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(e.Code())
		w.Write(e.Body())
		return
	}
}

func newKVHandler(w http.ResponseWriter, r *http.Request, nc NatsContext) error {
	bucket := r.URL.Query().Get("bucket")
	key := r.URL.Query().Get("key")
	if key == "" || bucket == "" {
		return NewClientError(fmt.Errorf("key and bucket must be defined"), http.StatusBadRequest)
	}

	ok := nc.Auth(r.Header.Get("Authorization"), key)
	if !ok {
		return NewClientError(fmt.Errorf(http.StatusText(http.StatusUnauthorized)), http.StatusUnauthorized)
	}

	js, err := nc.Conn.JetStream()
	if err != nil {
		return err
	}

	kv, err := js.KeyValue(bucket)
	if err != nil && errors.Is(err, nats.ErrKeyNotFound) {
		return NewClientError(fmt.Errorf(http.StatusText(http.StatusNotFound)), http.StatusNotFound)
	}
	if err != nil {
		return err
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}

	switch r.Method {
	case http.MethodPut:
		err := putKV(key, kv, body)
		if err != nil {
			return err
		}
		w.WriteHeader(204)
	case http.MethodGet:
		data, err := getKV(key, kv)
		if err != nil {
			return err
		}
		w.Write(data)
	case http.MethodDelete:
		err := deleteKV(key, kv)
		if err != nil {
			return err

		}
	default:
		return NewClientError(fmt.Errorf(http.StatusText(http.StatusMethodNotAllowed)), http.StatusMethodNotAllowed)
	}

	return nil
}

func getKV(key string, kv nats.KeyValue) ([]byte, error) {
	resp, err := kv.Get(key)
	if err != nil && errors.Is(err, nats.ErrKeyNotFound) {
		return nil, NewClientError(fmt.Errorf(http.StatusText(http.StatusNotFound)), http.StatusNotFound)
	}
	if err != nil {
		return nil, err
	}

	return resp.Value(), nil
}

func putKV(key string, kv nats.KeyValue, data []byte) error {
	_, err := kv.Put(key, data)
	if err != nil && errors.Is(err, nats.ErrKeyNotFound) {
		return NewClientError(fmt.Errorf(http.StatusText(http.StatusNotFound)), http.StatusNotFound)
	}
	if err != nil {
		return err
	}
	return nil
}

func deleteKV(key string, kv nats.KeyValue) error {
	err := kv.Delete(key)
	if err != nil && errors.Is(err, nats.ErrKeyNotFound) {
		return NewClientError(fmt.Errorf(http.StatusText(http.StatusNotFound)), http.StatusNotFound)
	}
	if err != nil {
		return err
	}

	return nil
}
