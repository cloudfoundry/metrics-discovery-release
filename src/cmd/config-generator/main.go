package main

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"os"
	"time"

	metrics "code.cloudfoundry.org/go-metric-registry"
	"code.cloudfoundry.org/metrics-discovery/cmd/config-generator/app"
	"github.com/nats-io/nats.go"
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
		TLSConfig:         getTLSConfig(config),
	}

	natsConn, err := opts.Connect()
	if err != nil {
		logger.Fatalf("Unable to connect to nats servers: %s", err)
	}

	m := metrics.NewRegistry(logger,
		metrics.WithTLSServer(config.MetricsPort, config.MetricsCertPath, config.MetricsKeyPath, config.MetricsCAPath),
	)

	generator := app.NewConfigGenerator(
		natsConn.Subscribe,
		config.WriteFrequency,
		config.ConfigTimeToLive,
		config.ConfigExpirationInterval,
		config.ScrapeConfigFilePath,
		m,
		logger,
	)

	generator.Start()
	defer generator.Stop()
}

func getTLSConfig(cfg app.Config) *tls.Config {
	if cfg.NatsCAPath != "" {
		caCert, err := ioutil.ReadFile(cfg.NatsCAPath)
		if err != nil {
			log.Fatal(err)
		}

		caCertPool := x509.NewCertPool()

		if !caCertPool.AppendCertsFromPEM(caCert) {
			log.Fatalf("Failed to load CA certificate from file %s", cfg.NatsCAPath)
		}

		return &tls.Config{
			RootCAs: caCertPool,
		}
	}
	return nil
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
