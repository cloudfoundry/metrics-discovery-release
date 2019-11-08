module code.cloudfoundry.org/metrics-discovery

go 1.12

require (
	cloud.google.com/go v0.47.0 // indirect
	code.cloudfoundry.org/go-diodes v0.0.0-20190809170250-f77fb823c7ee
	code.cloudfoundry.org/go-envstruct v1.5.0
	code.cloudfoundry.org/go-loggregator v0.0.0-20190809213911-969cb33bee6a // pinned
	code.cloudfoundry.org/go-metric-registry v0.0.0-20191004164645-33b67ef0f7d1
	code.cloudfoundry.org/loggregator-agent v0.0.0-20190918193342-14308cf69de1
	code.cloudfoundry.org/tlsconfig v0.0.0-20190710180242-462f72de1106
	github.com/benjamintf1/unmarshalledmatchers v0.0.0-20190408201839-bb1c1f34eaea
	github.com/gogo/protobuf v1.3.1
	github.com/golang/groupcache v0.0.0-20191027212112-611e8accdfc9 // indirect
	github.com/json-iterator/go v1.1.8 // indirect
	github.com/nats-io/nats-server/v2 v2.1.0 // indirect
	github.com/nats-io/nats.go v1.9.1
	github.com/nats-io/nkeys v0.1.2 // indirect
	github.com/onsi/ginkgo v1.10.3
	github.com/onsi/gomega v1.7.1
	github.com/prometheus/client_golang v1.2.1
	github.com/prometheus/client_model v0.0.0-20190812154241-14fe0d1b01d4
	github.com/prometheus/common v0.7.0
	github.com/prometheus/prometheus v2.13.1+incompatible // pinned
	github.com/square/certstrap v1.2.0 // indirect
	go.opencensus.io v0.22.1 // indirect
	golang.org/x/crypto v0.0.0-20191107222254-f4817d981bb6 // indirect
	golang.org/x/net v0.0.0-20191108174545-380dde419d29 // indirect
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e // indirect
	golang.org/x/sys v0.0.0-20191105231009-c1f44814a5cd // indirect
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0 // indirect
	google.golang.org/api v0.13.0 // indirect
	google.golang.org/appengine v1.6.5 // indirect
	google.golang.org/genproto v0.0.0-20191028173616-919d9bdd9fe6 // indirect
	google.golang.org/grpc v1.25.1
	gopkg.in/yaml.v2 v2.2.5
)

replace (
	code.cloudfoundry.org/loggregator-agent => code.cloudfoundry.org/loggregator-agent-release/src v0.0.0-20190828205358-fd77eb91324d
	github.com/prometheus/common => github.com/prometheus/common v0.7.0
)
