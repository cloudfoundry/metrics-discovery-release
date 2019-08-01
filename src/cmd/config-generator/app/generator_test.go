package app_test

import (
	"code.cloudfoundry.org/go-loggregator/metrics/testhelpers"
	"code.cloudfoundry.org/metrics-discovery/cmd/config-generator/app"
	"fmt"
	"github.com/nats-io/nats.go"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/prometheus/prometheus/config"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"time"
)

var _ = Describe("Config generator", func() {
	type testContext struct {
		subscriber *fakeSubscriber
		configPath string
		logger     *log.Logger
	}

	var setup = func() *testContext {
		tmpDir, err := ioutil.TempDir("", "")
		Expect(err).ToNot(HaveOccurred())

		return &testContext{
			subscriber: newFakeSubscriber(),
			configPath: tmpDir + "/prom_config.yml",
			logger:     log.New(GinkgoWriter, "", 0),
		}
	}

	var readScrapeConfigs = func(tc *testContext) []config.ScrapeConfig {
		fileData, err := ioutil.ReadFile(tc.configPath)
		Expect(err).ToNot(HaveOccurred())

		var scrapeConfigs []config.ScrapeConfig
		Expect(yaml.Unmarshal(fileData, &scrapeConfigs)).To(Succeed())

		return scrapeConfigs
	}

	It("Generates a config with data from the queue", func() {
		tc := setup()

		generator := app.NewConfigGenerator(tc.subscriber.Subscribe,
			time.Hour,
			time.Hour,
			tc.configPath,
			testhelpers.NewMetricsRegistry(),
			tc.logger,
		)
		go generator.Start()

		tc.subscriber.callback(&nats.Msg{
			Data: target("route1", "path1"),
		})
		tc.subscriber.callback(&nats.Msg{
			Data: target("route2", "path2"),
		})

		scrapeConfigs := readScrapeConfigs(tc)

		Expect(scrapeConfigs).To(ConsistOf(
			MatchFields(IgnoreExtras, Fields{
				"JobName":                Equal("route1"),
				"MetricsPath":            Equal("path1"),
			}),
			MatchFields(IgnoreExtras, Fields{
				"JobName":                Equal("route2"),
				"MetricsPath":            Equal("path2"),
			}),
		))
	})

	It("doesn't duplicate jobs", func() {
		tc := setup()

		generator := app.NewConfigGenerator(tc.subscriber.Subscribe,
			time.Hour,
			time.Hour,
			tc.configPath,
			testhelpers.NewMetricsRegistry(),
			tc.logger,
		)
		go generator.Start()

		tc.subscriber.callback(&nats.Msg{
			Data: target("route1", "path1"),
		})
		tc.subscriber.callback(&nats.Msg{
			Data: target("route1", "path1"),
		})

		scrapeConfigs := readScrapeConfigs(tc)

		Expect(scrapeConfigs).To(ConsistOf(
			MatchFields(IgnoreExtras, Fields{
				"JobName":                Equal("route1"),
				"MetricsPath":            Equal("path1"),
			}),
		))
	})

	It("expires configs after the given interval", func() {
		tc := setup()

		generator := app.NewConfigGenerator(
			tc.subscriber.Subscribe,
			600*time.Millisecond,
			100*time.Millisecond,
			tc.configPath,
			testhelpers.NewMetricsRegistry(),
			tc.logger,
		)
		go generator.Start()

		tc.subscriber.callback(&nats.Msg{
			Data: target("ephemeral", "ephemeral"),
		})
		tc.subscriber.callback(&nats.Msg{
			Data: target("persistent", "persistent"),
		})

		go func() {
			t := time.NewTicker(600 * time.Millisecond)
			for range t.C {
				tc.subscriber.callback(&nats.Msg{
					Data: target("persistent", "persistent"),
				})
			}
		}()

		Expect(readScrapeConfigs(tc)).To(ConsistOf(
			MatchFields(IgnoreExtras, Fields{
				"JobName":                Equal("ephemeral"),
				"MetricsPath":            Equal("ephemeral"),
			}),
			MatchFields(IgnoreExtras, Fields{
				"JobName":                Equal("persistent"),
				"MetricsPath":            Equal("persistent"),
			}),
		))

		Eventually(func() []config.ScrapeConfig {
			return readScrapeConfigs(tc)
		}).Should(ConsistOf(
			MatchFields(IgnoreExtras, Fields{
				"JobName":                Equal("persistent"),
				"MetricsPath":            Equal("persistent"),
			}),
		))
	})

	It("increments a delivered metric", func() {
		tc := setup()

		spyMetrics := testhelpers.NewMetricsRegistry()
		generator := app.NewConfigGenerator(
			tc.subscriber.Subscribe,
			600*time.Millisecond,
			100*time.Millisecond,
			tc.configPath,
			spyMetrics,
			tc.logger,
		)
		go generator.Start()

		tc.subscriber.callback(&nats.Msg{
			Data: target("ephemeral", "ephemeral"),
		})

		Eventually(func() int {
			return int(spyMetrics.GetMetric("delivered", map[string]string{}).Value())
		}).Should(Equal(1))
	})
})

func target(jobName, path string) []byte {
	return []byte(fmt.Sprintf(targetTemplate, jobName, path))
}

type fakeSubscriber struct {
	called   bool
	callback func(m *nats.Msg)
}

func newFakeSubscriber() *fakeSubscriber {
	return &fakeSubscriber{}
}

func (fs *fakeSubscriber) Subscribe(queue string, callback nats.MsgHandler) (*nats.Subscription, error) {
	fs.called = true
	fs.callback = callback

	return nil, nil
}


var targetTemplate = `
  job_name: "%s"
  metrics_path: "%s"
`