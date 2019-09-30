package app_test

import (
	"code.cloudfoundry.org/go-loggregator"
	"code.cloudfoundry.org/loggregator-agent/pkg/config"
	"code.cloudfoundry.org/loggregator-agent/pkg/scraper"
	"code.cloudfoundry.org/metrics-discovery/cmd/metrics-agent/app"
	"code.cloudfoundry.org/metrics-discovery/internal/target"
	"code.cloudfoundry.org/metrics-discovery/internal/testhelpers"
	metrichelpers "code.cloudfoundry.org/go-metric-registry/testhelpers"
	"code.cloudfoundry.org/tlsconfig"
	"context"
	"fmt"
	"github.com/gogo/protobuf/proto"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"time"
)

var _ = Describe("MetricsAgent", func() {
	var (
		metricsAgent *app.MetricsAgent
		grpcPort     uint16
		metricsPort  uint16
		testCerts    *testhelpers.TestCerts
		targetsFile  string

		ingressClient            *loggregator.IngressClient
		fakeScrapeConfigProvider app.ScrapeConfigProvider
	)

	BeforeEach(func() {
		testCerts = testhelpers.GenerateCerts("loggregatorCA")

		targetsFile = os.TempDir() + "metrics_targets.yml"
		grpcPort = getFreePort()
		metricsPort = getFreePort()
		cfg := app.Config{
			MetricsExporter: app.MetricsExporterConfig{
				Port:                 metricsPort,
				ExpirationInterval:   time.Minute,
				TimeToLive:           10 * time.Minute,
				WhitelistedTimerTags: []string{"whitelist1", "whitelist2"},
			},
			MetricsServer: config.MetricsServer{
				CAFile:   testCerts.CA(),
				CertFile: testCerts.Cert("client"),
				KeyFile:  testCerts.Key("client"),
			},
			Tags: map[string]string{
				"a": "1",
				"b": "2",
			},
			GRPC: app.GRPCConfig{
				Port:     grpcPort,
				CAFile:   testCerts.CA(),
				CertFile: testCerts.Cert("metron"),
				KeyFile:  testCerts.Key("metron"),
			},
			Addr:              "127.0.0.1",
			InstanceID:        "instance_id",
			MetricsTargetFile: targetsFile,
		}

		ingressClient = newTestingIngressClient(int(grpcPort), testCerts)

		stubPromServer := newStubPromServer()
		stubPromServer.resp = promOutput
		fakeScrapeConfigProvider = func() ([]scraper.PromScraperConfig, error) {
			return []scraper.PromScraperConfig{{
				Port:     stubPromServer.port,
				SourceID: "source_id_scraped",
				Scheme:   "http",
				Path:     "metrics",
				Labels: map[string]string{
					"scrape_config_label": "lemons",
				},
			}}, nil
		}

		testLogger := log.New(GinkgoWriter, "", log.LstdFlags)
		metricsAgent = app.NewMetricsAgent(cfg, fakeScrapeConfigProvider, metrichelpers.NewMetricsRegistry(), testLogger)
		go metricsAgent.Run()
		waitForMetricsEndpoint(metricsPort, testCerts)
	})

	AfterEach(func() {
		metricsAgent.Stop()
		_ = os.Remove(targetsFile)
	})

	It("creates a metrics_targets.yml file with the agent as a target.", func() {
		f, err := ioutil.ReadFile(targetsFile)
		Expect(err).ToNot(HaveOccurred())

		var targets []target.Target
		err = yaml.Unmarshal(f, &targets)
		Expect(err).ToNot(HaveOccurred())

		Expect(targets).To(ConsistOf(
			target.Target{
				Targets: []string{fmt.Sprintf("127.0.0.1:%d", metricsPort)},
				Labels:  map[string]string{
					"a":   "1",
					"b":   "2",
				},
				Source:  "metrics_agent_exporter__instance_id",
			},
			target.Target{
				Targets: []string{fmt.Sprintf("127.0.0.1:%d", metricsPort)},
				Labels: map[string]string{
					"__param_id":          "source_id_scraped",
					"a":                   "1",
					"b":                   "2",
					"scrape_config_label": "lemons",
				},
				Source: "source_id_scraped__instance_id",
			},
		))
	})

	It("adds default labels to the exporter target", func() {
		f, err := ioutil.ReadFile(targetsFile)
		Expect(err).ToNot(HaveOccurred())

		var targets []target.Target
		err = yaml.Unmarshal(f, &targets)
		Expect(err).ToNot(HaveOccurred())

		Expect(targets).To(ContainElement(
			target.Target{
				Targets: []string{fmt.Sprintf("127.0.0.1:%d", metricsPort)},
				Labels: map[string]string{
					"a":   "1",
					"b":   "2",
				},
				Source: "metrics_agent_exporter__instance_id",
			},
		))
	})

	It("adds default labels from scrape config and global config to target", func() {
		f, err := ioutil.ReadFile(targetsFile)
		Expect(err).ToNot(HaveOccurred())

		var targets []target.Target
		err = yaml.Unmarshal(f, &targets)
		Expect(err).ToNot(HaveOccurred())

		Expect(targets).To(ContainElement(
			target.Target{
				Targets: []string{fmt.Sprintf("127.0.0.1:%d", metricsPort)},
				Labels: map[string]string{
					"__param_id":          "source_id_scraped",
					"a":                   "1",
					"b":                   "2",
					"scrape_config_label": "lemons",
				},
				Source: "source_id_scraped__instance_id",
			},
		))
	})

	It("exposes metrics on a prometheus endpoint", func() {
		cancel := doUntilCancelled(func() {
			ingressClient.EmitCounter("total_counter", loggregator.WithTotal(22))
		})
		defer cancel()

		Eventually(getMetricFamilies(metricsPort, "", testCerts), 3).Should(HaveKey("total_counter"))

		metric := getMetric("total_counter", metricsPort, testCerts)
		Expect(metric.GetCounter().GetValue()).To(BeNumerically("==", 22))
	})

	It("filters timer tags not in whitelist", func() {
		cancel := doUntilCancelled(func() {
			ingressClient.EmitTimer("timer", time.Now().Add(-time.Second), time.Now(),
				loggregator.WithTimerSourceInfo("source-id-from-source-info", "instance-id-from-source-info"),
				loggregator.WithEnvelopeTags(map[string]string{
					"whitelist1": "whitelist1",
					"whitelist2": "whitelist2",
					"a":          "1",
					"b":          "2",
				}),
			)
		})
		defer cancel()

		Eventually(getMetricFamilies(metricsPort, "", testCerts), 3).Should(HaveKey("timer_seconds"))

		metric := getMetric("timer_seconds", metricsPort, testCerts)
		Expect(metric.GetLabel()).To(ConsistOf(
			&dto.LabelPair{Name: proto.String("whitelist1"), Value: proto.String("whitelist1")},
			&dto.LabelPair{Name: proto.String("whitelist2"), Value: proto.String("whitelist2")},

			// source and instance id are added from envelope properties
			&dto.LabelPair{Name: proto.String("source_id"), Value: proto.String("source-id-from-source-info")},
			&dto.LabelPair{Name: proto.String("instance_id"), Value: proto.String("instance-id-from-source-info")},
		))
	})

	It("filters out blacklisted source id envelopes", func() {
		cancel := doUntilCancelled(func() {
			ingressClient.EmitCounter("prom_scraped",
				loggregator.WithTotal(22),
				loggregator.WithCounterSourceInfo("source_id_scraped", "some-instance-id"),
			)

			ingressClient.EmitCounter("non_prom_scraped",
				loggregator.WithTotal(22),
				loggregator.WithCounterSourceInfo("source_id_non_scraped", "some-instance-id"),
			)
		})
		defer cancel()

		Eventually(getMetricFamilies(metricsPort, "", testCerts), 3).Should(HaveKey("non_prom_scraped"))
		Consistently(getMetricFamilies(metricsPort, "", testCerts), 3).Should(Not(HaveKey("prom_scraped")))
	})

	It("proxies to prom endpoints", func() {
		Eventually(getMetricFamilies(metricsPort, "source_id_scraped", testCerts), 3).Should(HaveKey("proxyMetric"))
	})

	It("only returns the metrics for the given ID", func() {
		cancel := doUntilCancelled(func() {
			ingressClient.EmitCounter("total_counter", loggregator.WithTotal(22))
		})
		defer cancel()

		Eventually(getMetricFamilies(metricsPort, "", testCerts), 3).Should(HaveLen(1))
		metric := getMetric("total_counter", metricsPort, testCerts)
		Expect(metric.GetCounter().GetValue()).To(BeNumerically("==", 22))

		Eventually(getMetricFamilies(metricsPort, "source_id_scraped", testCerts), 3).Should(HaveKey("proxyMetric"))
		Expect(getMetricFamilies(metricsPort, "source_id_scraped", testCerts)()).To(HaveLen(1))
	})

	It("returns a 404 for unknown IDs", func() {
		cancel := doUntilCancelled(func() {
			ingressClient.EmitCounter("total_counter", loggregator.WithTotal(22))
		})
		defer cancel()

		_, err := getMetricsResponse(metricsPort, "foobarbaz", testCerts)
		Expect(err).To(MatchError("unexpected status code 404"))
	})

	It("aggregates delta counters", func() {
		cancel := doUntilCancelled(func() {
			ingressClient.EmitCounter("delta_counter", loggregator.WithDelta(2))
		})
		defer cancel()

		Eventually(getMetricFamilies(metricsPort, "", testCerts), 3).Should(HaveKey("delta_counter"))

		originialValue := getMetric("delta_counter", metricsPort, testCerts).GetCounter().GetValue()

		Eventually(func() float64 {
			metric := getMetric("delta_counter", metricsPort, testCerts)
			if metric == nil {
				return 0
			}
			return metric.GetCounter().GetValue()
		}).Should(BeNumerically(">", originialValue))
	})
})

func doUntilCancelled(f func()) context.CancelFunc {
	ctx, cancelFunc := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.Tick(100 * time.Millisecond)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker:
				f()
			}
		}
	}()

	return cancelFunc
}

func waitForMetricsEndpoint(port uint16, testCerts *testhelpers.TestCerts) {
	Eventually(func() error {
		_, err := getMetricsResponse(port, "", testCerts)
		return err
	}).Should(Succeed())
}

func getMetricsResponse(port uint16, id string, testCerts *testhelpers.TestCerts) (*http.Response, error) {
	tlsConfig, err := tlsconfig.Build(tlsconfig.WithIdentityFromFile(testCerts.Cert("client"), testCerts.Key("client"))).
		Client(tlsconfig.WithAuthorityFromFile(testCerts.CA()))
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Transport: &http.Transport{TLSClientConfig: tlsConfig},
	}

	url := fmt.Sprintf("https://127.0.0.1:%d/metrics?id=%s", port, id)
	resp, err := client.Get(url)
	if err == nil && resp.StatusCode != http.StatusOK {
		return resp, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	return resp, err
}

func getMetricFamilies(port uint16, id string, testCerts *testhelpers.TestCerts) func() map[string]*dto.MetricFamily {
	return func() map[string]*dto.MetricFamily {
		resp, err := getMetricsResponse(port, id, testCerts)

		metricFamilies, err := new(expfmt.TextParser).TextToMetricFamilies(resp.Body)
		if err != nil {
			return nil
		}

		return metricFamilies
	}
}

func getMetric(metricName string, port uint16, testCerts *testhelpers.TestCerts) *dto.Metric {
	families := getMetricFamilies(port, "", testCerts)()
	family, ok := families[metricName]
	if !ok {
		return nil
	}

	metrics := family.Metric
	Expect(metrics).To(HaveLen(1))
	return metrics[0]
}

func newTestingIngressClient(grpcPort int, testCerts *testhelpers.TestCerts) *loggregator.IngressClient {
	tlsConfig, err := loggregator.NewIngressTLSConfig(testCerts.CA(), testCerts.Cert("metron"), testCerts.Key("metron"))
	Expect(err).ToNot(HaveOccurred())

	ingressClient, err := loggregator.NewIngressClient(
		tlsConfig,
		loggregator.WithAddr(fmt.Sprintf("127.0.0.1:%d", grpcPort)),
		loggregator.WithLogger(log.New(GinkgoWriter, "[TEST INGRESS CLIENT] ", 0)),
		loggregator.WithBatchMaxSize(1),
	)
	Expect(err).ToNot(HaveOccurred())

	return ingressClient
}

func getFreePort() uint16 {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		log.Fatal(err)
	}

	defer l.Close()
	return uint16(l.Addr().(*net.TCPAddr).Port)
}

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

func (s *stubPromServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	s.requestHeaders <- req.Header
	s.requestPaths <- req.URL.Path
	w.Write([]byte(s.resp))
}

const promOutput = `
# HELP proxyMetric The first counter.
# TYPE proxyMetric counter
proxyMetric 1
`
