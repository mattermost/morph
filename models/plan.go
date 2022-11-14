package models

import "errors"

const CurrentPlanVersion = 1

var ErrInvalidPlanVersion = errors.New("invalid plan version")

type Plan struct {
	// Version is the version of the plan.
	Version int
	// Auto is the mode of the plan. If true, the plan will rollback automatically in case of an error.
	Auto bool
	// Migrations is the list of migrations to be applied.
	Migrations []*Migration
	// RevertMigrations is the list of migrations to be applied in case of an error.
	RevertMigrations []*Migration
}

func NewPlan(migrations, rollback []*Migration) *Plan {
	return &Plan{
		Version:          CurrentPlanVersion,
		Migrations:       migrations,
		RevertMigrations: rollback,
	}
}

func (p *Plan) Validate() error {
	if p.Version != CurrentPlanVersion {
		return ErrInvalidPlanVersion
	}

	return nil
}
