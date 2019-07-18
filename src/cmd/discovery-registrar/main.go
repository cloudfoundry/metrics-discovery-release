package main

import (
	"code.cloudfoundry.org/metrics-discovery/cmd/discovery-registrar/app"
	"github.com/nats-io/nats.go"
	"log"
	"os"
	"time"
)

func main() {
	log := log.New(os.Stderr, "", log.LstdFlags)
	log.Printf("starting Metric Discovery Registrar...")
	defer log.Printf("closing Metric Discovery Registrar...")

	cfg := app.LoadConfig(log)

	opts := nats.Options{
		Servers: cfg.NatsHosts,
		PingInterval: 20 * time.Second,
		ClosedCB: closedCB(log),
		DisconnectedErrCB: disconnectErrHandler(log),
		ReconnectedCB: reconnectedCB(log),
	}

	nc, err := opts.Connect()
	if err != nil {
		log.Fatalf("Unable to connect to nats servers: %s", err)
	}

	registrar := app.NewRegistrar(cfg.Routes, cfg.PublishInterval, nc)
	registrar.Start()
	defer registrar.Stop()
}

func closedCB(log *log.Logger) func(conn *nats.Conn) {
	return func(conn *nats.Conn) {
		log.Println("Nats Connection Closed")
	}
}

func reconnectedCB(log *log.Logger) func(conn *nats.Conn) {
	return func(conn *nats.Conn) {
		log.Printf("Reconnected to %s\n", conn.ConnectedUrl())
	}
}

func disconnectErrHandler(log *log.Logger) func(conn *nats.Conn, err error) {
	return func(conn *nats.Conn, err error) {
		log.Printf("Nats Error %s\n", err)
	}
}