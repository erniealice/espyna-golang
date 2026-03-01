//go:build !workflow_templates && !workflow_http
// +build !workflow_templates,!workflow_http

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"github.com/erniealice/espyna-golang/internal/application/ports/infrastructure"
	dbinterfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	"github.com/erniealice/espyna-golang/internal/infrastructure/seeders"
	workflowseeder "github.com/erniealice/espyna-golang/internal/infrastructure/seeders/workflow"
	"leapfor.xyz/vya"
)

/*
ESPYNA SEEDER CLI - Workflow Template Seeder

This CLI seeds workflow templates from the vya package into the database.

Build Tags Required:
  go run -tags firestore,mock_auth,mock_storage,google_uuidv7 ./cmd/seeder/... [flags]

Usage:
  seeder [flags]

Flags:
  -workspace     Workspace ID (required)
  -business-type Business type filter (e.g., education)
  -template      Specific template ID to seed
  -reset         Delete existing templates before seeding
  -dry-run       Preview changes without persisting
  -verbose       Enable verbose output
  -list          List available templates
  -env           Path to .env file (default: .env)

Examples:
  # List all available templates
  go run -tags firestore,mock_auth,mock_storage,google_uuidv7 ./cmd/seeder/... -list

  # Dry run for education templates
  go run -tags firestore,mock_auth,mock_storage,google_uuidv7 ./cmd/seeder/... -workspace ws_123 -business-type education -dry-run

  # Seed all education templates
  go run -tags firestore,mock_auth,mock_storage,google_uuidv7 ./cmd/seeder/... -workspace ws_123 -business-type education
*/

func main() {
	// Define flags
	workspaceID := flag.String("workspace", "", "Workspace ID (required for seeding)")
	businessType := flag.String("business-type", "", "Business type filter")
	templateID := flag.String("template", "", "Specific template ID to seed")
	reset := flag.Bool("reset", false, "Delete existing templates before seeding")
	dryRun := flag.Bool("dry-run", false, "Preview changes without persisting")
	verbose := flag.Bool("verbose", false, "Enable verbose output")
	list := flag.Bool("list", false, "List available templates")
	envFile := flag.String("env", ".env", "Path to .env file")

	flag.Parse()

	// Load .env file
	if err := loadEnvFile(*envFile); err != nil {
		fmt.Printf("Warning: Could not load %s: %v\n", *envFile, err)
	}

	// Handle list command (no database needed)
	if *list {
		listTemplates(*businessType, *verbose)
		return
	}

	// Validate workspace for seeding operations
	if *workspaceID == "" {
		fmt.Fprintln(os.Stderr, "Error: -workspace is required")
		flag.Usage()
		os.Exit(1)
	}

	// Build options
	opts := seeders.Options{
		WorkspaceID:  *workspaceID,
		BusinessType: vya.BusinessType(*businessType),
		TemplateID:   *templateID,
		Reset:        *reset,
		DryRun:       *dryRun,
		Verbose:      *verbose,
	}

	// Run the seeder
	if err := runSeeder(opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runSeeder(opts seeders.Options) error {
	fmt.Println("Espyna Seeder CLI")
	fmt.Println("==================")
	fmt.Printf("Workspace:     %s\n", opts.WorkspaceID)
	fmt.Printf("Business Type: %s\n", opts.BusinessType)
	fmt.Printf("Template ID:   %s\n", opts.TemplateID)
	fmt.Printf("Reset:         %t\n", opts.Reset)
	fmt.Printf("Dry Run:       %t\n", opts.DryRun)
	fmt.Println()

	// Load vya templates
	if err := vya.MustLoadSafe(); err != nil {
		return fmt.Errorf("loading vya templates: %w", err)
	}

	// Get templates to seed
	var templates []*vya.WorkflowTemplate
	if opts.TemplateID != "" {
		tmpl, ok := vya.Get(opts.TemplateID)
		if !ok {
			return fmt.Errorf("template not found: %s", opts.TemplateID)
		}
		templates = []*vya.WorkflowTemplate{tmpl}
	} else if opts.BusinessType != "" {
		templates = vya.GetByBusinessType(opts.BusinessType)
	} else {
		templates = vya.All()
	}

	fmt.Printf("Templates to process: %d\n", len(templates))
	for _, t := range templates {
		stageCount := len(t.Stages)
		activityCount := 0
		for _, s := range t.Stages {
			activityCount += len(s.Activities)
		}
		fmt.Printf("  - %s (%d stages, %d activities)\n", t.ID, stageCount, activityCount)
	}
	fmt.Println()

	// If dry-run, just show what would happen
	if opts.DryRun {
		fmt.Println("[DRY RUN] The following templates would be seeded:")
		for _, t := range templates {
			fmt.Printf("  - %s: %s\n", t.ID, t.Name)
		}
		fmt.Println("\nNo changes were made to the database.")
		return nil
	}

	// Create container from environment
	fmt.Println("Connecting to database...")
	container, err := NewSeederContainer()
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}
	defer container.Close()

	// Get ID service
	idProvider := container.GetIDProvider()
	if idProvider == nil {
		return fmt.Errorf("ID provider not available")
	}

	// Extract IDService from provider
	var idService infrastructure.IDService
	if idWrapper, ok := idProvider.(interface {
		GetIDService() infrastructure.IDService
	}); ok {
		idService = idWrapper.GetIDService()
	}
	if idService == nil {
		fmt.Println("Warning: Using NoOp ID service (IDs will be timestamp-based)")
		idService = infrastructure.NewNoOpIDService()
	}

	// Get database operations
	dbOpsRaw := container.GetDatabaseOperations()
	if dbOpsRaw == nil {
		return fmt.Errorf("database operations not available")
	}

	// Cast to the DatabaseOperation interface
	dbOps, ok := dbOpsRaw.(dbinterfaces.DatabaseOperation)
	if !ok {
		return fmt.Errorf("database operations doesn't implement DatabaseOperation interface")
	}

	// Get table config for collection names
	tableConfig := container.GetDBTableConfig()
	if tableConfig == nil {
		return fmt.Errorf("database table config not available")
	}

	fmt.Printf("Using collections:\n")
	fmt.Printf("  - WorkflowTemplate: %s\n", tableConfig.WorkflowTemplate)
	fmt.Printf("  - StageTemplate:    %s\n", tableConfig.StageTemplate)
	fmt.Printf("  - ActivityTemplate: %s\n", tableConfig.ActivityTemplate)
	fmt.Println()

	// Seed templates using database operations
	ctx := context.Background()
	created := 0
	var seedErrors []string

	for _, tmpl := range templates {
		if opts.Verbose {
			fmt.Printf("Processing: %s\n", tmpl.ID)
		}

		// Generate IDs and create records
		templateRecordID := idService.GenerateID()

		// Convert and create workflow template
		workflowProto := workflowseeder.ConvertWorkflowTemplate(tmpl, templateRecordID, opts.WorkspaceID)
		workflowData, err := protoToMap(workflowProto)
		if err != nil {
			seedErrors = append(seedErrors, fmt.Sprintf("%s: proto conversion error: %v", tmpl.ID, err))
			continue
		}

		_, err = dbOps.Create(ctx, tableConfig.WorkflowTemplate, workflowData)
		if err != nil {
			seedErrors = append(seedErrors, fmt.Sprintf("%s: %v", tmpl.ID, err))
			continue
		}

		// Create stages
		stageSuccess := true
		for i, stageDef := range tmpl.Stages {
			stageID := idService.GenerateID()
			stageDef.OrderIndex = int32(i)
			stageProto := workflowseeder.ConvertStageTemplate(&stageDef, stageID, templateRecordID)
			stageData, err := protoToMap(stageProto)
			if err != nil {
				seedErrors = append(seedErrors, fmt.Sprintf("%s/stage/%s: proto conversion error: %v", tmpl.ID, stageDef.Name, err))
				stageSuccess = false
				continue
			}

			_, err = dbOps.Create(ctx, tableConfig.StageTemplate, stageData)
			if err != nil {
				seedErrors = append(seedErrors, fmt.Sprintf("%s/stage/%s: %v", tmpl.ID, stageDef.Name, err))
				stageSuccess = false
				continue
			}

			// Create activities
			for j, activityDef := range stageDef.Activities {
				activityID := idService.GenerateID()
				activityDef.OrderIndex = int32(j)
				activityProto := workflowseeder.ConvertActivityTemplate(&activityDef, activityID, stageID)
				activityData, err := protoToMap(activityProto)
				if err != nil {
					seedErrors = append(seedErrors, fmt.Sprintf("%s/stage/%s/activity/%s: proto conversion error: %v", tmpl.ID, stageDef.Name, activityDef.Name, err))
					continue
				}

				_, err = dbOps.Create(ctx, tableConfig.ActivityTemplate, activityData)
				if err != nil {
					seedErrors = append(seedErrors, fmt.Sprintf("%s/stage/%s/activity/%s: %v", tmpl.ID, stageDef.Name, activityDef.Name, err))
				}
			}
		}

		if stageSuccess {
			created++
			fmt.Printf("  ✓ Created: %s\n", tmpl.ID)
		} else {
			fmt.Printf("  ⚠ Partial: %s (some stages/activities failed)\n", tmpl.ID)
		}
	}

	// Print summary
	fmt.Println()
	fmt.Println("Summary")
	fmt.Println("=======")
	fmt.Printf("Created: %d\n", created)
	if len(seedErrors) > 0 {
		fmt.Printf("Errors:  %d\n", len(seedErrors))
		for _, e := range seedErrors {
			fmt.Printf("  - %s\n", e)
		}
	}

	return nil
}

func listTemplates(businessType string, verbose bool) {
	if err := vya.MustLoadSafe(); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading templates: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Available Workflow Templates")
	fmt.Println("============================")

	var templates []*vya.WorkflowTemplate
	if businessType != "" {
		templates = vya.GetByBusinessType(vya.BusinessType(businessType))
		fmt.Printf("Filtering by business type: %s\n\n", businessType)
	} else {
		templates = vya.All()
	}

	if len(templates) == 0 {
		fmt.Println("No templates found.")
		return
	}

	for _, t := range templates {
		fmt.Printf("ID:           %s\n", t.ID)
		fmt.Printf("Name:         %s\n", t.Name)
		fmt.Printf("Business:     %s\n", t.BusinessType)
		fmt.Printf("Version:      v%d\n", t.Version)
		fmt.Printf("Stages:       %d\n", len(t.Stages))

		if verbose {
			fmt.Printf("Description:  %s\n", t.Description)
			fmt.Printf("Category:     %s\n", t.Category)
			if len(t.Tags) > 0 {
				fmt.Printf("Tags:         %v\n", t.Tags)
			}
			fmt.Println("Stages:")
			for i, s := range t.Stages {
				fmt.Printf("  %d. %s (%s) - %d activities\n", i+1, s.Name, s.StageType, len(s.Activities))
				if verbose {
					for j, a := range s.Activities {
						fmt.Printf("     %d.%d %s (%s)\n", i+1, j+1, a.Name, a.ActivityType)
					}
				}
			}
		}
		fmt.Println()
	}

	fmt.Printf("Total: %d templates\n", len(templates))
}

// loadEnvFile loads environment variables from a file
func loadEnvFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		// Only set if not already set
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}

	return scanner.Err()
}

// protoToMap converts a protobuf message to a map[string]any
func protoToMap(msg proto.Message) (map[string]any, error) {
	// Use protojson to convert to JSON first
	marshaler := protojson.MarshalOptions{
		UseProtoNames:   true, // Use snake_case field names
		EmitUnpopulated: false,
	}

	jsonBytes, err := marshaler.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal proto to JSON: %w", err)
	}

	// Then unmarshal JSON to map
	var result map[string]any
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	return result, nil
}
