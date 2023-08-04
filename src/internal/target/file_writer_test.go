package target_test

import (
	"log"
	"os"

	"code.cloudfoundry.org/loggregator-agent-release/src/pkg/scraper"
	"code.cloudfoundry.org/metrics-discovery/internal/target"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
)

var _ = Describe("File Writer", func() {
	const host = "127.0.0.1:9991"

	var (
		scrapeCfgs []scraper.PromScraperConfig
		tmpDir     string
	)

	BeforeEach(func() {
		scrapeCfgs = []scraper.PromScraperConfig{}
		tmpDir = GinkgoT().TempDir()
	})

	JustBeforeEach(func() {
		cfg := target.WriterConfig{
			MetricsHost: host,
			DefaultLabels: map[string]string{
				"a": "1",
				"b": "2",
			},
			InstanceID:    "instance_id",
			File:          tmpDir + "/metrics_targets.yml",
			ScrapeConfigs: scrapeCfgs,
		}
		target.WriteFile(cfg, log.New(GinkgoWriter, "", 0))
	})

	var readTargetsFromFile = func(tmpDir string) []target.Target {
		f, err := os.ReadFile(tmpDir + "/metrics_targets.yml")
		Expect(err).ToNot(HaveOccurred())

		var targets []target.Target
		err = yaml.Unmarshal(f, &targets)
		Expect(err).ToNot(HaveOccurred())

		return targets
	}

	It("creates a metrics_targets.yml file with the agent as a target", func() {
		Expect(readTargetsFromFile(tmpDir)).To(ConsistOf(
			target.Target{
				Targets: []string{host},
				Labels: map[string]string{
					"a":           "1",
					"b":           "2",
					"instance_id": "instance_id",
				},
				Source: "metrics_agent_exporter__instance_id",
			},
		))
	})

	Context("when other prom scraper configs are provided", func() {
		BeforeEach(func() {
			scrapeCfgs = []scraper.PromScraperConfig{{
				SourceID:   "source_id_scraped",
				InstanceID: "ignore this",
				Labels: map[string]string{
					"scrape_config_label": "lemons",
				},
			}}
		})

		It("adds default labels from those prom scraper configs to the target", func() {
			Expect(readTargetsFromFile(tmpDir)).To(ContainElement(
				target.Target{
					Targets: []string{host},
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
})
