package main

import (
	"code.cloudfoundry.org/metrics-discovery/cmd/dynamic-registrar/app"
	"code.cloudfoundry.org/metrics-discovery/cmd/dynamic-registrar/internal/targetprovider"
	"github.com/nats-io/nats.go"
	"log"
	"os"
	"time"
)

func main() {
	logger := log.New(os.Stderr, "", log.LstdFlags)
	logger.Printf("starting Metric Dynamic Registrar...")
	defer logger.Printf("closing Metric Dynamic Registrar...")

	cfg := app.LoadConfig(logger)

	opts := nats.Options{
		Servers: cfg.NatsHosts,
		PingInterval: 20 * time.Second,
		ClosedCB: closedCB(logger),
		DisconnectedErrCB: disconnectErrHandler(logger),
		ReconnectedCB: reconnectedCB(logger),
	}

	natsConn, err := opts.Connect()
	if err != nil {
		logger.Fatalf("Unable to connect to nats servers: %s", err)
	}

	targetProvider := targetprovider.NewFileProvider(cfg.RoutesGlob, cfg.RouteRefreshInterval)
	go targetProvider.Start()

	registrar := app.NewDynamicRegistrar(targetProvider, natsConn, cfg)
	go registrar.Start()

	defer registrar.Stop()
}

func closedCB(logger *log.Logger) func(conn *nats.Conn) {
	return func(conn *nats.Conn) {
		logger.Println("Nats Connection Closed")
	}
}

func reconnectedCB(logger *log.Logger) func(conn *nats.Conn) {
	return func(conn *nats.Conn) {
		logger.Printf("Reconnected to %s\n", conn.ConnectedUrl())
	}
}

func disconnectErrHandler(logger *log.Logger) func(conn *nats.Conn, err error) {
	return func(conn *nats.Conn, err error) {
		logger.Printf("Nats Error %s\n", err)
	}
}
