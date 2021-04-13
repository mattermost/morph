package sources

import (
	"fmt"
	"net/url"
	"sync"

	"github.com/go-morph/morph/models"
)

var sourcesMu sync.RWMutex
var registeredSources = make(map[string]Source)

type Source interface {
	Open(sourceURL string) (source Source, err error)
	Close() (err error)
	Migrations() (migrations []*models.Migration)
}

func Register(name string, source Source) {
	sourcesMu.Lock()
	defer sourcesMu.Unlock()

	registeredSources[name] = source
}

func List() []string {
	sourcesMu.Lock()
	defer sourcesMu.Unlock()

	sources := make([]string, 0, len(registeredSources))
	for source := range registeredSources {
		sources = append(sources, source)
	}

	return sources
}

func Open(sourceURL string) (Source, error) {
	uri, err := url.Parse(sourceURL)
	if err != nil {
		return nil, fmt.Errorf("unsupported source scheme found: %w", err)
	}

	sourcesMu.RLock()
	source, ok := registeredSources[uri.Scheme]
	sourcesMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unsupported source %q found", uri.Scheme)
	}

	return source.Open(sourceURL)
}
