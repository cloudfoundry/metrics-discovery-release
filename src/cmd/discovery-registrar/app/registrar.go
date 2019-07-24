package app

import (
	"code.cloudfoundry.org/go-loggregator/metrics"
	"time"
)

type ScrapeTargetProvider func() []string

type Publisher interface {
	Publish(queue string, route []byte) error
	Close()
}

type DynamicRegistrar struct {
	publisher       Publisher
	targetProvider  ScrapeTargetProvider
	publishInterval time.Duration
	stop            chan struct{}
	done            chan struct{}
	sent            metrics.Counter
}

type metricsRegistry interface {
	NewCounter(name string, opts ...metrics.MetricOption) metrics.Counter
}

func NewDynamicRegistrar(tp ScrapeTargetProvider, p Publisher, m metricsRegistry, cfg Config) *DynamicRegistrar {
	return &DynamicRegistrar{
		targetProvider:  tp,
		publisher:       p,
		publishInterval: cfg.PublishInterval,
		sent:            m.NewCounter("sent"),
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
		r.publisher.Publish("metrics.endpoints", []byte(t))
		r.sent.Add(float64(1))
	}
}

func (r *DynamicRegistrar) Stop() {
	close(r.stop)
	<-r.done

	r.publisher.Close()
}
