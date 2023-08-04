package app

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof" // nolint:gosec
	"time"

	metrics "code.cloudfoundry.org/go-metric-registry"
	"code.cloudfoundry.org/metrics-discovery/internal/registry"
	"code.cloudfoundry.org/metrics-discovery/internal/target"
	"gopkg.in/yaml.v3"
)

type TargetProvider func() []*target.Target

type Publisher interface {
	Publish(queue string, route []byte) error
	Close()
}

type DynamicRegistrar struct {
	publisher       Publisher
	targetProvider  TargetProvider
	publishInterval time.Duration

	stop chan struct{}
	done chan struct{}

	logger      *log.Logger
	sent        metrics.Counter
	metrics     metricsRegistry
	pprofServer *http.Server
}

type metricsRegistry interface {
	NewCounter(name, helpText string, opts ...metrics.MetricOption) metrics.Counter
	RegisterDebugMetrics()
}

func NewDynamicRegistrar(tp TargetProvider, p Publisher, publishInterval time.Duration, m metricsRegistry, log *log.Logger) *DynamicRegistrar {
	return &DynamicRegistrar{
		targetProvider:  tp,
		publisher:       p,
		publishInterval: publishInterval,
		sent:            m.NewCounter("sent", "Total number of messages successfully sent to NATs."),
		metrics:         m,
		stop:            make(chan struct{}),
		done:            make(chan struct{}),
	}
}

func (r *DynamicRegistrar) Start(debugMetrics bool, pprofPort uint16) {
	if debugMetrics {
		r.metrics.RegisterDebugMetrics()
		r.pprofServer = &http.Server{
			Addr:              fmt.Sprintf("127.0.0.1:%d", pprofPort),
			Handler:           http.DefaultServeMux,
			ReadHeaderTimeout: 2 * time.Second,
		}
		go func() {
			err := r.pprofServer.ListenAndServe()
			if err != http.ErrServerClosed {
				log.Fatalf("pprof error: %s", err)
			}
		}()
	}
	ticker := time.NewTicker(r.publishInterval)

	r.publishTargets()

	for {
		select {
		case <-ticker.C:
			r.publishTargets()
		case <-r.stop:
			close(r.done)
			return
		}
	}
}

func (r *DynamicRegistrar) publishTargets() {
	targets := r.targetProvider()
	for _, t := range targets {
		bytes, err := yaml.Marshal(t)
		if err != nil {
			r.logger.Printf("unable to marshal target(%s): %s\n", t.Source, err)
			continue
		}

		err = r.publisher.Publish(registry.ScrapeTargetQueueName, bytes)
		if err != nil {
			r.logger.Printf("unable to publish target(%s): %s\n", t.Source, err)
			continue
		}
		r.sent.Add(float64(1))
	}
}

func (r *DynamicRegistrar) Stop() {
	if r.pprofServer != nil {
		r.pprofServer.Close()
	}
	close(r.stop)
	<-r.done

	r.publisher.Close()
}
