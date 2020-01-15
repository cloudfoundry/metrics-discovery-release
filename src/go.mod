module code.cloudfoundry.org/metrics-discovery

go 1.12

require (
	cloud.google.com/go v0.51.0 // indirect
	code.cloudfoundry.org/go-diodes v0.0.0-20190809170250-f77fb823c7ee
	code.cloudfoundry.org/go-envstruct v1.5.0
	code.cloudfoundry.org/go-loggregator v0.0.0-20190809213911-969cb33bee6a // pinned
	code.cloudfoundry.org/go-metric-registry v0.0.0-20191209165758-93cfd5e30bb0
	code.cloudfoundry.org/loggregator-agent v0.0.0-20190918193342-14308cf69de1
	code.cloudfoundry.org/tlsconfig v0.0.0-20200108215323-551ec42d1f74
	github.com/benjamintf1/unmarshalledmatchers v0.0.0-20190408201839-bb1c1f34eaea
	github.com/go-logfmt/logfmt v0.5.0 // indirect
	github.com/gogo/protobuf v1.3.1
	github.com/google/go-cmp v0.4.0 // indirect
	github.com/json-iterator/go v1.1.9 // indirect
	github.com/kr/pretty v0.2.0 // indirect
	github.com/nats-io/nats-server/v2 v2.1.2 // indirect
	github.com/nats-io/nats.go v1.9.1
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.8.1
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_golang v1.3.0
	github.com/prometheus/client_model v0.1.0
	github.com/prometheus/common v0.8.0
	github.com/prometheus/prometheus v2.13.1+incompatible // pinned
	golang.org/x/crypto v0.0.0-20200115085410-6d4e4cb37c7d // indirect
	golang.org/x/net v0.0.0-20200114155413-6afb5195e5aa // indirect
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d // indirect
	golang.org/x/sys v0.0.0-20200113162924-86b910548bc1 // indirect
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0 // indirect
	google.golang.org/genproto v0.0.0-20200113173426-e1de0a7b01eb // indirect
	google.golang.org/grpc v1.26.0
	gopkg.in/yaml.v2 v2.2.7
)

replace (
	code.cloudfoundry.org/loggregator-agent => code.cloudfoundry.org/loggregator-agent-release/src v0.0.0-20190828205358-fd77eb91324d
	github.com/prometheus/common => github.com/prometheus/common v0.7.0
)
