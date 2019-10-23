package app_test

import (
	"code.cloudfoundry.org/go-metric-registry/testhelpers"
	"code.cloudfoundry.org/metrics-discovery/cmd/config-generator/app"
	"fmt"
	. "github.com/benjamintf1/unmarshalledmatchers"
	"github.com/nats-io/nats.go"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
			configPath: tmpDir + "/scrape_targets.json",
			logger:     log.New(GinkgoWriter, "", 0),
		}
	}

	var readTargets = func(tc *testContext) string {
		var fileData []byte

		Eventually(func() error {
			var err error
			fileData, err = ioutil.ReadFile(tc.configPath)
			return err
		}).Should(Succeed())

		return string(fileData)
	}

	It("Generates a config with data from the queue", func() {
		tc := setup()

		generator := app.NewConfigGenerator(
			tc.subscriber.Subscribe,
			100 * time.Millisecond,
			time.Hour,
			time.Hour,
			tc.configPath,
			testhelpers.NewMetricsRegistry(),
			tc.logger,
		)
		go generator.Start()

		tc.subscriber.callback(&nats.Msg{
			Data: target("job1"),
		})
		tc.subscriber.callback(&nats.Msg{
			Data: target("job2"),
		})

		targets := readTargets(tc)
		Expect(targets).To(MatchUnorderedJSON(`[
			{
				"targets": [
				  "localhost:8080"
				],
				"labels": {
				  "job": "job1"
				}
			},
			{
				"targets": [
				  "localhost:8080"
				],
				"labels": {
				  "job": "job2"
				}
			}
		]`))
	})

	It("generates the config at the given interval", func() {
		tc := setup()

		generator := app.NewConfigGenerator(
			tc.subscriber.Subscribe,
			time.Hour,
			time.Hour,
			time.Hour,
			tc.configPath,
			testhelpers.NewMetricsRegistry(),
			tc.logger,
		)
		go generator.Start()

		tc.subscriber.callback(&nats.Msg{
			Data: target("job1"),
		})
		tc.subscriber.callback(&nats.Msg{
			Data: target("job2"),
		})

		Consistently(func() error {
			_, err := ioutil.ReadFile(tc.configPath)
			return err
		}).ShouldNot(Succeed())
	})

	It("doesn't duplicate jobs", func() {
		tc := setup()

		generator := app.NewConfigGenerator(
			tc.subscriber.Subscribe,
			100 * time.Millisecond,
			time.Hour,
			time.Hour,
			tc.configPath,
			testhelpers.NewMetricsRegistry(),
			tc.logger,
		)
		go generator.Start()

		tc.subscriber.callback(&nats.Msg{
			Data: target("job1"),
		})
		tc.subscriber.callback(&nats.Msg{
			Data: target("job1", "localhost:8081"),
		})

		targets := readTargets(tc)

		Expect(targets).To(MatchJSON(`[
			{
				"targets": [
				  "localhost:8080",
				  "localhost:8081"
				],
				"labels": {
				  "job": "job1"
				}
			}
		]`))
	})

	It("expires configs after the given interval", func() {
		tc := setup()

		generator := app.NewConfigGenerator(
			tc.subscriber.Subscribe,
			100 * time.Millisecond,
			200*time.Millisecond,
			100*time.Millisecond,
			tc.configPath,
			testhelpers.NewMetricsRegistry(),
			tc.logger,
		)
		go generator.Start()

		tc.subscriber.callback(&nats.Msg{
			Data: target("ephemeral"),
		})
		tc.subscriber.callback(&nats.Msg{
			Data: target("persistent"),
		})

		go func() {
			t := time.NewTicker(100 * time.Millisecond)
			for range t.C {
				tc.subscriber.callback(&nats.Msg{
					Data: target("persistent"),
				})
			}
		}()

		Eventually(func() string {
			return readTargets(tc)
		}).Should(MatchUnorderedJSON(`[
			{
				"targets": [
				  "localhost:8080"
				],
				"labels": {
				  "job": "ephemeral"
				}
			},
			{
				"targets": [
				  "localhost:8080"
				],
				"labels": {
				  "job": "persistent"
				}
			}
		]`))

		Eventually(func() string {
			return readTargets(tc)
		}).Should(MatchUnorderedJSON(`[
			{
				"targets": [
				  "localhost:8080"
				],
				"labels": {
				  "job": "persistent"
				}
			}
		]`))
	})

	It("increments a delivered metric", func() {
		tc := setup()

		spyMetrics := testhelpers.NewMetricsRegistry()
		generator := app.NewConfigGenerator(
			tc.subscriber.Subscribe,
			100 * time.Millisecond,
			600*time.Millisecond,
			100*time.Millisecond,
			tc.configPath,
			spyMetrics,
			tc.logger,
		)
		go generator.Start()

		tc.subscriber.callback(&nats.Msg{
			Data: target("ephemeral"),
		})

		Eventually(func() int {
			return int(spyMetrics.GetMetric("delivered", map[string]string{}).Value())
		}).Should(Equal(1))
	})
})

func target(source string, additionalTargets ...string) []byte {
	additionalTargetString := ""
	for _, t := range additionalTargets {
		additionalTargetString += fmt.Sprintf("  - %s\n", t)
	}

	return []byte(fmt.Sprintf(targetTemplate, additionalTargetString, source, source))
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

var targetTemplate = `---
targets:
  - "localhost:8080"
%s
labels:
  job: %s
source: %s
`