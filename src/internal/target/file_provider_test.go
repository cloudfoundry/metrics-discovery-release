package target_test

import (
	"code.cloudfoundry.org/metrics-discovery/internal/target"
	. "code.cloudfoundry.org/metrics-discovery/internal/testhelpers"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"io/ioutil"
	"log"
	"os"
	"time"
)

var _ = Describe("FileProvider", func() {
	var logger = log.New(GinkgoWriter, "", 0)

	It("parses a file and provides scrape targets", func() {
		f := targetConfigFile("targets.yml")
		writeConfigFile(fmt.Sprintf(targetListTemplate, "/metrics", "https"), f)

		provider := target.NewFileProvider(f.Name(), time.Second, logger)
		go provider.Start()

		Eventually(provider.GetTargets).Should(ContainElement(MatchFields(IgnoreExtras, Fields{
			"JobName":                Equal("some-job"),
			"MetricsPath":            Equal("/metrics"),
			"Scheme":                 Equal("https"),
			"ServiceDiscoveryConfig": HaveDNSConfig("some-host", "A", 1111),
		})))
	})

	It("updates scrapes targets on an interval", func() {
		f := targetConfigFile("targets.yml")

		writeConfigFile(fmt.Sprintf(targetListTemplate, "/metrics", "https"), f)
		provider := target.NewFileProvider(f.Name(), 300*time.Millisecond, logger)
		go provider.Start()

		Eventually(provider.GetTargets).Should(ContainElement(MatchFields(IgnoreExtras, Fields{
			"JobName":                Equal("some-job"),
			"MetricsPath":            Equal("/metrics"),
			"Scheme":                 Equal("https"),
			"ServiceDiscoveryConfig": HaveDNSConfig("some-host", "A", 1111),
		})))

		writeConfigFile(fmt.Sprintf(targetListTemplate, "/some-other-path", "https"), f)
		Eventually(provider.GetTargets).Should(ContainElement(MatchFields(IgnoreExtras, Fields{
			"JobName":                Equal("some-job"),
			"MetricsPath":            Equal("/some-other-path"),
			"Scheme":                 Equal("https"),
			"ServiceDiscoveryConfig": HaveDNSConfig("some-host", "A", 1111),
		})))

		Eventually(provider.GetTargets).ShouldNot(ContainElement(MatchFields(IgnoreExtras, Fields{
			"JobName":                Equal("some-job"),
			"MetricsPath":            Equal("/metrics"),
			"Scheme":                 Equal("https"),
			"ServiceDiscoveryConfig": HaveDNSConfig("some-host", "A", 1111),
		})))
	})
})

func targetConfigFile(fileName string) *os.File {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		log.Fatal(err)
	}

	f, err := ioutil.TempFile(dir, fileName)
	if err != nil {
		log.Fatal(err)
	}

	return f
}

func writeConfigFile(config string, f *os.File) {
	err := f.Truncate(0)
	if err != nil {
		log.Fatal(err)
	}

	_, err = f.WriteAt([]byte(config), 0) //truncate removes content but doesn't change offset
	if err != nil {
		log.Fatal(err)
	}
}

const (
	targetListTemplate = `---
scrape_configs:
- job_name: some-job
  metrics_path: "%s"
  scheme: %s
  dns_sd_configs:
  - names:
    - some-host
    type: A
    port: 1111
`
)
