package foundation

import (
	"database/sql"
	"io/ioutil"
	"testing"

	"github.com/jmoiron/sqlx"
)

type Foundation struct {
	t            *testing.T
	currentStep  int
	stepByStep   bool
	migrator     Migrator
	db           *sqlx.DB
	interceptors map[int]func() error
}

type Migrator interface {
	DB() *sql.DB
	DriverName() string
	Setup() error
	MigrateToStep(step int) error
	Interceptors() map[int]func() error
	TearDown() error
}

// ToDo: change T to TB?
func New(t *testing.T, migrator Migrator) *Foundation {
	if err := migrator.Setup(); err != nil {
		t.Fatalf("error setting up the migrator: %s", err)
	}

	db := sqlx.NewDb(migrator.DB(), migrator.DriverName())

	return &Foundation{
		t:           t,
		currentStep: 0,
		// if true, will run the migrator Step function once per step
		// instead of just once with the final step
		stepByStep:   false,
		migrator:     migrator,
		interceptors: migrator.Interceptors(),
		db:           db,
	}
}

// RegisterInterceptors replaced the migrator interceptors with new
// ones, in case we want to check a special case for a given test
func (f *Foundation) RegisterInterceptors(interceptors map[int]func() error) *Foundation {
	f.interceptors = interceptors
	return f
}

func (f *Foundation) SetStepByStep(stepByStep bool) *Foundation {
	f.stepByStep = stepByStep
	return f
}

// calculateNextStep returns the next step in the chain that has an
// interceptor or the final step to migrate to
func (f *Foundation) calculateNextStep(step int) int {
	// should never happen
	if f.currentStep >= step {
		// nothing to do
		return step // ToDo: or 0? merge the two conditions
	}

	// if there are no interceptors, next step is directly the final
	// one
	if f.interceptors == nil {
		return step
	}

	i := f.currentStep
	for i < step {
		i++

		if _, ok := f.interceptors[i]; ok {
			break
		}
	}

	return i
}

func (f *Foundation) migrateToStep(step int, skipLastInterceptor bool) *Foundation {
	if step == f.currentStep {
		// log nothing to do
		return f
	}

	if step < f.currentStep {
		f.t.Fatal("Down migrations not supported yet")
	}

	// if there are no interceptors, just migrate to the last step
	if f.interceptors == nil {
		if err := f.doMigrateToStep(step); err != nil {
			f.t.Fatalf("migration to step %d failed: %s", step, err)
		}

		return f
	}

	for f.currentStep < step {
		nextStep := f.calculateNextStep(step)

		if err := f.doMigrateToStep(nextStep); err != nil {
			f.t.Fatalf("migration to step %d failed: %s", nextStep, err)
		}

		// if we want to skip the last interceptor and we're in the
		// last step, just continue
		if skipLastInterceptor && nextStep == step {
			continue
		}

		interceptorFn, ok := f.interceptors[nextStep]
		if ok {
			if err := interceptorFn(); err != nil {
				f.t.Fatalf("interceptor function for step %d failed: %s", nextStep, err)
			}
		}
	}

	return f
}

// MigrateToStep instructs the migrator to move forward until step is
// reached. While migrating, it will run the interceptors after the
// step they're defined for
func (f *Foundation) MigrateToStep(step int) *Foundation {
	return f.migrateToStep(step, false)
}

// MigrateToStepSkippingLastInterceptor instructs the migrator to move
// forward until step is reached, skipping the last interceptor. This
// is useful if we want to load fixtures on the last step but before
// running the interceptor code, so we can check how that data is
// modified by the interceptor
func (f *Foundation) MigrateToStepSkippingLastInterceptor(step int) *Foundation {
	return f.migrateToStep(step, true)
}

// RunInterceptor executes the code of the interceptor corresponding
// to step
func (f *Foundation) RunInterceptor(step int) *Foundation {
	interceptorFn, ok := f.interceptors[step]
	if !ok {
		f.t.Fatalf("no interceptor found for step %d", step)
	}

	if err := interceptorFn(); err != nil {
		f.t.Fatalf("interceptor function for step %d failed: %s", step, err)
	}

	return f
}

// doMigrateToStep executes the migrator function to migrate to a
// specific step and updates the foundation currentStep to reflect the
// result. This function doesn't take into account interceptors, that
// happens on MigrateToStep
func (f *Foundation) doMigrateToStep(step int) error {
	if f.stepByStep {
		for f.currentStep < step {
			if err := f.migrator.MigrateToStep(f.currentStep + 1); err != nil {
				return err
			}

			f.currentStep++
		}

		return nil
	}

	if err := f.migrator.MigrateToStep(step); err != nil {
		return err
	}

	f.currentStep = step
	return nil
}

func (f *Foundation) TearDown() {
	if err := f.migrator.TearDown(); err != nil {
		f.t.Fatalf("error tearing down migrator: %s", err)
	}
}

func (f *Foundation) DB() *sqlx.DB {
	return f.db
}

func (f *Foundation) Exec(s string) *Foundation {
	if _, err := f.DB().Exec(s); err != nil {
		f.t.Fatalf("failed to run %s: %s", s, err)
	}

	return f
}

func (f *Foundation) ExecFile(filePath string) *Foundation {
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		f.t.Fatalf("failed to read file %s: %s", filePath, err)
	}

	return f.Exec(string(b))
}
