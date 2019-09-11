package app

import (
	"code.cloudfoundry.org/go-loggregator/metrics"
	"code.cloudfoundry.org/metrics-discovery/internal/registry"
	"code.cloudfoundry.org/metrics-discovery/internal/target"
	"gopkg.in/yaml.v2"
	"log"
	"time"
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

	logger *log.Logger
	sent   metrics.Counter
}

type metricsRegistry interface {
	NewCounter(name string, opts ...metrics.MetricOption) metrics.Counter
}

func NewDynamicRegistrar(tp TargetProvider, p Publisher, publishInterval time.Duration, m metricsRegistry, log *log.Logger) *DynamicRegistrar {
	return &DynamicRegistrar{
		targetProvider:  tp,
		publisher:       p,
		publishInterval: publishInterval,
		sent:            m.NewCounter("sent", metrics.WithHelpText("Total number of messages successfully sent to NATs.")),
		stop:            make(chan struct{}),
		done:            make(chan struct{}),
	}
}

func (r *DynamicRegistrar) Start() {
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
	close(r.stop)
	<-r.done

	r.publisher.Close()
}
