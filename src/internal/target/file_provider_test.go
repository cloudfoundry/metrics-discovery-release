package target_test

import (
	"fmt"
	"log"
	"os"
	"time"

	"code.cloudfoundry.org/metrics-discovery/internal/target"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("FileProvider", func() {
	var logger = log.New(GinkgoWriter, "", 0)

	It("parses a file and provides scrape targets", func() {
		f := targetConfigFile("targets.yml")
		writeConfigFile(multiTargetListTemplate, f)

		provider := target.NewFileProvider(f.Name(), time.Second, logger)
		go provider.Start()

		Eventually(provider.GetTargets).Should(ConsistOf(
			&target.Target{
				Targets: []string{"localhost:8080"},
				Source:  "source-1",
			},
			&target.Target{
				Targets: []string{"localhost:8080"},
				Source:  "source-2",
			},
		))
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
	dir, err := os.MkdirTemp("", "")
	if err != nil {
		log.Fatal(err)
	}

	f, err := os.CreateTemp(dir, fileName)
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
- targets:
  - "localhost:8080"
  labels:
    job: job-name
  source: %s
`
	multiTargetListTemplate = `---
- targets:
  - "localhost:8080"
  source: source-1
- targets:
  - "localhost:8080"
  source: source-2
`
	targetMissingSource = `---
- targets:
  - "localhost:8080"
  labels:
    job: job-name
`
)
