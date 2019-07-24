package main

import (
	"code.cloudfoundry.org/metrics-discovery/cmd/config-generator/app"
	"github.com/nats-io/nats.go"
	"log"
	"os"
	"time"
)

func main() {
	logger := log.New(os.Stderr, "", log.LstdFlags)
	logger.Printf("starting Scrape Config Generator...")
	defer log.Printf("closing Scrape Config Generator...")

	config := app.LoadConfig(logger)

	opts := nats.Options{
		Servers:           config.NatsHosts,
		PingInterval:      20 * time.Second,
		AllowReconnect:    true,
		MaxReconnect:      -1,
		ReconnectWait:     100 * time.Millisecond,
		ClosedCB:          closedCB(logger),
		DisconnectedErrCB: disconnectErrHandler(logger),
		ReconnectedCB:     reconnectedCB(logger),
	}

	natsConn, err := opts.Connect()
	if err != nil {
		logger.Fatalf("Unable to connect to nats servers: %s", err)
	}

	generator := app.NewConfigGenerator(
		natsConn.Subscribe,
		config.ConfigTimeToLive,
		config.ConfigExpirationInterval,
		config.ScrapeConfigFilePath,
		logger,
	)

	generator.Start()
	defer generator.Stop()
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
