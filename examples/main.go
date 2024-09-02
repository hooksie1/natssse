package main

import (
	"log"
	"net/http"

	"github.com/hooksie1/natssse"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

var permissions = map[string]string{
	"john": "users.john.*",
	"pete": "users.pete.>",
}

func authFunc(authorization, subject string) bool {

	allowed, ok := permissions[authorization]
	if !ok {
		return false
	}

	return server.SubjectsCollide(allowed, subject)
}

func main() {
	r := http.NewServeMux()

	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatal(err)
	}

	handler := natssse.NewSubHandler(nc, authFunc)
	pubHandler := natssse.NewPubHandler(nc, authFunc)
	reqHandler := natssse.NewReqHandler(nc, authFunc)

	r.Handle("GET /subscribe", handler)
	r.Handle("POST /publish", pubHandler)
	r.Handle("POST /request", reqHandler)
	log.Fatal(http.ListenAndServe(":8080", r))
}
