package target_test

import (
	"code.cloudfoundry.org/metrics-discovery/internal/target"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"log"
	"os"
	"time"
)

var _ = Describe("FileProvider", func() {
	var logger = log.New(GinkgoWriter, "", 0)

	It("parses a file and provides scrape targets", func() {
		f := targetConfigFile("targets.yml")
		writeConfigFile(fmt.Sprintf(targetListTemplate, "source"), f)

		provider := target.NewFileProvider(f.Name(), time.Second, logger)
		go provider.Start()

		Eventually(provider.GetTargets).Should(ConsistOf(&target.Target{
			Targets: []string{"localhost:8080"},
			Labels: map[string]string{
				"job": "job-name",
			},
			Source: "source",
		}))
	})

	It("updates scrapes targets on an interval", func() {
		f := targetConfigFile("targets.yml")

		writeConfigFile(fmt.Sprintf(targetListTemplate, "source1"), f)
		provider := target.NewFileProvider(f.Name(), 300*time.Millisecond, logger)
		go provider.Start()

		Eventually(provider.GetTargets).Should(ConsistOf(&target.Target{
			Targets: []string{"localhost:8080"},
			Labels: map[string]string{
				"job": "job-name",
			},
			Source: "source1",
		}))

		writeConfigFile(fmt.Sprintf(targetListTemplate, "source2"), f)
		Eventually(provider.GetTargets).Should(ContainElement(&target.Target{
			Targets: []string{"localhost:8080"},
			Labels: map[string]string{
				"job": "job-name",
			},
			Source: "source2",
		}))

		Eventually(provider.GetTargets).ShouldNot(ContainElement(&target.Target{
			Targets: []string{"localhost:8080"},
			Labels: map[string]string{
				"job": "job-name",
			},
			Source: "source1",
		}))
	})

	It("ignores targets missing a source", func() {
		f := targetConfigFile("targets.yml")
		writeConfigFile(targetMissingSource, f)

		provider := target.NewFileProvider(f.Name(), time.Second, logger)
		go provider.Start()

		Consistently(provider.GetTargets).Should(BeEmpty())
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
targets:
  - "localhost:8080"
labels:
  job: job-name
source: %s
`
	targetMissingSource = `---
targets:
  - "localhost:8080"
labels:
  job: job-name
`
)
