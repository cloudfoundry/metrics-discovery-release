package app_test

import (
	"code.cloudfoundry.org/go-loggregator/metrics/testhelpers"
	"code.cloudfoundry.org/metrics-discovery/cmd/discovery-registrar/app"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"gopkg.in/yaml.v2"

	"github.com/prometheus/prometheus/config"
	"log"
	"sync"
	"time"
)

var _ = Describe("Dynamic Registrar", func() {
	type testContext struct {
		publisher      *fakePublisher
		targetProvider *fakeTargetProvider
		metrics        *testhelpers.SpyMetricsRegistry
		logger         *log.Logger
		registrar      *app.DynamicRegistrar
	}

	var setup = func(publishInterval time.Duration) *testContext {
		tc := &testContext{
			publisher: newFakePublisher(),
			targetProvider: &fakeTargetProvider{
				targets: []config.ScrapeConfig{{
					JobName:     "some-job",
					MetricsPath: "/metrics",
					Scheme:      "https",
				}},
			},
			metrics: testhelpers.NewMetricsRegistry(),
			logger:  log.New(GinkgoWriter, "", 0),
		}

		tc.registrar = app.NewDynamicRegistrar(tc.targetProvider.GetTargets, tc.publisher, publishInterval, tc.metrics, tc.logger)
		go tc.registrar.Start()

		return tc
	}

	var teardown = func(tc *testContext) {
		tc.registrar.Stop()
	}

	It("publishes targets from the target provider", func() {
		tc := setup(time.Second)
		defer teardown(tc)

		Eventually(tc.publisher.targets).Should(HaveLen(1))

		var scrapeConfig config.ScrapeConfig
		err := yaml.Unmarshal(tc.publisher.targets()[0], &scrapeConfig)
		Expect(err).ToNot(HaveOccurred())
		Expect(scrapeConfig).To(MatchFields(IgnoreExtras, Fields{
			"JobName":     Equal("some-job"),
			"MetricsPath": Equal("/metrics"),
			"Scheme":      Equal("https"),
		}))
		Expect(tc.publisher.publishedToQueue()).To(Equal("metrics.scrape_targets"))
		Expect(tc.targetProvider.timesCalled()).To(Equal(1))
	})

	It("publishes targets from the target provider on an interval", func() {
		tc := setup(300 * time.Millisecond)
		defer teardown(tc)

		Eventually(tc.targetProvider.timesCalled).Should(BeNumerically(">=", 4))
		Expect(len(tc.publisher.targets())).To(BeNumerically(">=", 4))
		Expect(tc.publisher.publishedToQueue()).To(Equal("metrics.scrape_targets"))
	})

	It("increments a delivered metric", func() {
		tc := setup(300 * time.Millisecond)
		defer teardown(tc)

		Eventually(func() int {
			return int(tc.metrics.GetMetric("sent", map[string]string{}).Value())
		}).Should(BeNumerically(">=", 1))
	})
})

type fakePublisher struct {
	sync.Mutex
	messages [][]byte
	called   int
	queue    string
}

func newFakePublisher() *fakePublisher {
	return &fakePublisher{}
}

func (fp *fakePublisher) Publish(queue string, msg []byte) error {
	fp.Lock()
	defer fp.Unlock()

	fp.queue = queue
	fp.called++
	fp.messages = append(fp.messages, msg)

	return nil
}

func (fp *fakePublisher) Close() {}

func (fp *fakePublisher) targets() [][]byte {
	fp.Lock()
	defer fp.Unlock()

	dst := make([][]byte, len(fp.messages))
	copy(dst, fp.messages)

	return dst
}

func (fp *fakePublisher) callsToPublish() int {
	fp.Lock()
	defer fp.Unlock()

	return fp.called
}

func (fp *fakePublisher) publishedToQueue() string {
	fp.Lock()
	defer fp.Unlock()

	return fp.queue
}

type fakeTargetProvider struct {
	sync.Mutex
	called  int
	targets []config.ScrapeConfig
}

func (ftp *fakeTargetProvider) GetTargets() []config.ScrapeConfig {
	ftp.Lock()
	defer ftp.Unlock()

	ftp.called++
	return ftp.targets
}

func (ftp *fakeTargetProvider) timesCalled() int {
	ftp.Lock()
	defer ftp.Unlock()

	return ftp.called
}
