package target

import (
	"github.com/prometheus/prometheus/config"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"path/filepath"
	"sync"
	"time"
)

type targetsConfig struct {
	ScrapeConfigs []config.ScrapeConfig `yaml:"scrape_configs"`
}

type fileProvider struct {
	sync.Mutex
	configGlob      string
	refreshInterval time.Duration
	targets         []config.ScrapeConfig

	logger *log.Logger
}

func NewFileProvider(glob string, i time.Duration, logger *log.Logger) *fileProvider {
	return &fileProvider{
		configGlob:      glob,
		refreshInterval: i,
		logger:          logger,
	}
}

func (fp *fileProvider) Start() {
	ticker := time.NewTicker(fp.refreshInterval)

	fp.populateTargets()
	for range ticker.C {
		fp.populateTargets()
	}
}

func (fp *fileProvider) populateTargets() {
	fp.Lock()
	defer fp.Unlock()

	fp.targets = make([]config.ScrapeConfig, 0)
	files, err := filepath.Glob(fp.configGlob)
	if err != nil {
		fp.logger.Println("Unable to read downstream port location")
		return
	}

	for _, f := range files {
		yamlFile, err := ioutil.ReadFile(f)
		if err != nil {
			fp.logger.Printf("cannot read file: %s", err)
			continue
		}

		var t targetsConfig

		err = yaml.Unmarshal(yamlFile, &t)
		if err != nil {
			fp.logger.Printf("Unmarshal: %v", err)
			continue
		}

		fp.targets = append(fp.targets, t.ScrapeConfigs...)
	}
}

func (fp *fileProvider) GetTargets() []config.ScrapeConfig {
	fp.Lock()
	defer fp.Unlock()

	dst := make([]config.ScrapeConfig, len(fp.targets))
	copy(dst, fp.targets)

	return dst
}
