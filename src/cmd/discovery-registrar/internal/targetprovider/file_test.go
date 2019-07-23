package targetprovider_test

import (
	"code.cloudfoundry.org/metrics-discovery/cmd/discovery-registrar/internal/targetprovider"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"log"
	"os"
	"time"
)

var _ = Describe("FileTargetProvider", func() {
	It("parses a file and provides scrape targets", func() {
		f := routeListConfigFile("targets.yml")
		writeRouteList(fmt.Sprintf(routeListTemplate, "https://some-host:9999/metrics"), f)

		provider := targetprovider.NewFileProvider(f.Name(), time.Second)
		go provider.Start()

		Eventually(provider.GetTargets).Should (ContainElement("https://some-host:9999/metrics"))
	})

	It("updates scrapes targets on an interval", func() {
		f := routeListConfigFile("targets.yml")

		writeRouteList(fmt.Sprintf(routeListTemplate, "https://some-host:9999/metrics"), f)
		provider := targetprovider.NewFileProvider(f.Name(), 300 * time.Millisecond)
		go provider.Start()

		Eventually(provider.GetTargets).Should(ContainElement("https://some-host:9999/metrics"))

		writeRouteList(fmt.Sprintf(routeListTemplate, "https://some-other-host:9999/metrics"), f)
		Eventually(provider.GetTargets).Should(ContainElement("https://some-other-host:9999/metrics"))
		Eventually(provider.GetTargets).ShouldNot(ContainElement("https://some-host:9999/metrics"))
	})
})

func routeListConfigFile(fileName string) *os.File {
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

func writeRouteList(config string, f *os.File) {
	err := f.Truncate(0)
	if err != nil {
		log.Fatal(err)
	}

	_, err = f.WriteAt([]byte(config), 0)  //truncate removes content but doesn't change offset
	if err != nil {
		log.Fatal(err)
	}
}

const routeListTemplate = `---
routes:
- %s
`