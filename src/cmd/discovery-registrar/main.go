package main

import (
	"code.cloudfoundry.org/go-loggregator/metrics"
	"code.cloudfoundry.org/metrics-discovery/cmd/discovery-registrar/app"
	"code.cloudfoundry.org/metrics-discovery/cmd/discovery-registrar/internal/targetprovider"
	"github.com/nats-io/nats.go"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	logger := log.New(os.Stderr, "", log.LstdFlags)
	logger.Printf("starting Metric Discovery Registrar...")
	defer logger.Printf("closing Metric Discovery Registrar...")

	cfg := app.LoadConfig(logger)

	opts := nats.Options{
		Servers:      cfg.NatsHosts,
		PingInterval: 20 * time.Second,

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

	routeProvider := func() []string { return cfg.Routes }
	if cfg.Routes == nil {
		targetProvider := targetprovider.NewFileProvider(cfg.RoutesGlob, cfg.RouteRefreshInterval)
		go targetProvider.Start()

		routeProvider = targetProvider.GetTargets
	}

	m := metrics.NewRegistry(logger,
		metrics.WithDefaultTags(map[string]string{
			"origin":    "loggregator.config_generator",
			"source_id": "config_generator",
		}),
	)

	registrar := app.NewDynamicRegistrar(routeProvider, natsConn, m, cfg)
	go registrar.Start()
	defer registrar.Stop()

	waitForTermination()
}

func waitForTermination() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
	<-c
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
