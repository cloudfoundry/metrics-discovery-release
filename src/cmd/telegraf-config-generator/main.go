package main

import (
	"code.cloudfoundry.org/metrics-discovery/internal/target"
	"encoding/json"
	"fmt"
	"github.com/influxdata/toml"
	"io/ioutil"
	"os"
)

type AgentConfig struct {
	MetricBufferLimit int `toml:"metric_buffer_limit"`
	MetricBatchSize   int `toml:"metric_batch_size"`
}

type TelegrafConfig struct {
	Agent   AgentConfig                       `toml:"agent"`
	Inputs  map[string]*PromInputConfig       `toml:"inputs"`
	Outputs map[string]*WavefrontOutputConfig `toml:"outputs"`
}

type PromInputConfig struct {
	// An array of urls to scrape metrics from.
	URLs []string `toml:"urls"`

	TLSCA              string `toml:"tls_ca"`
	TLSCert            string `toml:"tls_cert"`
	TLSKey             string `toml:"tls_key"`
	InsecureSkipVerify bool   `toml:"insecure_skip_verify"`
}

type WavefrontOutputConfig struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
}

func main() {
	b, err := ioutil.ReadFile("/var/vcap/data/scrape-config-generator/scrape_targets.json")
	if err != nil {
		panic(fmt.Errorf("cannot read file: %s", err))
	}

	var targets []*target.Target
	err = json.Unmarshal(b, &targets)
	if err != nil {
		panic(err)
	}

	inputCfg := &PromInputConfig{
		TLSCA:              "/var/vcap/jobs/prom_scraper/config/certs/scrape_ca.crt",
		TLSCert:            "/var/vcap/jobs/prom_scraper/config/certs/scrape.crt",
		TLSKey:             "/var/vcap/jobs/prom_scraper/config/certs/scrape.key",
		InsecureSkipVerify: true,
	}

	for _, t := range targets {
		url := fmt.Sprintf("https://%s/metrics", t.Targets[0])
		if id, ok := t.Labels["__param_id"]; ok {
			url += "?id=" + id
		}

		inputCfg.URLs = append(inputCfg.URLs, url)
	}

	cfg := TelegrafConfig{
		Agent: AgentConfig{
			// Need to increase buffer size due to scraped metrics all coming in at once
			MetricBufferLimit: 100000,
			MetricBatchSize:   10000,
		},
		Inputs: map[string]*PromInputConfig{
			"prometheus": inputCfg,
		},
		Outputs: map[string]*WavefrontOutputConfig{
			"wavefront": {
				Host: "localhost",
				Port: 2878, // Wavefront proxy default
			},
		},
	}

	cfgBytes, err := toml.Marshal(cfg)
	if err != nil {
		panic(err)
	}

	// Add processors - struggled with toml marshaling so it's just a string :/
	cfgBytes = append(cfgBytes, []byte(regexConf)...)

	err = ioutil.WriteFile("/var/vcap/data/scrape-config-generator/telegraf.toml", cfgBytes, os.ModePerm)
	if err != nil {
		panic(err)
	}
}

// Pull source from url or source_id tags
var regexConf = `
[[processors.regex]]
[[processors.regex.tags]]
  key = "url"
  pattern = ".*id=(\\w+).*"
  replacement = "${1}"
  result_key = "source"

[[processors.regex.tags]]
  key = "source_id"
  pattern = "(.*)"
  replacement = "${1}"
  result_key = "source"
`
