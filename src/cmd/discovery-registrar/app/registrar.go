package app

import (
	"log"
	"time"
)

type Registrar struct {
	routes          []string
	publishInterval time.Duration
	publisher       Publisher
	done            chan struct{}
	stop            chan struct{}
}

type Publisher interface {
	Publish(queue string, route []byte) error
	Close()
}

func NewRegistrar(routes []string, i time.Duration, p Publisher) *Registrar {
	return &Registrar{
		routes:          routes,
		publishInterval: i,
		publisher:       p,
		done:            make(chan struct{}),
		stop:            make(chan struct{}),
	}
}

func (r *Registrar) Start() {
	t := time.NewTicker(r.publishInterval)

	r.publishRoutes()
	for {
		select {
		case <-t.C:
			r.publishRoutes()
		case <-r.stop:
			close(r.done)
			return
		}
	}
}

func (r *Registrar) publishRoutes() {
	for _, route := range r.routes {
		err := r.publisher.Publish("metrics.endpoints", []byte(route))
		if err != nil {
			log.Printf("Error publishing to nats: %s", err)
		}
	}
}

func (r *Registrar) Stop() {
	close(r.stop)
	<-r.done
	r.publisher.Close()
}
