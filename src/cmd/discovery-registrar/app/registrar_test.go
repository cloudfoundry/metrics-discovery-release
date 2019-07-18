package app_test

import (
	"code.cloudfoundry.org/metrics-discovery/cmd/discovery-registrar/app"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sync"
	"time"
)

var _ = Describe("App", func() {
	var (
		fp  *fakePublisher
		r   *app.Registrar
	)

	BeforeEach(func() {
		fp = newFakePublisher()
	})

	AfterEach(func() {
		r.Stop()
	})

	It("publishes the configured routes", func() {
		routes := []string{
			"https://route-1.com:8080/metrics",
			"https://route-2.com:8080/metrics",
			"https://route-3.com:8080/metrics",
		}

		r = app.NewRegistrar(routes, 5*time.Second, fp)
		go r.Start()

		Eventually(fp.routes).Should(ConsistOf(
			"https://route-1.com:8080/metrics",
			"https://route-2.com:8080/metrics",
			"https://route-3.com:8080/metrics",
		))
		Expect(fp.publishedToQueue()).To(Equal("metrics.endpoints"))
	})

	It("publishes all the routes on an interval", func() {
		routes := []string{
			"https://route-1.com:8080/metrics",
			"https://route-2.com:8080/metrics",
		}

		r = app.NewRegistrar(routes, 300 * time.Millisecond, fp)
		go r.Start()

		Eventually(fp.callsToPublish).Should(BeNumerically(">=", 6))
		Expect(fp.publishedToQueue()).To(Equal("metrics.endpoints"))
	})
})

type fakePublisher struct {
	sync.Mutex
	rts     []string
	called  int
	rtQueue string
}

func newFakePublisher() *fakePublisher {
	return &fakePublisher{}
}

func (fp *fakePublisher) Publish(queue string, msg []byte) error {
	fp.Lock()
	defer fp.Unlock()

	fp.rtQueue = queue
	fp.called++
	fp.rts = append(fp.rts, string(msg))

	return nil
}

func (fp *fakePublisher) Close() {}

func (fp *fakePublisher) routes() []string {
	fp.Lock()
	defer fp.Unlock()

	dst := make([]string, len(fp.rts))
	copy(dst, fp.rts)

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

	return fp.rtQueue
}


