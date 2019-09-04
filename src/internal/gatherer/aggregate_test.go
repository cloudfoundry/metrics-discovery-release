package gatherer_test

import (
	"code.cloudfoundry.org/metrics-discovery/internal/gatherer"
	"github.com/gogo/protobuf/proto"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"log"
)

var _ = Describe("Aggregator", func() {

	It("aggregatres metrics from contained gatherers", func() {
		g1 := mockGatherer{
			mf: []*dto.MetricFamily{
				{
					Name:   proto.String("First Gatherer"),
				},
			},
		}
		g2 := mockGatherer{
			mf: []*dto.MetricFamily{
				{
					Name:   proto.String("Second Gatherer"),
				},
			},
		}

		ag := gatherer.Aggregate{
			Gatherers: []prometheus.Gatherer{g1, g2},
			Logger:    log.New(GinkgoWriter, "", 0),
		}

		mfs, err := ag.Gather()
		Expect(err).ToNot(HaveOccurred())

		var names []string
		for _, mf := range mfs {
			names = append(names, *mf.Name)
		}

		Expect(names).To(ConsistOf("First Gatherer", "Second Gatherer"))
	})
})

type mockGatherer struct {
	mf []*dto.MetricFamily
}

func (sg mockGatherer) Gather() ([]*dto.MetricFamily, error) {
	return sg.mf, nil
}
