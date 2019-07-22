package app_test

import (
	"code.cloudfoundry.org/metrics-discovery/cmd/dynamic-registrar/app"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sync"
	"time"
)

var _ = Describe("Dynamic Registrar", func() {
	var (
		fp  *fakePublisher
		r   *app.DynamicRegistrar
		ftp *fakeTargetProvider
		cfg app.Config
	)

	BeforeEach(func() {
		fp = newFakePublisher()
		ftp = &fakeTargetProvider{
			targets: []string{
				"https://some-host:8080/metrics",
			},
		}
	})

	AfterEach(func() {
		r.Stop()
	})

	It("publishes routes from the target provider", func() {
		cfg = app.Config{
			PublishInterval: time.Second,
		}

		r = app.NewDynamicRegistrar(ftp, fp, cfg)
		go r.Start()

		Eventually(fp.routes).Should(ConsistOf("https://some-host:8080/metrics"))
		Expect(fp.publishedToQueue()).To(Equal("metrics.endpoints"))
		Expect(ftp.timesCalled()).To(Equal(1))
	})

	It("publishes routes from the target provider on an interval", func() {
		cfg = app.Config{
			PublishInterval: 300 * time.Millisecond,
		}

		r = app.NewDynamicRegistrar(ftp, fp, cfg)
		go r.Start()

		Eventually(ftp.timesCalled).Should(BeNumerically(">=", 4))
		Expect(len(fp.routes())).To(BeNumerically(">=", 4))
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

type fakeTargetProvider struct {
	sync.Mutex
	called  int
	targets []string
}

func (ftp *fakeTargetProvider) GetTargets() []string {
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
