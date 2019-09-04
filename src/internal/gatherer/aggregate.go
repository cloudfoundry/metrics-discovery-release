package gatherer

import (
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"log"
)

type Aggregate struct {
	Gatherers []prometheus.Gatherer
	Logger    *log.Logger
}

func (ag Aggregate) Gather() ([]*dto.MetricFamily, error) {
	var families []*dto.MetricFamily
	for _, gatherer := range ag.Gatherers {
		f, err := gatherer.Gather()
		if err != nil {
			ag.Logger.Printf("unable to gather metrics: %s", err)
			continue
		}
		families = append(families, f...)
	}

	return families, nil
}
