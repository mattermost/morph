package apply

import (
	"github.com/go-morph/morph"
	"github.com/go-morph/morph/sources"
)

func Migrate(dsn, source, driverName, path string, options ...morph.EngineOption) error {
	engine, err := initializeEngine(dsn, source, driverName, path, options...)
	if err != nil {
		return err
	}

	return engine.ApplyAll()
}

func Up(limit int, dsn, source, driverName, path string, options ...morph.EngineOption) (int, error) {
	engine, err := initializeEngine(dsn, source, driverName, path, options...)
	if err != nil {
		return -1, err
	}

	return engine.Apply(limit)
}

func Down(limit int, dsn, source, driverName, path string, options ...morph.EngineOption) (int, error) {
	engine, err := initializeEngine(dsn, source, driverName, path, options...)
	if err != nil {
		return -1, err
	}

	return engine.ApplyDown(limit)
}

func initializeEngine(dsn, source, driverName, path string, options ...morph.EngineOption) (*morph.Morph, error) {
	src, err := sources.Open(source, path)
	if err != nil {
		return nil, err
	}
	defer src.Close()

	engine, err := morph.NewFromConnURL(dsn, src, driverName, options...)
	if err != nil {
		return nil, err
	}

	return engine, err
}
