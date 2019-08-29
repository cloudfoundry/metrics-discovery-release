module code.cloudfoundry.org/metrics-discovery

go 1.12

require (
	cloud.google.com/go v0.44.3 // indirect
	code.cloudfoundry.org/go-diodes v0.0.0-20190809170250-f77fb823c7ee
	code.cloudfoundry.org/go-envstruct v1.5.0
	code.cloudfoundry.org/go-loggregator v0.0.0-20190809213911-969cb33bee6a // pinned
	code.cloudfoundry.org/loggregator-agent v0.0.0-20190531203354-322071cc6807
	code.cloudfoundry.org/tlsconfig v0.0.0-20190710180242-462f72de1106
	github.com/go-kit/kit v0.9.0 // indirect
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/gogo/protobuf v1.2.1
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/jpillora/backoff v0.0.0-20180909062703-3050d21c67d7 // indirect
	github.com/mwitkow/go-conntrack v0.0.0-20190716064945-2f068394615f // indirect
	github.com/nats-io/nats-server/v2 v2.0.4 // indirect
	github.com/nats-io/nats.go v1.8.1
	github.com/onsi/ginkgo v1.9.0
	github.com/onsi/gomega v1.7.0
	github.com/prometheus/client_golang v1.1.0
	github.com/prometheus/client_model v0.0.0-20190812154241-14fe0d1b01d4
	github.com/prometheus/common v0.6.0
	github.com/prometheus/prometheus v2.11.0+incompatible // pinned
	golang.org/x/crypto v0.0.0-20190829043050-9756ffdc2472 // indirect
	google.golang.org/api v0.9.0 // indirect
	google.golang.org/appengine v1.6.2 // indirect
	google.golang.org/grpc v1.23.0
	gopkg.in/yaml.v2 v2.2.2
)

replace (
	code.cloudfoundry.org/loggregator-agent => code.cloudfoundry.org/loggregator-agent-release/src v0.0.0-20190828205358-ce6b7b280d44
	github.com/prometheus/common => github.com/prometheus/common v0.5.0
)
