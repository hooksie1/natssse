# NATS SSE

HTTP handlers for NATS server side events

Currently, this library uses a single NATS connection. Auth is handled in functions passed into the constructors. 

The example program uses the NATS server's `SubjectCollides` function to verify auth, but this could be whatever method you want.
