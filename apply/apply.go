package apply

import (
	"github.com/go-morph/morph"
	"github.com/go-morph/morph/sources"
)

func Migrate(dsn, source, driverName, path string) error {
	engine, err := initializeEngine(dsn, source, driverName, path)
	if err != nil {
		return err
	}

	return engine.ApplyAll()
}

func Up(limit int, dsn, source, driverName, path string) (int, error) {
	engine, err := initializeEngine(dsn, source, driverName, path)
	if err != nil {
		return -1, err
	}

	return engine.Apply(limit)
}

func Down(limit int, dsn, source, driverName, path string) (int, error) {
	engine, err := initializeEngine(dsn, source, driverName, path)
	if err != nil {
		return -1, err
	}

	return engine.ApplyDown(limit)
}

func initializeEngine(dsn, source, driverName, path string) (*morph.Morph, error) {
	src, err := sources.Open(source, path)
	if err != nil {
		return nil, err
	}
	defer src.Close()

	engine, err := morph.NewFromConnURL(dsn, src, driverName)
	if err != nil {
		return nil, err
	}

	return engine, err
}
