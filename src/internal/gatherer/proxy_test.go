package gatherer_test

import (
	"log"
	"net/http"
	"net/http/httptest"
	"strings"

	metrichelpers "code.cloudfoundry.org/go-metric-registry/testhelpers"
	"code.cloudfoundry.org/loggregator-agent-release/src/pkg/scraper"
	"code.cloudfoundry.org/metrics-discovery/internal/gatherer"
	"code.cloudfoundry.org/metrics-discovery/internal/testhelpers"
	"code.cloudfoundry.org/tlsconfig"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	io_prometheus_client "github.com/prometheus/client_model/go"
)

var _ = Describe("Proxy", func() {
	type testContext struct {
		promServer   *stubPromServer
		scrapeCerts  *testhelpers.TestCerts
		scrapeConfig scraper.PromScraperConfig
		metrics      *metrichelpers.SpyMetricsRegistry
		loggr        *log.Logger
	}

	var setup = func(scheme, scrapePath string, scrapeHeaders map[string]string) *testContext {
		scrapeCerts := testhelpers.GenerateCerts("scrapeCA")

		var promServer *stubPromServer
		var serverName string
		if scheme == "https" {
			promServer = newStubHttpsPromServer(scrapeCerts)
			serverName = "server"
		} else {
			promServer = newStubPromServer()
		}
		promServer.resp = promOutput

		scrapeConfig := scraper.PromScraperConfig{
			Port:       promServer.port,
			Scheme:     scheme,
			Path:       scrapePath,
			ServerName: serverName,
			Headers:    scrapeHeaders,
		}

		return &testContext{
			promServer:   promServer,
			scrapeCerts:  scrapeCerts,
			scrapeConfig: scrapeConfig,
			metrics:      metrichelpers.NewMetricsRegistry(),
			loggr:        log.New(GinkgoWriter, "", 0),
		}
	}

	var buildProxyCollector = func(tc *testContext) *gatherer.ProxyGatherer {
		return gatherer.NewProxyGatherer(
			tc.scrapeConfig,
			tc.scrapeCerts.Cert("client"),
			tc.scrapeCerts.Key("client"),
			tc.scrapeCerts.CA(),
			tc.metrics,
			tc.loggr,
		)
	}

	It("collects metrics from a prom target", func() {
		tc := setup("http", "metrics", nil)
		proxyCollector := buildProxyCollector(tc)

		mfs, err := proxyCollector.Gather()
		Expect(err).ToNot(HaveOccurred())

		Expect(mfs).To(ConsistOf(
			haveFamilyName("metric1"),
			haveFamilyName("metric2"),
			And(
				haveFamilyName("metric3"),
				haveMetrics(
					gaugeWith(11, map[string]string{"direction": "ingress"}),
					gaugeWith(22, map[string]string{"direction": "egress"}),
				),
			),
		))
	})

	It("can scrape with mTLS", func() {
		tc := setup("https", "metrics", nil)
		proxyCollector := buildProxyCollector(tc)

		mfs, err := proxyCollector.Gather()
		Expect(err).ToNot(HaveOccurred())

		Expect(mfs).To(ConsistOf(
			haveFamilyName("metric1"),
			haveFamilyName("metric2"),
			And(
				haveFamilyName("metric3"),
				haveMetrics(
					gaugeWith(11, map[string]string{"direction": "ingress"}),
					gaugeWith(22, map[string]string{"direction": "egress"}),
				),
			),
		))
	})

	It("scrapes the correct path", func() {
		tc := setup("https", "alternative-path", nil)
		proxyCollector := buildProxyCollector(tc)

		mfs, err := proxyCollector.Gather()
		Expect(err).ToNot(HaveOccurred())

		Expect(tc.promServer.requestPaths).To(Receive(Equal("/alternative-path")))
		Expect(mfs).To(ConsistOf(
			haveFamilyName("metric1"),
			haveFamilyName("metric2"),
			And(
				haveFamilyName("metric3"),
				haveMetrics(
					gaugeWith(11, map[string]string{"direction": "ingress"}),
					gaugeWith(22, map[string]string{"direction": "egress"}),
				),
			),
		))
	})

	It("adds scrape headers", func() {
		tc := setup("https", "metrics", map[string]string{
			"header": "value",
		})
		proxyCollector := buildProxyCollector(tc)

		mfs, err := proxyCollector.Gather()
		Expect(err).ToNot(HaveOccurred())

		Expect(tc.promServer.requestHeaders).To(Receive(HaveKeyWithValue("Header", []string{"value"})))
		Expect(mfs).To(ConsistOf(
			haveFamilyName("metric1"),
			haveFamilyName("metric2"),
			And(
				haveFamilyName("metric3"),
				haveMetrics(
					gaugeWith(11, map[string]string{"direction": "ingress"}),
					gaugeWith(22, map[string]string{"direction": "egress"}),
				),
			),
		))
	})

	It("returns an error if the scrape fails", func() {
		tc := setup("http", "metrics", nil)
		tc.scrapeConfig = scraper.PromScraperConfig{
			Port:     "9091",
			Scheme:   "http",
			Path:     "this_server_does_not_exist",
			SourceID: "failed_scrape_id",
		}

		proxyCollector := buildProxyCollector(tc)

		_, err := proxyCollector.Gather()
		Expect(err).To(HaveOccurred())

		Expect(tc.metrics.GetMetric(
			"failed_scrapes",
			map[string]string{"scrape_source_id": "failed_scrape_id"}).Value(),
		).To(Equal(1.0))
	})
})

type stubPromServer struct {
	resp string
	port string

	requestHeaders chan http.Header
	requestPaths   chan string
}

func newStubPromServer() *stubPromServer {
	s := &stubPromServer{
		requestHeaders: make(chan http.Header, 100),
		requestPaths:   make(chan string, 100),
	}

	server := httptest.NewServer(s)

	addr := server.URL
	tokens := strings.Split(addr, ":")
	s.port = tokens[len(tokens)-1]

	return s
}

func newStubHttpsPromServer(scrapeCerts *testhelpers.TestCerts) *stubPromServer {
	s := &stubPromServer{
		requestHeaders: make(chan http.Header, 100),
		requestPaths:   make(chan string, 100),
	}

	var serverOpts []tlsconfig.ServerOption
	serverOpts = append(serverOpts, tlsconfig.WithClientAuthenticationFromFile(scrapeCerts.CA()))
	serverConf, err := tlsconfig.Build(
		tlsconfig.WithIdentityFromFile(scrapeCerts.Cert("server"), scrapeCerts.Key("server")),
	).Server(serverOpts...)
	Expect(err).ToNot(HaveOccurred())

	server := httptest.NewUnstartedServer(s)
	server.TLS = serverConf
	server.StartTLS()

	addr := server.Listener.Addr().String()
	tokens := strings.Split(addr, ":")
	s.port = tokens[len(tokens)-1]

	return s
}

func (s *stubPromServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	s.requestHeaders <- req.Header
	s.requestPaths <- req.URL.Path
	_, err := w.Write([]byte(s.resp))
	Expect(err).ToNot(HaveOccurred())
}

func haveFamilyName(name string) types.GomegaMatcher {
	return WithTransform(func(mf *io_prometheus_client.MetricFamily) string {
		return mf.GetName()
	}, Equal(name))
}

func haveMetrics(metricMatchers ...types.GomegaMatcher) types.GomegaMatcher {
	return WithTransform(func(mf *io_prometheus_client.MetricFamily) []*io_prometheus_client.Metric {
		return mf.GetMetric()
	}, ConsistOf(metricMatchers))
}

func gaugeWith(value float64, labels map[string]string) types.GomegaMatcher {
	return And(
		WithTransform(func(m *io_prometheus_client.Metric) float64 {
			gauge := m.GetGauge()
			Expect(gauge).ToNot(BeNil())
			return gauge.GetValue()
		}, Equal(value)),

		WithTransform(func(m *io_prometheus_client.Metric) map[string]string {
			labels := map[string]string{}
			for _, labelPair := range m.GetLabel() {
				labels[labelPair.GetName()] = labelPair.GetValue()
			}

			return labels
		}, Equal(labels)),
	)
}

const promOutput = `
# HELP metric1 The first counter.
# TYPE metric1 counter
metric1 1
# HELP metric2 The first gauge.
# TYPE metric2 gauge
metric2 2
# HELP metric3 The second gauge.
# TYPE metric3 gauge
metric3 {direction="ingress"} 11
metric3 {direction="egress"} 22
`
