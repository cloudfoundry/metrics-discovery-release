package app

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"

	metrics "code.cloudfoundry.org/go-metric-registry"
	"code.cloudfoundry.org/metrics-discovery/internal/registry"
	"code.cloudfoundry.org/metrics-discovery/internal/target"
	"github.com/nats-io/nats.go"
	"gopkg.in/yaml.v2"
)

type Subscriber func(queue string, callback nats.MsgHandler) (*nats.Subscription, error)

type ConfigGenerator struct {
	sync.Mutex
	path                     string
	writeFrequency           time.Duration
	configTTL                time.Duration
	configExpirationInterval time.Duration

	subscriber Subscriber
	done       chan struct{}
	stop       chan struct{}
	delivered  metrics.Counter
	logger     *log.Logger

	timestampedTargets map[string]timestampedTarget
}

type metricsRegistry interface {
	NewCounter(name, helpText string, opts ...metrics.MetricOption) metrics.Counter
}

type timestampedTarget struct {
	scrapeTarget *target.Target
	ts           time.Time
}

func NewConfigGenerator(
	subscriber Subscriber,
	writeFrequency,
	ttl,
	expirationInterval time.Duration,
	path string,
	m metricsRegistry,
	logger *log.Logger,
) *ConfigGenerator {
	configGenerator := &ConfigGenerator{
		timestampedTargets:       make(map[string]timestampedTarget),
		path:                     path,
		writeFrequency:           writeFrequency,
		configTTL:                ttl,
		configExpirationInterval: expirationInterval,

		logger:     logger,
		subscriber: subscriber,
		delivered:  m.NewCounter("delivered", "Total number of messages successfully delivered from NATs."),
		stop:       make(chan struct{}),
		done:       make(chan struct{}),
	}

	// If this doesn't happen synchronously, it could fail when the subscriber is called
	_, err := configGenerator.subscriber(registry.ScrapeTargetQueueName, configGenerator.generate)
	if err != nil {
		configGenerator.logger.Fatalf("failed to subscribe to %s: %s", registry.ScrapeTargetQueueName, err)
	}

	return configGenerator
}

func (cg *ConfigGenerator) Start() {
	expirationTicker := time.NewTicker(cg.configExpirationInterval)
	writeTicker := time.NewTicker(cg.writeFrequency)
	for {
		select {
		case <-writeTicker.C:
			cg.writeConfigToFile()
		case <-expirationTicker.C:
			cg.expireScrapeConfigs()
		case <-cg.stop:
			close(cg.done)
			return
		}
	}
}

func (cg *ConfigGenerator) Stop() {
	close(cg.stop)
	<-cg.done
}

func (cg *ConfigGenerator) generate(message *nats.Msg) {
	cg.delivered.Add(float64(1))

	scrapeTarget, ok := cg.unmarshalScrapeTarget(message)
	if !ok {
		return
	}

	cg.addTarget(scrapeTarget)
}

func (cg *ConfigGenerator) unmarshalScrapeTarget(message *nats.Msg) (*target.Target, bool) {
	var t target.Target
	err := yaml.Unmarshal(message.Data, &t)
	if err != nil {
		cg.logger.Printf("failed to unmarshal message data: %s\n", err)
		return nil, false
	}

	return &t, true
}

func (cg *ConfigGenerator) addTarget(scrapeTarget *target.Target) {
	cg.Lock()
	defer cg.Unlock()

	cg.timestampedTargets[scrapeTarget.Source] = timestampedTarget{
		scrapeTarget: scrapeTarget,
		ts:           time.Now(),
	}
}

func (cg *ConfigGenerator) writeConfigToFile() {
	targets := cg.copyTargets()

	data, err := json.Marshal(targets)
	if err != nil {
		cg.logger.Printf("failed to marshal scrape configs: %s\n", err)
		return
	}

	err = ioutil.WriteFile(cg.path, data, os.ModePerm)
	if err != nil {
		cg.logger.Printf("failed to write scrape config file: %s\n", err)
	}
}

func (cg *ConfigGenerator) copyTargets() []*target.Target {
	var targets []*target.Target

	cg.Lock()
	defer cg.Unlock()
	for _, cfg := range cg.timestampedTargets {
		targets = append(targets, cfg.scrapeTarget)
	}

	return targets
}

func (cg *ConfigGenerator) expireScrapeConfigs() {
	cg.Lock()
	defer cg.Unlock()

	for k, scrapeConfig := range cg.timestampedTargets {
		if time.Since(scrapeConfig.ts) >= cg.configTTL {
			delete(cg.timestampedTargets, k)
		}
	}
}
