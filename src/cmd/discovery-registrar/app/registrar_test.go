package app_test

import (
	"fmt"
	"net/http"

	"code.cloudfoundry.org/go-metric-registry/testhelpers"
	"code.cloudfoundry.org/metrics-discovery/cmd/discovery-registrar/app"
	"code.cloudfoundry.org/metrics-discovery/internal/target"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"

	"log"
	"sync"
	"time"
)

var _ = Describe("Dynamic Registrar", func() {
	type testContext struct {
		publisher      *fakePublisher
		targetProvider *fakeTargetProvider
		pprofPort      uint16
		metrics        *testhelpers.SpyMetricsRegistry
		logger         *log.Logger
		registrar      *app.DynamicRegistrar
	}

	var setup = func(publishInterval time.Duration, debugMetrics bool) *testContext {
		tc := &testContext{
			publisher: newFakePublisher(),
			targetProvider: &fakeTargetProvider{
				targets: []*target.Target{
					{
						Targets: []string{"10.0.0.1:8080"},
					},
				},
			},
			pprofPort: 1234,
			metrics:   testhelpers.NewMetricsRegistry(),
			logger:    log.New(GinkgoWriter, "", 0),
		}

		tc.registrar = app.NewDynamicRegistrar(tc.targetProvider.GetTargets, tc.publisher, publishInterval, tc.metrics, tc.logger)
		go tc.registrar.Start(debugMetrics, tc.pprofPort)

		return tc
	}

	var teardown = func(tc *testContext) {
		tc.registrar.Stop()
	}

	It("publishes targets from the target provider", func() {
		tc := setup(time.Second, false)
		defer teardown(tc)

		Eventually(tc.publisher.targets).Should(HaveLen(1))

		var target target.Target
		err := yaml.Unmarshal(tc.publisher.targets()[0], &target)
		Expect(err).ToNot(HaveOccurred())
		Expect(target.Targets).To(ConsistOf("10.0.0.1:8080"))
		Expect(tc.publisher.publishedToQueue()).To(Equal("metrics.scrape_targets"))
		Expect(tc.targetProvider.timesCalled()).To(Equal(1))
	})

	It("publishes targets from the target provider on an interval", func() {
		tc := setup(300*time.Millisecond, false)
		defer teardown(tc)

		Eventually(tc.targetProvider.timesCalled).Should(BeNumerically(">=", 4))
		Eventually(func() int {
			return len(tc.publisher.targets())
		}).Should(BeNumerically(">=", 4))
		Expect(tc.publisher.publishedToQueue()).To(Equal("metrics.scrape_targets"))
	})

	It("increments a delivered metric", func() {
		tc := setup(300*time.Millisecond, false)
		defer teardown(tc)

		Eventually(func() int {
			return int(tc.metrics.GetMetric("sent", map[string]string{}).Value())
		}).Should(BeNumerically(">=", 1))
	})

	It("does not emit debug metrics by default", func() {
		tc := setup(300*time.Millisecond, false)
		defer teardown(tc)

		Consistently(tc.metrics.GetDebugMetricsEnabled, 3).ShouldNot(BeTrue())
		Consistently(func() error {
			_, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/debug/pprof/", tc.pprofPort))
			return err
		}).ShouldNot(BeNil())
	})

	It("can enable debug metrics", func() {
		tc := setup(300*time.Millisecond, true)
		defer teardown(tc)

		Eventually(tc.metrics.GetDebugMetricsEnabled, 3).Should(BeTrue())
		var resp *http.Response
		Eventually(func() error {
			var err error
			resp, err = http.Get(fmt.Sprintf("http://127.0.0.1:%d/debug/pprof/", tc.pprofPort))
			return err
		}).Should(BeNil())
		Expect(resp.StatusCode).To(Equal(200))
		Eventually(tc.publisher.targets).Should(HaveLen(1))
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

func (fp *fakePublisher) publishedToQueue() string {
	fp.Lock()
	defer fp.Unlock()

	return fp.queue
}

type fakeTargetProvider struct {
	sync.Mutex
	called  int
	targets []*target.Target
}

func (ftp *fakeTargetProvider) GetTargets() []*target.Target {
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
