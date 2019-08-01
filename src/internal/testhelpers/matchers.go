package testhelpers

import (
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
	sd_config "github.com/prometheus/prometheus/discovery/config"
	"github.com/prometheus/prometheus/discovery/dns"
)

func HaveDNSConfig(domain, dnsType string, port int) types.GomegaMatcher {
	return WithTransform(
		func(sdConfig sd_config.ServiceDiscoveryConfig) []*dns.SDConfig {
			return sdConfig.DNSSDConfigs
		}, ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
			"Names": ConsistOf(domain),
			"Type":  Equal(dnsType),
			"Port":  Equal(port),
		}))))
}
