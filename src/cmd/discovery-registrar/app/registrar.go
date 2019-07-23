package app

import (
	"time"
)

type ScrapeTargetProvider func()[]string

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
}

func NewDynamicRegistrar(tp ScrapeTargetProvider, p Publisher, cfg Config) *DynamicRegistrar {
	return &DynamicRegistrar{
		targetProvider:  tp,
		publisher:       p,
		publishInterval: cfg.PublishInterval,
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
	}
}

func (r *DynamicRegistrar) Stop() {
	close(r.stop)
	<-r.done

	r.publisher.Close()
}
