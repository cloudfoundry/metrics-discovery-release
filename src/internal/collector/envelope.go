package collector

import (
	b64 "encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"code.cloudfoundry.org/go-loggregator/v9/rpc/loggregator_v2"
	metrics "code.cloudfoundry.org/go-metric-registry"
	v2 "code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress/v2"
	"github.com/prometheus/client_golang/prometheus"
)

const help = "Metrics Agent collected metric"

var (
	buckets                  = []float64{0.01, 0.2, 1.0, 15.0, 60.0}
	invalidNameRegex         = regexp.MustCompile(`[^a-zA-Z0-9_:]`)
	invalidTagCharacterRegex = regexp.MustCompile(`[^a-zA-Z0-9_]`)
)

type sourceIDBucket struct {
	lastUpdate time.Time
	metrics    map[string]metricWithExpiry
}

type metricWithExpiry struct {
	lastUpdate time.Time
	metric     prometheus.Metric
}

func newSourceIDBucket() *sourceIDBucket {
	return &sourceIDBucket{
		lastUpdate: time.Now(),
		metrics:    map[string]metricWithExpiry{},
	}
}

func (b *sourceIDBucket) addMetric(id string, metric prometheus.Metric) {
	b.metrics[id] = metricWithExpiry{metric: metric, lastUpdate: time.Now()}
	b.lastUpdate = time.Now()
}

type EnvelopeCollector struct {
	sync.RWMutex

	metricBuckets map[string]*sourceIDBucket

	sourceIDTTL                time.Duration
	sourceIDExpirationInterval time.Duration
	defaultTags                map[string]string
	metrics                    debugMetrics
}

type EnvelopeCollectorOption func(*EnvelopeCollector)

type debugMetrics interface {
	NewCounter(name, helpText string, opts ...metrics.MetricOption) metrics.Counter
}

func NewEnvelopeCollector(m debugMetrics, opts ...EnvelopeCollectorOption) *EnvelopeCollector {
	c := &EnvelopeCollector{
		metricBuckets:              map[string]*sourceIDBucket{},
		sourceIDTTL:                time.Hour,
		sourceIDExpirationInterval: time.Minute,
		metrics:                    m,
	}

	for _, opt := range opts {
		opt(c)
	}

	go c.expireMetrics()

	return c
}

func WithSourceIDExpiration(ttl, expirationInterval time.Duration) EnvelopeCollectorOption {
	return func(c *EnvelopeCollector) {
		c.sourceIDTTL = ttl
		c.sourceIDExpirationInterval = expirationInterval
	}
}

func WithDefaultTags(tags map[string]string) EnvelopeCollectorOption {
	return func(c *EnvelopeCollector) {
		c.defaultTags = tags
	}
}

func (c *EnvelopeCollector) expireMetrics() {
	expirationTicker := time.NewTicker(c.sourceIDExpirationInterval)
	for range expirationTicker.C {
		tooOld := time.Now().Add(-c.sourceIDTTL)

		c.Lock()
		for sourceID, bucket := range c.metricBuckets {
			if bucket.lastUpdate.Before(tooOld) {
				delete(c.metricBuckets, sourceID)
			}
			for id, metric := range bucket.metrics {
				if metric.lastUpdate.Before(tooOld) {
					delete(bucket.metrics, id)
				}
			}
		}
		c.Unlock()
	}
}

// Describe implements prometheus.Collector
// Unimplemented because metric descriptors should not be checked against other collectors
func (c *EnvelopeCollector) Describe(ch chan<- *prometheus.Desc) {}

// Collect implements prometheus.Collector
func (c *EnvelopeCollector) Collect(ch chan<- prometheus.Metric) {
	c.RLock()
	defer c.RUnlock()

	for _, bucket := range c.metricBuckets {
		for _, metric := range bucket.metrics {
			ch <- metric.metric
		}
	}
}

// Write implements v2.Writer
func (c *EnvelopeCollector) Write(env *loggregator_v2.Envelope) error {
	metrics, err := c.convertEnvelope(env)
	if err != nil {
		return err
	}

	c.Lock()
	defer c.Unlock()
	for id, metric := range metrics {
		c.getOrCreateBucket(env.GetSourceId()).addMetric(id, metric)
	}

	return nil
}

func (c *EnvelopeCollector) getOrCreateBucket(sourceID string) *sourceIDBucket {
	bucket, ok := c.metricBuckets[sourceID]
	if ok {
		return bucket
	}

	bucket = newSourceIDBucket()
	c.metricBuckets[sourceID] = bucket
	return bucket
}

func (c *EnvelopeCollector) convertEnvelope(env *loggregator_v2.Envelope) (map[string]prometheus.Metric, error) {
	switch env.GetMessage().(type) {
	case *loggregator_v2.Envelope_Counter:
		id, metric, err := c.convertCounter(env)
		return map[string]prometheus.Metric{id: metric}, err
	case *loggregator_v2.Envelope_Gauge:
		return c.convertGaugeEnvelope(env)
	case *loggregator_v2.Envelope_Timer:
		id, metric, err := c.convertTimer(env)
		return map[string]prometheus.Metric{id: metric}, err
	default:
		return nil, nil
	}
}

func (c *EnvelopeCollector) convertCounter(env *loggregator_v2.Envelope) (metricID string, metric prometheus.Metric, err error) {
	originalName := env.GetCounter().GetName()
	name, modified := sanitizeName(originalName)
	if modified {
		c.incrementCounter("modified_tags", env.GetSourceId())
	}

	labelNames, labelValues := c.convertTags(env)
	labelNames, labelValues = addLoggregatorNameTag(labelNames, labelValues, originalName)

	desc := prometheus.NewDesc(name, help, labelNames, nil)
	metric, err = prometheus.NewConstMetric(desc, prometheus.CounterValue, float64(env.GetCounter().GetTotal()), labelValues...)
	if err != nil {
		return "", nil, err
	}

	return buildMetricID(name, labelNames, labelValues), metric, nil
}

func (c *EnvelopeCollector) convertGaugeEnvelope(env *loggregator_v2.Envelope) (map[string]prometheus.Metric, error) {
	labelNames, labelValues := c.convertTags(env)

	promMetrics := map[string]prometheus.Metric{}

	for name, metric := range env.GetGauge().GetMetrics() {
		sanitizedName, modified := sanitizeName(name)
		if modified {
			c.incrementCounter("modified_tags", env.GetSourceId())
		}

		id, metric, err := convertGaugeValue(name, sanitizedName, metric, labelNames, labelValues)
		if err != nil {
			return nil, fmt.Errorf("invalid metric: %s", err)
		}
		promMetrics[id] = metric
	}

	return promMetrics, nil
}

func convertGaugeValue(originalName, sanitizedName string, gaugeValue *loggregator_v2.GaugeValue, envelopeLabelNames, envelopeLabelValues []string) (string, prometheus.Metric, error) {
	labelNames, labelValues := gaugeLabels(gaugeValue, envelopeLabelNames, envelopeLabelValues)
	labelNames, labelValues = addLoggregatorNameTag(labelNames, labelValues, originalName)
	desc := prometheus.NewDesc(sanitizedName, help, labelNames, nil)
	metric, err := prometheus.NewConstMetric(desc, prometheus.GaugeValue, gaugeValue.Value, labelValues...)
	if err != nil {
		return "", nil, err
	}

	return buildMetricID(sanitizedName, envelopeLabelNames, envelopeLabelValues), metric, nil
}

func buildMetricID(name string, envelopeLabelNames, envelopeLabelValues []string) string {
	labelTags := labelTags(envelopeLabelNames, envelopeLabelValues)

	return name + v2.HashTags(labelTags)
}

func labelTags(envelopeLabelNames, envelopeLabelValues []string) map[string]string {
	labelTags := map[string]string{}
	for i, labelName := range envelopeLabelNames {
		labelTags[labelName] = envelopeLabelValues[i]
	}
	return labelTags
}

func gaugeLabels(metric *loggregator_v2.GaugeValue, envelopeLabelNames, envelopeLabelValues []string) ([]string, []string) {
	if metric.Unit == "" {
		return envelopeLabelNames, envelopeLabelValues
	}

	return append(envelopeLabelNames, "unit"), append(envelopeLabelValues, metric.Unit)
}

func (c *EnvelopeCollector) convertTimer(env *loggregator_v2.Envelope) (metricID string, metric prometheus.Metric, err error) {
	timer := env.GetTimer()
	name := timer.GetName() + "_seconds"
	name, modified := sanitizeName(name)
	if modified {
		c.incrementCounter("modified_tags", env.GetSourceId())
	}

	labelNames, labelValues := c.convertTags(env)
	labelNames, labelValues = addLoggregatorNameTag(labelNames, labelValues, timer.GetName())
	id := buildMetricID(name, labelNames, labelValues)

	c.Lock()
	bucket := c.getOrCreateBucket(env.GetSourceId())
	metricWithExpiry, ok := bucket.metrics[id]
	c.Unlock()

	if ok {
		metric = metricWithExpiry.metric
	} else {
		metric = prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:        name,
			Help:        help,
			Buckets:     buckets,
			ConstLabels: labelTags(labelNames, labelValues),
		})
	}
	metric.(prometheus.Histogram).Observe(durationInSeconds(timer))

	return id, metric, nil
}

func durationInSeconds(timer *loggregator_v2.Timer) float64 {
	return float64(timer.GetStop()-timer.GetStart()) / float64(time.Second)
}

func (c *EnvelopeCollector) convertTags(env *loggregator_v2.Envelope) ([]string, []string) {
	labelNames, labelValues := c.toLabels(env.SourceId, c.allTags(env))

	if _, ok := env.GetTags()["source_id"]; !ok {
		labelNames = append(labelNames, "source_id")
		labelValues = append(labelValues, env.GetSourceId())
	}

	if _, ok := env.GetTags()["instance_id"]; !ok && env.GetInstanceId() != "" {
		labelNames = append(labelNames, "instance_id")
		labelValues = append(labelValues, env.GetInstanceId())
	}

	return labelNames, labelValues
}

func (c *EnvelopeCollector) allTags(env *loggregator_v2.Envelope) map[string]string {
	allTags := make(map[string]string)
	for k, v := range env.GetTags() {
		allTags[k] = v
	}

	for k, v := range c.defaultTags {
		_, exists := allTags[k]
		if exists {
			continue
		}
		allTags[k] = v
	}
	return allTags
}

func (c *EnvelopeCollector) toLabels(sourceID string, tags map[string]string) ([]string, []string) {
	var labelNames, labelValues []string
	for name, value := range tags {
		if invalidTag(name, value) {
			c.incrementCounter("invalid_metric_label", sourceID)
			continue
		}

		name, modified := sanitizeTagName(name)
		if modified {
			c.incrementCounter("modified_tags", sourceID)
		}

		labelNames = append(labelNames, name)
		labelValues = append(labelValues, value)
	}
	return labelNames, labelValues
}

func (c *EnvelopeCollector) incrementCounter(metricName, originatingSourceID string) {
	c.metrics.NewCounter(
		metricName,
		fmt.Sprintf("Total number of %s for the originating source id from the envelope", metricName),
		metrics.WithMetricLabels(map[string]string{"originating_source_id": originatingSourceID}),
	).Add(1)
}

func addLoggregatorNameTag(labelNames, labelValues []string, name string) ([]string, []string) {
	name = b64.StdEncoding.EncodeToString([]byte(name))
	return append(labelNames, "loggregator_name"), append(labelValues, name)
}

func sanitizeTagName(name string) (string, bool) {
	sanitized := invalidTagCharacterRegex.ReplaceAllString(name, "_")
	return sanitized, sanitized != name
}

func sanitizeName(name string) (string, bool) {
	sanitized := invalidNameRegex.ReplaceAllString(name, "_")
	return sanitized, sanitized != name
}

func invalidTag(name, value string) bool {
	return strings.HasPrefix(name, "__") || value == ""
}
