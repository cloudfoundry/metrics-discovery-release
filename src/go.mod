module code.cloudfoundry.org/metrics-discovery

go 1.12

require (
	code.cloudfoundry.org/go-envstruct v1.5.0
	code.cloudfoundry.org/go-loggregator v0.0.0-20190809213911-969cb33bee6a // pinned
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/nats-io/jwt v0.2.12 // indirect
	github.com/nats-io/nats-server/v2 v2.0.2 // indirect
	github.com/nats-io/nats.go v1.8.1
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/prometheus/common v0.6.0
	github.com/prometheus/prometheus v2.11.0+incompatible // pinned
	golang.org/x/net v0.0.0-20190724013045-ca1201d0de80 // indirect
	golang.org/x/sys v0.0.0-20190812172437-4e8604ab3aff // indirect
	golang.org/x/text v0.3.2 // indirect
	google.golang.org/grpc v1.21.1 // indirect
	gopkg.in/yaml.v2 v2.2.2
)

replace github.com/prometheus/common => github.com/prometheus/common v0.5.0
