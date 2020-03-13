package collector_test

import (
	b64 "encoding/base64"
	"fmt"
	"time"

	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	"code.cloudfoundry.org/go-metric-registry/testhelpers"
	"code.cloudfoundry.org/metrics-discovery/internal/collector"
	"github.com/gogo/protobuf/proto"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

var _ = Describe("EnvelopeCollector", func() {
	It("collects all received metrics", func() {
		envelopeCollector := collector.NewEnvelopeCollector(testhelpers.NewMetricsRegistry())
		Expect(envelopeCollector.Write(totalCounter("metric1", 22))).To(Succeed())
		Expect(envelopeCollector.Write(totalCounter("metric2", 22))).To(Succeed())

		metrics := collectMetrics(envelopeCollector)
		Expect(metrics).To(HaveLen(2))
		Expect([]prometheus.Metric{<-metrics, <-metrics}).To(ConsistOf(
			haveName("metric1"),
			haveName("metric2"),
		))

		Expect(collectMetrics(envelopeCollector)).To(receiveInAnyOrder(
			haveName("metric1"),
			haveName("metric2"),
		))
	})

	It("converts metrics with invalid characters", func() {
		spyMetricsRegistry := testhelpers.NewMetricsRegistry()
		envelopeCollector := collector.NewEnvelopeCollector(spyMetricsRegistry)

		Expect(envelopeCollector.Write(gauge(map[string]float64{
			"gauge1.wrong.name": 11,
			"gauge2/also-wrong": 22,
		}))).To(Succeed())
		Expect(envelopeCollector.Write(totalCounter("counter.wrong.name", 11))).To(Succeed())
		Expect(envelopeCollector.Write(timer("timer.wrong.name", int64(time.Millisecond), int64(2*time.Millisecond)))).To(Succeed())

		Expect(collectMetrics(envelopeCollector)).To(receiveInAnyOrder(
			And(
				haveName("gauge1_wrong_name"),
				gaugeWithValue(11),
			),
			And(
				haveName("gauge2_also_wrong"),
				gaugeWithValue(22),
			),
			And(
				haveName("counter_wrong_name"),
				counterWithValue(11),
			),
			And(
				haveName("timer_wrong_name_seconds"),
				histogramWithCount(1),
				histogramWithSum(float64(time.Millisecond)/float64(time.Second)),
				histogramWithBuckets(0.01, 0.2, 1.0, 15.0, 60.0),
			),
		))

		Expect(spyMetricsRegistry.GetMetricValue("modified_tags", map[string]string{"originating_source_id": "some-source-id"})).To(Equal(4.0))
	})

	Context("envelope types", func() {
		It("collects counters from the provider", func() {
			envelopeCollector := collector.NewEnvelopeCollector(testhelpers.NewMetricsRegistry())
			Expect(envelopeCollector.Write(totalCounter("some_total_counter", 22))).To(Succeed())

			Expect(collectMetrics(envelopeCollector)).To(Receive(And(
				haveName("some_total_counter"),
				counterWithValue(22),
			)))

			Expect(envelopeCollector.Write(totalCounter("some_total_counter", 37))).To(Succeed())
			Expect(collectMetrics(envelopeCollector)).To(Receive(And(
				haveName("some_total_counter"),
				counterWithValue(37),
			)))
		})

		It("collects gauges from the provider", func() {
			envelopeCollector := collector.NewEnvelopeCollector(testhelpers.NewMetricsRegistry())
			Expect(envelopeCollector.Write(gauge(map[string]float64{
				"gauge1": 11,
				"gauge2": 22,
			}))).To(Succeed())

			Expect(collectMetrics(envelopeCollector)).To(receiveInAnyOrder(
				And(
					haveName("gauge1"),
					gaugeWithValue(11),
				),
				And(
					haveName("gauge2"),
					gaugeWithValue(22),
				),
			))

			Expect(envelopeCollector.Write(gauge(map[string]float64{
				"gauge1": 111,
				"gauge2": 222,
			}))).To(Succeed())

			Expect(collectMetrics(envelopeCollector)).To(receiveInAnyOrder(
				And(
					haveName("gauge1"),
					gaugeWithValue(111),
				),
				And(
					haveName("gauge2"),
					gaugeWithValue(222),
				),
			))
		})

		It("collects timers from the provider", func() {
			envelopeCollector := collector.NewEnvelopeCollector(testhelpers.NewMetricsRegistry())
			Expect(envelopeCollector.Write(timer("http", int64(time.Millisecond), int64(2*time.Millisecond)))).To(Succeed())

			Expect(collectMetrics(envelopeCollector)).To(receiveInAnyOrder(
				And(
					haveName("http_seconds"),
					histogramWithCount(1),
					histogramWithSum(float64(time.Millisecond)/float64(time.Second)),
					histogramWithBuckets(0.01, 0.2, 1.0, 15.0, 60.0),
				),
			))

			Expect(envelopeCollector.Write(timer("http", 0, int64(time.Second)))).To(Succeed())

			Expect(collectMetrics(envelopeCollector)).To(receiveInAnyOrder(
				And(
					haveName("http_seconds"),
					histogramWithCount(2),
					histogramWithSum(float64(time.Second+time.Millisecond)/float64(time.Second)),
					histogramWithBuckets(0.01, 0.2, 1.0, 15.0, 60.0),
				),
			))
		})

		It("ignores unknown envelope types", func() {
			envelopeCollector := collector.NewEnvelopeCollector(testhelpers.NewMetricsRegistry())
			Expect(envelopeCollector.Write(&loggregator_v2.Envelope{})).To(Succeed())
			Expect(collectMetrics(envelopeCollector)).ToNot(Receive())
		})
	})

	Context("tags", func() {
		It("includes tags for counters", func() {
			envelopeCollector := collector.NewEnvelopeCollector(testhelpers.NewMetricsRegistry())
			counter := counterWithTags("label_counter", 1, map[string]string{
				"a": "1",
				"b": "2",
			})
			encodedName := b64.StdEncoding.EncodeToString([]byte("label_counter"))
			Expect(envelopeCollector.Write(counter)).To(Succeed())

			Expect(collectMetrics(envelopeCollector)).To(Receive(And(
				haveName("label_counter"),
				haveLabels(
					labelPair("a", "1"),
					labelPair("b", "2"),
					labelPair("source_id", "some-source-id"),
					labelPair("instance_id", "some-instance-id"),
					labelPair("loggregator_name", encodedName),
				),
			)))
		})

		It("includes tags for gauges", func() {
			envelopeCollector := collector.NewEnvelopeCollector(testhelpers.NewMetricsRegistry())
			gauge := gaugeWithUnit("some_gauge", "percentage")
			encodedName := b64.StdEncoding.EncodeToString([]byte("some_gauge"))
			Expect(envelopeCollector.Write(gauge)).To(Succeed())

			Expect(collectMetrics(envelopeCollector)).To(Receive(And(
				haveName("some_gauge"),
				haveLabels(
					labelPair("unit", "percentage"),
					labelPair("source_id", "some-source-id"),
					labelPair("instance_id", "some-instance-id"),
					labelPair("loggregator_name", encodedName),
				),
			)))
		})

		It("includes tags for timers", func() {
			envelopeCollector := collector.NewEnvelopeCollector(testhelpers.NewMetricsRegistry())
			timer := timerWithTags("some_timer", map[string]string{
				"a": "1",
				"b": "2",
			})
			encodedName := b64.StdEncoding.EncodeToString([]byte("some_timer"))
			Expect(envelopeCollector.Write(timer)).To(Succeed())

			Expect(collectMetrics(envelopeCollector)).To(Receive(And(
				haveName("some_timer_seconds"),
				haveLabels(
					labelPair("a", "1"),
					labelPair("b", "2"),
					labelPair("source_id", "some-source-id"),
					labelPair("instance_id", "some-instance-id"),
					labelPair("loggregator_name", encodedName),
				),
			)))
		})

		It("ignores units if empty", func() {
			envelopeCollector := collector.NewEnvelopeCollector(testhelpers.NewMetricsRegistry())
			encodedName := b64.StdEncoding.EncodeToString([]byte("some_gauge"))
			Expect(envelopeCollector.Write(gauge(map[string]float64{"some_gauge": 7}))).To(Succeed())

			Expect(collectMetrics(envelopeCollector)).To(Receive(And(
				haveName("some_gauge"),
				haveLabels(
					labelPair("source_id", "some-source-id"),
					labelPair("instance_id", "some-instance-id"),
					labelPair("loggregator_name", encodedName),
				),
			)))
		})

		It("converts invalid tags", func() {
			spyRegistry := testhelpers.NewMetricsRegistry()
			envelopeCollector := collector.NewEnvelopeCollector(spyRegistry)
			counter := counterWithTags("label_counter", 1, map[string]string{
				"not.valid":    "2",
				"totally_fine": "3",
			})
			encodedName := b64.StdEncoding.EncodeToString([]byte("label_counter"))
			Expect(envelopeCollector.Write(counter)).To(Succeed())

			Expect(collectMetrics(envelopeCollector)).To(Receive(And(
				haveName("label_counter"),
				haveLabels(
					labelPair("not_valid", "2"),
					labelPair("totally_fine", "3"),
					labelPair("source_id", "some-source-id"),
					labelPair("instance_id", "some-instance-id"),
					labelPair("loggregator_name", encodedName),
				),
			)))
			Expect(spyRegistry.GetMetricValue("modified_tags", map[string]string{"originating_source_id": "some-source-id"})).To(Equal(1.0))
		})

		It("drops reserved tags", func() {
			spyRegistry := testhelpers.NewMetricsRegistry()
			envelopeCollector := collector.NewEnvelopeCollector(spyRegistry)
			counter := counterWithTags("label_counter", 1, map[string]string{
				"__invalid":    "1",
				"totally_fine": "3",
			})
			encodedName := b64.StdEncoding.EncodeToString([]byte("label_counter"))
			Expect(envelopeCollector.Write(counter)).To(Succeed())

			Expect(collectMetrics(envelopeCollector)).To(Receive(And(
				haveName("label_counter"),
				haveLabels(
					labelPair("totally_fine", "3"),
					labelPair("source_id", "some-source-id"),
					labelPair("instance_id", "some-instance-id"),
					labelPair("loggregator_name", encodedName),
				),
			)))
			Expect(spyRegistry.GetMetricValue("invalid_metric_label", map[string]string{"originating_source_id": "some-source-id"})).To(Equal(1.0))
		})

		It("ignores tags with empty values", func() {
			envelopeCollector := collector.NewEnvelopeCollector(testhelpers.NewMetricsRegistry())
			counter := counterWithTags("label_counter", 1, map[string]string{
				"a": "1",
				"b": "2",
				"c": "",
			})
			encodedName := b64.StdEncoding.EncodeToString([]byte("label_counter"))
			Expect(envelopeCollector.Write(counter)).To(Succeed())

			Expect(collectMetrics(envelopeCollector)).To(Receive(And(
				haveName("label_counter"),
				haveLabels(
					labelPair("a", "1"),
					labelPair("b", "2"),
					labelPair("source_id", "some-source-id"),
					labelPair("instance_id", "some-instance-id"),
					labelPair("loggregator_name", encodedName),
				),
			)))
		})

		It("does not include instance_id if empty", func() {
			envelopeCollector := collector.NewEnvelopeCollector(testhelpers.NewMetricsRegistry())
			counter := counterWithEmptyInstanceID("some_name", 1)
			encodedName := b64.StdEncoding.EncodeToString([]byte("some_name"))
			Expect(envelopeCollector.Write(counter)).To(Succeed())

			Expect(collectMetrics(envelopeCollector)).To(Receive(And(
				haveName("some_name"),
				haveLabels(
					labelPair("source_id", "some-source-id"),
					labelPair("loggregator_name", encodedName),
				),
			)))
		})

		It("does not include instance_id or source_id if present in envelop tags", func() {
			envelopeCollector := collector.NewEnvelopeCollector(testhelpers.NewMetricsRegistry())
			counter := counterWithTags("some_name", 1, map[string]string{
				"source_id":   "source_id_from_tags",
				"instance_id": "instance_id_from_tags",
			})
			encodedName := b64.StdEncoding.EncodeToString([]byte("some_name"))
			Expect(envelopeCollector.Write(counter)).To(Succeed())

			Expect(collectMetrics(envelopeCollector)).To(Receive(And(
				haveName("some_name"),
				haveLabels(
					labelPair("source_id", "source_id_from_tags"),
					labelPair("instance_id", "instance_id_from_tags"),
					labelPair("loggregator_name", encodedName),
				),
			)))
		})

		It("includes default tags", func() {
			envelopeCollector := collector.NewEnvelopeCollector(
				testhelpers.NewMetricsRegistry(),
				collector.WithDefaultTags(map[string]string{
					"a":                   "1",
					"b":                   "2",
					"already_on_envelope": "17",
				}))
			counter := counterWithTags("some_name", 1, map[string]string{
				"already_on_envelope": "3",
			})
			encodedName := b64.StdEncoding.EncodeToString([]byte("some_name"))
			Expect(envelopeCollector.Write(counter)).To(Succeed())

			Expect(collectMetrics(envelopeCollector)).To(Receive(And(
				haveName("some_name"),
				haveLabels(
					labelPair("source_id", "some-source-id"),
					labelPair("instance_id", "some-instance-id"),
					labelPair("a", "1"),
					labelPair("b", "2"),
					labelPair("already_on_envelope", "3"),
					labelPair("loggregator_name", encodedName),
				),
			)))
		})
	})

	It("differentiates between metrics with the same name but different labels", func() {
		envelopeCollector := collector.NewEnvelopeCollector(testhelpers.NewMetricsRegistry())
		counter1 := counterWithTags("some_counter", 1, map[string]string{
			"a": "1",
		})
		counter2 := counterWithTags("some_counter", 2, map[string]string{
			"a": "2",
		})
		counter3 := counterWithTags("some_counter", 3, map[string]string{
			"a": "1",
			"b": "2",
		})
		Expect(envelopeCollector.Write(counter1)).To(Succeed())
		Expect(envelopeCollector.Write(counter2)).To(Succeed())
		Expect(envelopeCollector.Write(counter3)).To(Succeed())

		Expect(collectMetrics(envelopeCollector)).To(receiveInAnyOrder(
			counterWithValue(1),
			counterWithValue(2),
			counterWithValue(3),
		))
	})

	It("differentiates between metrics with the same name and same tags but different source id", func() {
		envelopeCollector := collector.NewEnvelopeCollector(testhelpers.NewMetricsRegistry())
		counter := counterWithTags("some_counter", 1, map[string]string{
			"a": "1",
		})
		Expect(envelopeCollector.Write(counter)).To(Succeed())

		sameCounter := counterWithTags("some_counter", 3, map[string]string{
			"a": "1",
		})
		Expect(envelopeCollector.Write(sameCounter)).To(Succeed())

		counter.SourceId = "different_source_id"
		Expect(envelopeCollector.Write(counter)).To(Succeed())

		Expect(collectMetrics(envelopeCollector)).To(HaveLen(2))
	})

	Context("expiring metrics", func() {
		It("removes metrics for source IDs that haven't been updated recently", func() {
			envelopeCollector := collector.NewEnvelopeCollector(testhelpers.NewMetricsRegistry(), collector.WithSourceIDExpiration(time.Second, time.Millisecond))

			Expect(envelopeCollector.Write(counterWithSourceID("counter_to_keep", "persistent"))).To(Succeed())
			go func() {
				for range time.Tick(100 * time.Millisecond) {
					Expect(envelopeCollector.Write(counterWithSourceID("counter_to_keep", "persistent"))).To(Succeed())
				}
			}()

			Expect(envelopeCollector.Write(counterWithSourceID("counter_to_expire", "soon-to-not-exist"))).To(Succeed())
			Expect(envelopeCollector.Write(gaugeWithSourceID("gauge_to_expire", "soon-to-not-exist"))).To(Succeed())
			Expect(collectMetrics(envelopeCollector)).To(receiveInAnyOrder(
				haveName("counter_to_expire"),
				haveName("gauge_to_expire"),
				haveName("counter_to_keep"),
			))

			Eventually(func() chan prometheus.Metric {
				return collectMetrics(envelopeCollector)
			}, 2).Should(receiveOnly(
				haveName("counter_to_keep"),
			))
		})
	})
})

func labelPair(name, value string) *dto.LabelPair {
	return &dto.LabelPair{Name: proto.String(name), Value: proto.String(value)}
}

func receiveInAnyOrder(elements ...interface{}) types.GomegaMatcher {
	return WithTransform(func(metricChan chan prometheus.Metric) []prometheus.Metric {
		close(metricChan)
		var metricSlice []prometheus.Metric
		for metric := range metricChan {
			metricSlice = append(metricSlice, metric)
		}

		return metricSlice
	}, ConsistOf(elements...))
}

func receiveOnly(element interface{}) types.GomegaMatcher {
	return receiveInAnyOrder(element)
}

func collectMetrics(envelopeCollector prometheus.Collector) chan prometheus.Metric {
	collectedMetrics := make(chan prometheus.Metric, 10)
	envelopeCollector.Collect(collectedMetrics)
	return collectedMetrics
}

func counterWithValue(val float64) types.GomegaMatcher {
	return WithTransform(func(metric prometheus.Metric) float64 {
		dtoMetric := &dto.Metric{}
		err := metric.Write(dtoMetric)
		Expect(err).ToNot(HaveOccurred())

		return dtoMetric.GetCounter().GetValue()
	}, Equal(val))
}

func gaugeWithValue(val float64) types.GomegaMatcher {
	return WithTransform(func(metric prometheus.Metric) float64 {
		dtoMetric := &dto.Metric{}
		err := metric.Write(dtoMetric)
		Expect(err).ToNot(HaveOccurred())

		return dtoMetric.GetGauge().GetValue()
	}, Equal(val))
}

func histogramWithCount(count uint64) types.GomegaMatcher {
	return WithTransform(func(metric prometheus.Metric) uint64 {
		histogram := asHistogram(metric)
		return histogram.GetSampleCount()
	}, Equal(count))
}

func histogramWithBuckets(buckets ...float64) types.GomegaMatcher {
	return WithTransform(func(metric prometheus.Metric) []float64 {
		histogram := asHistogram(metric)
		var upperBounds []float64
		for _, bucket := range histogram.GetBucket() {
			upperBounds = append(upperBounds, bucket.GetUpperBound())
		}

		return upperBounds
	}, Equal(buckets))
}

func asHistogram(metric prometheus.Metric) *dto.Histogram {
	dtoMetric := &dto.Metric{}
	err := metric.Write(dtoMetric)
	Expect(err).ToNot(HaveOccurred())

	return dtoMetric.GetHistogram()
}

func histogramWithSum(sum float64) types.GomegaMatcher {
	return WithTransform(func(metric prometheus.Metric) float64 {
		dtoMetric := &dto.Metric{}
		err := metric.Write(dtoMetric)
		Expect(err).ToNot(HaveOccurred())

		histogram := dtoMetric.GetHistogram()
		return histogram.GetSampleSum()
	}, Equal(sum))
}

func haveLabels(labels ...interface{}) types.GomegaMatcher {
	return WithTransform(func(metric prometheus.Metric) []*dto.LabelPair {
		dtoMetric := &dto.Metric{}
		err := metric.Write(dtoMetric)
		Expect(err).ToNot(HaveOccurred())

		return dtoMetric.GetLabel()
	}, ConsistOf(labels...))
}

func haveName(name string) types.GomegaMatcher {
	return WithTransform(func(metric prometheus.Metric) string {
		return metric.Desc().String()
	}, ContainSubstring(fmt.Sprintf(`fqName: "%s"`, name)))
}

func totalCounter(name string, total uint64) *loggregator_v2.Envelope {
	return &loggregator_v2.Envelope{
		SourceId:   "some-source-id",
		InstanceId: "some-instance-id",
		Message: &loggregator_v2.Envelope_Counter{
			Counter: &loggregator_v2.Counter{
				Name:  name,
				Total: total,
			},
		},
	}
}

func counterWithSourceID(name, sourceID string) *loggregator_v2.Envelope {
	return &loggregator_v2.Envelope{
		SourceId:   sourceID,
		InstanceId: "some-instance-id",
		Message: &loggregator_v2.Envelope_Counter{
			Counter: &loggregator_v2.Counter{
				Name:  name,
				Total: 79,
			},
		},
	}
}

func counterWithTags(name string, total uint64, tags map[string]string) *loggregator_v2.Envelope {
	return &loggregator_v2.Envelope{
		SourceId:   "some-source-id",
		InstanceId: "some-instance-id",
		Message: &loggregator_v2.Envelope_Counter{
			Counter: &loggregator_v2.Counter{
				Name:  name,
				Total: total,
			},
		},
		Tags: tags,
	}
}

func timerWithTags(name string, tags map[string]string) *loggregator_v2.Envelope {
	return &loggregator_v2.Envelope{
		SourceId:   "some-source-id",
		InstanceId: "some-instance-id",
		Message: &loggregator_v2.Envelope_Timer{
			Timer: &loggregator_v2.Timer{
				Name:  name,
				Start: 0,
				Stop:  int64(time.Second),
			},
		},
		Tags: tags,
	}
}

func gauge(gauges map[string]float64) *loggregator_v2.Envelope {
	gaugeValues := map[string]*loggregator_v2.GaugeValue{}
	for name, value := range gauges {
		gaugeValues[name] = &loggregator_v2.GaugeValue{Value: value}
	}

	return &loggregator_v2.Envelope{
		SourceId:   "some-source-id",
		InstanceId: "some-instance-id",
		Message: &loggregator_v2.Envelope_Gauge{
			Gauge: &loggregator_v2.Gauge{
				Metrics: gaugeValues,
			},
		},
	}
}

func gaugeWithUnit(name, unit string) *loggregator_v2.Envelope {
	return &loggregator_v2.Envelope{
		SourceId:   "some-source-id",
		InstanceId: "some-instance-id",
		Message: &loggregator_v2.Envelope_Gauge{
			Gauge: &loggregator_v2.Gauge{
				Metrics: map[string]*loggregator_v2.GaugeValue{
					name: {
						Unit:  unit,
						Value: 1,
					},
				},
			},
		},
	}
}

func gaugeWithSourceID(name, sourceID string) *loggregator_v2.Envelope {
	return &loggregator_v2.Envelope{
		SourceId:   sourceID,
		InstanceId: "some-instance-id",
		Message: &loggregator_v2.Envelope_Gauge{
			Gauge: &loggregator_v2.Gauge{
				Metrics: map[string]*loggregator_v2.GaugeValue{
					name: {Value: 1},
				},
			},
		},
	}
}

func timer(name string, start, stop int64) *loggregator_v2.Envelope {
	return &loggregator_v2.Envelope{
		SourceId:   "some-source-id",
		InstanceId: "some-instance-id",
		Message: &loggregator_v2.Envelope_Timer{
			Timer: &loggregator_v2.Timer{
				Name:  name,
				Start: start,
				Stop:  stop,
			},
		},
	}
}

func counterWithEmptyInstanceID(name string, total uint64) *loggregator_v2.Envelope {
	return &loggregator_v2.Envelope{
		SourceId: "some-source-id",
		Message: &loggregator_v2.Envelope_Counter{
			Counter: &loggregator_v2.Counter{
				Name:  name,
				Total: total,
			},
		},
	}
}
