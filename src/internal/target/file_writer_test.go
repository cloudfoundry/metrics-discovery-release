package target_test

import (
	"io/ioutil"
	"log"
	"os"

	"code.cloudfoundry.org/loggregator-agent-release/src/pkg/scraper"
	"code.cloudfoundry.org/metrics-discovery/internal/target"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"
)

var _ = Describe("File Writer", func() {
	type testContext struct {
		targetsFile string
		metricsHost string
		cfg         target.WriterConfig
		logger      *log.Logger
	}

	var setup = func(scrapeConfigs []scraper.PromScraperConfig) *testContext {
		tc := &testContext{
			targetsFile: os.TempDir() + "/metrics_targets.yml",
			metricsHost: "127.0.0.1:9991",
			logger:      log.New(GinkgoWriter, "", 0),
		}

		tc.cfg = target.WriterConfig{
			MetricsHost: tc.metricsHost,
			DefaultLabels: map[string]string{
				"a": "1",
				"b": "2",
			},
			InstanceID:    "instance_id",
			File:          tc.targetsFile,
			ScrapeConfigs: scrapeConfigs,
		}

		return tc
	}

	var teardown = func(tc *testContext) {
		os.Remove(tc.targetsFile)
	}

	var readTargetsFromFile = func(tc *testContext) []target.Target {
		f, err := ioutil.ReadFile(tc.targetsFile)
		Expect(err).ToNot(HaveOccurred())

		var targets []target.Target
		err = yaml.Unmarshal(f, &targets)
		Expect(err).ToNot(HaveOccurred())

		return targets
	}

	It("creates a metrics_targets.yml file with the agent as a target.", func() {
		tc := setup(nil)
		defer teardown(tc)

		target.WriteFile(tc.cfg, tc.logger)

		Expect(readTargetsFromFile(tc)).To(ConsistOf(
			target.Target{
				Targets: []string{tc.metricsHost},
				Labels: map[string]string{
					"a":           "1",
					"b":           "2",
					"instance_id": "instance_id",
				},
				Source: "metrics_agent_exporter__instance_id",
			},
		))
	})

	It("adds default labels from scrape config and global config to target", func() {
		tc := setup([]scraper.PromScraperConfig{{
			SourceID:   "source_id_scraped",
			InstanceID: "ignore this",
			Labels: map[string]string{
				"scrape_config_label": "lemons",
			},
		}})
		defer teardown(tc)

		target.WriteFile(tc.cfg, tc.logger)

		Expect(readTargetsFromFile(tc)).To(ContainElement(
			target.Target{
				Targets: []string{tc.metricsHost},
				Labels: map[string]string{
					"__param_id":          "source_id_scraped",
					"a":                   "1",
					"b":                   "2",
					"scrape_config_label": "lemons",
					"source_id":           "source_id_scraped",
					"instance_id":         "instance_id",
				},
				Source: "source_id_scraped__instance_id",
			},
		))
	})
})
