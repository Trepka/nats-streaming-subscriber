package main

import (
	"nats-streaming-subscriber/subscriber/server"
)

var port = "8080"

func main() {
	server.StartServer(port)
}
