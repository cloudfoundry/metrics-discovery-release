package gatherer

import (
	"code.cloudfoundry.org/go-loggregator/metrics"
	"code.cloudfoundry.org/loggregator-agent/pkg/scraper"
	"code.cloudfoundry.org/tlsconfig"
	"fmt"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type ProxyGatherer struct {
	scrapeConfigs []scraper.PromScraperConfig
	httpDoers     map[string]func(*http.Request) (*http.Response, error)
	metrics       metricsRegistry
}

type metricsRegistry interface {
	NewCounter(string, ...metrics.MetricOption) metrics.Counter
}

func NewProxyGatherer(
	scrapeConfigs []scraper.PromScraperConfig,
	certPath,
	keyPath,
	caPath string,
	metrics metricsRegistry,
	loggr *log.Logger,
) *ProxyGatherer {
	pg := &ProxyGatherer{
		scrapeConfigs: scrapeConfigs,
		metrics:       metrics,
	}

	httpDoers := map[string]func(*http.Request) (*http.Response, error){}
	for _, sc := range scrapeConfigs {
		httpDoers[sc.SourceID] = buildHttpClient(certPath, keyPath, caPath, sc.ServerName, loggr).Do
		pg.newFailedScrapeMetric(sc.SourceID)
	}

	pg.httpDoers = httpDoers

	return pg
}

func buildHttpClient(certPath, keyPath, caPath, serverName string, loggr *log.Logger) *http.Client {
	tlsOptions := []tlsconfig.TLSOption{tlsconfig.WithInternalServiceDefaults()}
	clientOptions := []tlsconfig.ClientOption{}

	if certPath != "" && keyPath != "" {
		tlsOptions = append(tlsOptions, tlsconfig.WithIdentityFromFile(certPath, keyPath))
		clientOptions = append(clientOptions, tlsconfig.WithServerName(serverName))
	}

	if caPath != "" {
		clientOptions = append(clientOptions, tlsconfig.WithAuthorityFromFile(caPath))
	}

	tlsConfig, err := tlsconfig.Build(tlsOptions...).Client(clientOptions...)
	if err != nil {
		loggr.Fatal(err)
	}

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
		Timeout: 5 * time.Second,
	}
}

// Gather implements prometheus.Gatherer
func (c *ProxyGatherer) Gather() ([]*io_prometheus_client.MetricFamily, error) {
	var mfs []*io_prometheus_client.MetricFamily

	wg := sync.WaitGroup{}
	mfsLock := sync.Mutex{}

	for _, sc := range c.scrapeConfigs {
		wg.Add(1)
		go func(sc scraper.PromScraperConfig) {
			defer wg.Done()

			scrapeResults, err := c.scrape(sc)
			if err != nil {
				c.incFailedScrapes(sc.SourceID)
				return
			}

			mfsLock.Lock()
			mfs = append(mfs, scrapeResults...)
			mfsLock.Unlock()
		}(sc)
	}

	wg.Wait()
	return mfs, nil
}

func (c *ProxyGatherer) incFailedScrapes(sourceID string) {
	c.newFailedScrapeMetric(sourceID).Add(1)
}

func (c *ProxyGatherer) newFailedScrapeMetric(sourceID string) metrics.Counter {
	return c.metrics.NewCounter(
		"failed_scrapes",
		metrics.WithHelpText("Total failures when scraping target."),
		metrics.WithMetricTags(map[string]string{
			"scrape_source_id": sourceID,
		}),
	)
}

func (c *ProxyGatherer) scrape(scrapeConfig scraper.PromScraperConfig) ([]*io_prometheus_client.MetricFamily, error) {
	req, err := c.scrapeRequest(scrapeConfig)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpDoers[scrapeConfig.SourceID](req)
	if err != nil {
		return nil, err
	}

	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, body)
	}

	p := &expfmt.TextParser{}
	res, err := p.TextToMetricFamilies(resp.Body)
	if err != nil {
		return nil, err
	}

	var families []*io_prometheus_client.MetricFamily
	for _, family := range res {
		families = append(families, family)
	}

	return families, err
}

func (c *ProxyGatherer) scrapeRequest(scrapeConfig scraper.PromScraperConfig) (*http.Request, error) {
	url := fmt.Sprintf("%s://127.0.0.1:%s/%s",
		scrapeConfig.Scheme, scrapeConfig.Port, strings.TrimPrefix(scrapeConfig.Path, "/"))
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	requestHeader := http.Header{}
	for k, v := range scrapeConfig.Headers {
		requestHeader[k] = []string{v}
	}
	req.Header = requestHeader

	return req, nil
}
