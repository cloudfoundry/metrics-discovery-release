package app_test

import (
	"code.cloudfoundry.org/metrics-discovery/cmd/config-generator/app"
	"github.com/nats-io/nats.go"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/config"
	sd_config "github.com/prometheus/prometheus/discovery/config"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
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

		app.StartConfigGeneration(tc.subscriber.Subscribe, tc.configPath, tc.logger)
		tc.subscriber.callback(&nats.Msg{
			Data: []byte("https://route-1.com:8080/something"),
		})
		tc.subscriber.callback(&nats.Msg{
			Data: []byte("http://route-2.com:8080/metrics"),
		})

		scrapeConfigs := readScrapeConfigs(tc)

		Expect(scrapeConfigs).To(ConsistOf(
			MatchFields(IgnoreExtras, Fields{
				"JobName":                Equal("https://route-1.com:8080/something"),
				"MetricsPath":            Equal("/something"),
				"Scheme":                 Equal("https"),
				"ServiceDiscoveryConfig": haveTarget("route-1.com:8080"),
			}),
			MatchFields(IgnoreExtras, Fields{
				"JobName":                Equal("http://route-2.com:8080/metrics"),
				"MetricsPath":            Equal("/metrics"),
				"Scheme":                 Equal("http"),
				"ServiceDiscoveryConfig": haveTarget("route-2.com:8080"),
			}),
		))
	})

	It("doesn't duplicate addresses", func() {
		tc := setup()

		app.StartConfigGeneration(tc.subscriber.Subscribe, tc.configPath, tc.logger)
		tc.subscriber.callback(&nats.Msg{
			Data: []byte("https://route-1.com:8080/something"),
		})
		tc.subscriber.callback(&nats.Msg{
			Data: []byte("https://route-1.com:8080/something"),
		})

		scrapeConfigs := readScrapeConfigs(tc)

		Expect(scrapeConfigs).To(ConsistOf(
			MatchFields(IgnoreExtras, Fields{
				"JobName":                Equal("https://route-1.com:8080/something"),
				"MetricsPath":            Equal("/something"),
				"Scheme":                 Equal("https"),
				"ServiceDiscoveryConfig": haveTarget("route-1.com:8080"),
			}),
		))
	})
})

func haveTarget(target string) types.GomegaMatcher {
	return WithTransform(
		func(sdConfig sd_config.ServiceDiscoveryConfig) []*targetgroup.Group {
			return sdConfig.StaticConfigs
		}, ConsistOf(
			&targetgroup.Group{
				Targets: []model.LabelSet{
					{"__address__": model.LabelValue(target)},
				},
				Source: "0",
			},
		))
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
