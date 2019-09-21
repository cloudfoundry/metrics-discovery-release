module code.cloudfoundry.org/metrics-discovery

go 1.12

require (
	cloud.google.com/go v0.46.3 // indirect
	code.cloudfoundry.org/go-diodes v0.0.0-20190809170250-f77fb823c7ee
	code.cloudfoundry.org/go-envstruct v1.5.0
	code.cloudfoundry.org/go-loggregator v0.0.0-20190809213911-969cb33bee6a // pinned
	code.cloudfoundry.org/loggregator-agent v0.0.0-20190918193342-14308cf69de1
	code.cloudfoundry.org/tlsconfig v0.0.0-20190710180242-462f72de1106
	github.com/benjamintf1/unmarshalledmatchers v0.0.0-20190408201839-bb1c1f34eaea
	github.com/go-kit/kit v0.9.0 // indirect
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/gogo/protobuf v1.3.0
	github.com/google/btree v1.0.0 // indirect
	github.com/gopherjs/gopherjs v0.0.0-20190915194858-d3ddacdb130f // indirect
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/jpillora/backoff v0.0.0-20180909062703-3050d21c67d7 // indirect
	github.com/mwitkow/go-conntrack v0.0.0-20190716064945-2f068394615f // indirect
	github.com/nats-io/nats-server/v2 v2.1.0 // indirect
	github.com/nats-io/nats.go v1.8.1
	github.com/onsi/ginkgo v1.10.1
	github.com/onsi/gomega v1.7.0
	github.com/prometheus/client_golang v1.1.0
	github.com/prometheus/client_model v0.0.0-20190812154241-14fe0d1b01d4
	github.com/prometheus/common v0.7.0
	github.com/prometheus/procfs v0.0.5 // indirect
	github.com/prometheus/prometheus v2.11.0+incompatible // pinned
	github.com/smartystreets/assertions v1.0.1 // indirect
	github.com/smartystreets/goconvey v0.0.0-20190731233626-505e41936337 // indirect
	github.com/square/certstrap v1.2.0 // indirect
	go.opencensus.io v0.22.1 // indirect
	golang.org/x/crypto v0.0.0-20190911031432-227b76d455e7 // indirect
	golang.org/x/net v0.0.0-20190921015927-1a5e07d1ff72 // indirect
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e // indirect
	golang.org/x/sys v0.0.0-20190920190810-ef0ce1748380 // indirect
	golang.org/x/time v0.0.0-20190921001708-c4c64cad1fd0 // indirect
	google.golang.org/api v0.10.0 // indirect
	google.golang.org/appengine v1.6.3 // indirect
	google.golang.org/genproto v0.0.0-20190916214212-f660b8655731 // indirect
	google.golang.org/grpc v1.23.1
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	gopkg.in/yaml.v2 v2.2.2
)

replace (
	code.cloudfoundry.org/loggregator-agent => code.cloudfoundry.org/loggregator-agent-release/src v0.0.0-20190828205358-fd77eb91324d
	github.com/prometheus/common => github.com/prometheus/common v0.5.0
)
