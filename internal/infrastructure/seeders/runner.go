package seeders

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports/infrastructure"
)

// Seeder is the interface for all seeders
type Seeder interface {
	Name() string
	Seed(ctx context.Context, opts Options) (*SeedResult, error)
	Reset(ctx context.Context, opts Options) (*SeedResult, error)
}

// Repositories contains repository dependencies for seeders
type Repositories interface {
	// WorkflowTemplateRepository returns the workflow template repository
	GetWorkflowTemplateRepository() any
	// StageTemplateRepository returns the stage template repository
	GetStageTemplateRepository() any
	// ActivityTemplateRepository returns the activity template repository
	GetActivityTemplateRepository() any
}

// Runner orchestrates multiple seeders
type Runner struct {
	seeders   []Seeder
	idService infrastructure.IDService
}

// NewRunner creates a new seeder runner
func NewRunner(idService infrastructure.IDService) *Runner {
	return &Runner{
		seeders:   []Seeder{},
		idService: idService,
	}
}

// RegisterSeeder adds a seeder to the runner
func (r *Runner) RegisterSeeder(s Seeder) {
	r.seeders = append(r.seeders, s)
}

// Run executes all seeders
func (r *Runner) Run(ctx context.Context, opts Options) ([]*SeedResult, error) {
	var results []*SeedResult

	for _, seeder := range r.seeders {
		var result *SeedResult
		var err error

		if opts.Reset {
			result, err = seeder.Reset(ctx, opts)
		} else {
			result, err = seeder.Seed(ctx, opts)
		}

		if err != nil {
			return results, fmt.Errorf("%s: %w", seeder.Name(), err)
		}

		results = append(results, result)
	}

	return results, nil
}

// GetIDService returns the ID service for seeders to use
func (r *Runner) GetIDService() infrastructure.IDService {
	return r.idService
}
