package app_test

import (
	"bytes"
	"log"
	"os"

	"code.cloudfoundry.org/go-envstruct"
	"code.cloudfoundry.org/metrics-discovery/cmd/config-generator/app"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("configuration", func() {

	var requiredVars = []string{
		"NATS_CA_PATH",
		"NATS_CERT_PATH",
		"NATS_KEY_PATH",
		"SCRAPE_CONFIG_FILE_PATH",
	}

	BeforeEach(func() {
		err := os.Setenv("NATS_HOSTS", "some-secret")
		Expect(err).ToNot(HaveOccurred())

		for _, v := range requiredVars {
			err := os.Setenv(v, "some-value")
			Expect(err).ToNot(HaveOccurred())
		}
	})
	AfterEach(func() {
		err := os.Unsetenv("NATS_HOSTS")
		Expect(err).ToNot(HaveOccurred())

		for _, v := range requiredVars {
			err := os.Unsetenv(v)
			Expect(err).ToNot(HaveOccurred())
		}
	})

	It("does not report the value of the NATS_HOSTS environment variable", func() {
		var output bytes.Buffer
		envstruct.ReportWriter = &output
		logger := log.New(GinkgoWriter, "", log.LstdFlags)
		app.LoadConfig(logger)
		Expect(output.String()).ToNot(ContainSubstring("some-secret"))
	})

})
