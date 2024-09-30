# NATS SSE

HTTP handlers for NATS server side events

Currently, this library uses a single NATS connection. Auth is handled in functions passed into the constructors. 

The example program uses the NATS server's `SubjectCollides` function to verify auth, but this could be whatever method you want.

## Example Auth Function

The auth function can be as simple as needed. Here's an example using the NATS Server module to verify the caller has access to the subject.

```
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
```

This could be a call out to an external service to verify a JWT or as simple as returning true. 

## Handlers

It's simple to add handlers to a server. For pub/sub or requests the only requirement is a `subject` wildcard variable. For KV, you need a variable for `bucket` and a wildcard variable for `key`. KV also supports passing the JetStream domain as a query parameter named `domain`. 

Here's a very simple example: 

```
r := http.NewServeMux()

nc, err := nats.Connect(nats.DefaultURL)
if err != nil {
	log.Fatal(err)
}

handler := natssse.NewSubHandler(nc, authFunc)
pubHandler := natssse.NewPubHandler(nc, authFunc)
reqHandler := natssse.NewReqHandler(nc, authFunc)
kvHandler := natssse.NewKVHandler(nc, authFunc)

r.Handle("GET /subscribe/{subject...}", handler)
r.Handle("POST /publish/{subject...}", pubHandler)
r.Handle("POST /request/{subject...}", reqHandler)
r.Handle("POST /kv/{bucket}/{key...}", kvHandler)
r.Handle("GET /kv/{bucket}/{key...}", kvHandler)
log.Fatal(http.ListenAndServe(":8080", r))
```
