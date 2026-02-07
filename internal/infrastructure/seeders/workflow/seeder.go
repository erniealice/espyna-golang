package workflow

import (
	"context"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports/infrastructure"
	"leapfor.xyz/espyna/internal/infrastructure/seeders"
	"leapfor.xyz/vya"
)

// TemplateRepository defines the interface for workflow template persistence
type TemplateRepository interface {
	Create(ctx context.Context, data any) error
	FindBySystemID(ctx context.Context, systemID, workspaceID string) (any, error)
	DeleteSystemTemplates(ctx context.Context, workspaceID, businessType string) (int, error)
}

// StageTemplateRepository defines the interface for stage template persistence
type StageTemplateRepository interface {
	Create(ctx context.Context, data any) error
}

// ActivityTemplateRepository defines the interface for activity template persistence
type ActivityTemplateRepository interface {
	Create(ctx context.Context, data any) error
}

// Seeder handles workflow template seeding
type Seeder struct {
	templateRepo         TemplateRepository
	stageTemplateRepo    StageTemplateRepository
	activityTemplateRepo ActivityTemplateRepository
	idService            infrastructure.IDService
}

// New creates a new workflow seeder
func New(
	templateRepo TemplateRepository,
	stageTemplateRepo StageTemplateRepository,
	activityTemplateRepo ActivityTemplateRepository,
	idService infrastructure.IDService,
) *Seeder {
	return &Seeder{
		templateRepo:         templateRepo,
		stageTemplateRepo:    stageTemplateRepo,
		activityTemplateRepo: activityTemplateRepo,
		idService:            idService,
	}
}

// Name returns the seeder name
func (s *Seeder) Name() string {
	return "workflow"
}

// Seed populates the database with workflow templates
func (s *Seeder) Seed(ctx context.Context, opts seeders.Options) (*seeders.SeedResult, error) {
	result := &seeders.SeedResult{
		SeederName: s.Name(),
		Details:    []seeders.SeedDetail{},
	}

	// Load vya templates
	if err := vya.MustLoadSafe(); err != nil {
		return nil, fmt.Errorf("failed to load vya templates: %w", err)
	}

	// Get templates from vya
	var templates []*vya.WorkflowTemplate

	if opts.TemplateID != "" {
		// Seed specific template
		tmpl, ok := vya.Get(opts.TemplateID)
		if !ok {
			return nil, fmt.Errorf("template not found: %s", opts.TemplateID)
		}
		templates = []*vya.WorkflowTemplate{tmpl}
	} else if opts.BusinessType != "" {
		// Seed by business type
		templates = vya.GetByBusinessType(opts.BusinessType)
	} else {
		// Seed all
		templates = vya.All()
	}

	// Seed each template
	for _, tmpl := range templates {
		detail := seeders.SeedDetail{
			ID:   tmpl.ID,
			Name: tmpl.Name,
		}

		if opts.DryRun {
			// Check if would create or skip
			existing, _ := s.templateRepo.FindBySystemID(ctx, tmpl.ID, opts.WorkspaceID)
			if existing != nil {
				detail.Action = "would_skip"
				detail.Reason = "already exists"
				result.Skipped++
			} else {
				detail.Action = "would_create"
				result.Created++
			}
			result.Details = append(result.Details, detail)
			continue
		}

		// Check if exists
		existing, _ := s.templateRepo.FindBySystemID(ctx, tmpl.ID, opts.WorkspaceID)
		if existing != nil {
			detail.Action = "skipped"
			detail.Reason = "already exists"
			result.Skipped++
			result.Details = append(result.Details, detail)
			continue
		}

		// Create template hierarchy
		if err := s.createTemplateHierarchy(ctx, tmpl, opts.WorkspaceID); err != nil {
			detail.Action = "error"
			detail.Reason = err.Error()
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", tmpl.ID, err))
			result.Details = append(result.Details, detail)
			continue
		}

		detail.Action = "created"
		result.Created++
		result.Details = append(result.Details, detail)
	}

	return result, nil
}

// Reset deletes and re-seeds templates
func (s *Seeder) Reset(ctx context.Context, opts seeders.Options) (*seeders.SeedResult, error) {
	result := &seeders.SeedResult{
		SeederName: s.Name(),
	}

	if !opts.DryRun {
		// Delete existing system templates for this business type
		deleted, err := s.templateRepo.DeleteSystemTemplates(ctx, opts.WorkspaceID, string(opts.BusinessType))
		if err != nil {
			return nil, fmt.Errorf("failed to delete templates: %w", err)
		}
		result.Deleted = deleted
	}

	// Re-seed
	seedResult, err := s.Seed(ctx, opts)
	if err != nil {
		return result, err
	}

	result.Created = seedResult.Created
	result.Skipped = seedResult.Skipped
	result.Errors = seedResult.Errors
	result.Details = seedResult.Details

	return result, nil
}

func (s *Seeder) createTemplateHierarchy(ctx context.Context, tmpl *vya.WorkflowTemplate, workspaceID string) error {
	// Generate ID for workflow template
	templateID := s.idService.GenerateID()

	// Convert and create workflow template
	proto := ConvertWorkflowTemplate(tmpl, templateID, workspaceID)

	if err := s.templateRepo.Create(ctx, proto); err != nil {
		return fmt.Errorf("create workflow template: %w", err)
	}

	// Create stages
	for i, stageDef := range tmpl.Stages {
		stageID := s.idService.GenerateID()
		stageDef.OrderIndex = int32(i) // Ensure order index is set

		stageProto := ConvertStageTemplate(&stageDef, stageID, templateID)

		if err := s.stageTemplateRepo.Create(ctx, stageProto); err != nil {
			return fmt.Errorf("create stage template %s: %w", stageDef.Name, err)
		}

		// Create activities
		for j, activityDef := range stageDef.Activities {
			activityID := s.idService.GenerateID()
			activityDef.OrderIndex = int32(j) // Ensure order index is set

			activityProto := ConvertActivityTemplate(&activityDef, activityID, stageID)

			if err := s.activityTemplateRepo.Create(ctx, activityProto); err != nil {
				return fmt.Errorf("create activity template %s: %w", activityDef.Name, err)
			}
		}
	}

	return nil
}
