package target

import (
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v2"
)

type fileProvider struct {
	sync.Mutex
	configGlob      string
	refreshInterval time.Duration
	targets         []*Target

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

	fp.targets = make([]*Target, 0)
	files, err := filepath.Glob(fp.configGlob)
	if err != nil {
		fp.logger.Println("Unable to read downstream port location")
		return
	}

	for _, f := range files {
		yamlFile, err := os.ReadFile(f)
		if err != nil {
			fp.logger.Printf("cannot read file: %s", err)
			continue
		}

		var targets []*Target
		err = yaml.Unmarshal(yamlFile, &targets)
		if err != nil {
			fp.logger.Printf("Unmarshal: %v", err)
			continue
		}

		for _, t := range targets {
			if t.Source == "" {
				fp.logger.Printf("Target from %s is missing source", f)
				continue
			}

			fp.targets = append(fp.targets, t)
		}
	}
}

func (fp *fileProvider) GetTargets() []*Target {
	fp.Lock()
	defer fp.Unlock()

	dst := make([]*Target, len(fp.targets))
	copy(dst, fp.targets)

	return dst
}
