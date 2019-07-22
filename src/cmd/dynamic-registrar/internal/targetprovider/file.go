package targetprovider

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"path/filepath"
	"sync"
	"time"
)

type fileProvider struct {
	sync.Mutex
	configGlob      string
	refreshInterval time.Duration
	targets []string
}

func NewFileProvider(glob string, i time.Duration) *fileProvider {
	return &fileProvider{
		configGlob:      glob,
		refreshInterval: i,
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

	fp.targets = make([]string, 0)
	files, err := filepath.Glob(fp.configGlob)
	if err != nil {
		log.Fatal("Unable to read downstream port location")
	}

	for _, f := range files {
		yamlFile, err := ioutil.ReadFile(f)
		if err != nil {
			log.Fatalf("cannot read file: %s", err)
		}

		var r struct {
			Routes []string `yaml:"routes"`
		}

		err = yaml.Unmarshal(yamlFile, &r)
		if err != nil {
			log.Fatalf("Unmarshal: %v", err)
		}
		fp.targets = append(fp.targets, r.Routes...)
	}

}

func (fp *fileProvider) GetTargets() []string {
	fp.Lock()
	defer fp.Unlock()

	dst := make([]string, len(fp.targets))
	copy(dst, fp.targets)

	return dst
}
