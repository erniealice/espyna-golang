//go:build workflow_templates
// +build workflow_templates

// Package main implements a workflow template seeder for the subscription checkout flow.
//
// Usage:
//
//	go run -tags "workflow_templates" cmd/seeder/workflow_templates.go
//
// Environment Variables (from .env):
//   - CONFIG_DATABASE_PROVIDER=firestore
//   - FIRESTORE_PROJECT_ID=your-project-id
//   - FIRESTORE_CREDENTIALS_PATH=path/to/service-account.json
//   - FIRESTORE_DATABASE=(optional) database name
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"leapfor.xyz/espyna/consumer"
	activitytemplatepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/activity_template"
	stagetemplatepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/stage_template"
	workflowtemplatepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/workflow_template"
)

const (
	// WorkflowTemplateID is the unique identifier for the subscription checkout workflow
	WorkflowTemplateID = "generic:submit_subscription_checkout:v1"
)

func main() {
	log.Println("🌱 Workflow Template Seeder")
	log.Println("============================")

	// Create container from environment
	container := consumer.NewContainerFromEnv()
	if container == nil {
		log.Fatal("❌ Failed to create container from environment")
	}

	// Initialize container (sets up use cases, repositories, etc.)
	if err := container.Initialize(); err != nil {
		log.Fatalf("❌ Failed to initialize container: %v", err)
	}
	defer container.Close()

	log.Println("✅ Container initialized")

	ctx := context.Background()
	useCases := container.GetUseCases()

	if useCases == nil || useCases.Workflow == nil {
		log.Fatal("❌ Workflow use cases not available")
	}

	// Seed the workflow template
	if err := seedWorkflowTemplate(ctx, useCases); err != nil {
		log.Fatalf("❌ Failed to seed workflow template: %v", err)
	}

	log.Println("✅ Workflow template seeded successfully!")
	log.Println("")
	log.Println("📋 Template Details:")
	log.Printf("   ID: %s", WorkflowTemplateID)
	log.Println("   Activities:")
	log.Println("     1. Read Price Plan (subscription.price_plan.read)")
	log.Println("     2. Find or Create Client (entity.client.find_or_create)")
	log.Println("     3. Create Subscription (subscription.subscription.create)")
	log.Println("     4. Create Payment (payment.payment.create)")
	log.Println("     5. Create Checkout Session (integration.payment.create_checkout)")
}

func seedWorkflowTemplate(ctx context.Context, useCases *consumer.UseCases) error {
	// 1. Create WorkflowTemplate
	inputSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"first_name":      map[string]interface{}{"type": "string"},
			"last_name":       map[string]interface{}{"type": "string"},
			"email":           map[string]interface{}{"type": "string"},
			"contact_number":  map[string]interface{}{"type": "string"},
			"price_plan_id":   map[string]interface{}{"type": "string"},
			"affiliate_code":  map[string]interface{}{"type": "string"},
			"conversion_rate": map[string]interface{}{"type": "number"},
			"base_url":        map[string]interface{}{"type": "string"},
		},
		"required": []string{"first_name", "last_name", "email", "contact_number", "price_plan_id"},
	}

	inputSchemaJSON, _ := json.Marshal(inputSchema)

	workflowTmpl := &workflowtemplatepb.WorkflowTemplate{
		Id:              WorkflowTemplateID,
		Name:            "Submit Subscription with Checkout",
		Description:     stringPtr("Create subscription, payment, and checkout session in one automated flow"),
		BusinessType:    "generic",
		Version:         int32Ptr(1),
		IsSystem:        boolPtr(true),
		InputSchemaJson: stringPtr(string(inputSchemaJSON)),
		Status:          "active",
		Active:          true,
	}

	log.Println("📝 Creating WorkflowTemplate...")
	_, err := useCases.Workflow.WorkflowTemplate.CreateWorkflowTemplate.Execute(ctx, &workflowtemplatepb.CreateWorkflowTemplateRequest{
		Data: workflowTmpl,
	})
	if err != nil {
		return fmt.Errorf("create workflow template: %w", err)
	}
	log.Printf("   ✅ Created: %s", workflowTmpl.Id)

	// 2. Create StageTemplates
	// We'll use a single stage for all activities since they're all automated
	stageTmpl := &stagetemplatepb.StageTemplate{
		Id:                 WorkflowTemplateID + ":stage:0",
		WorkflowTemplateId: WorkflowTemplateID,
		Name:               "Subscription Checkout",
		OrderIndex:         int32Ptr(0),
		StageType:          "automated",
		IsRequired:         boolPtr(true),
	}

	log.Println("📝 Creating StageTemplate...")
	_, err = useCases.Workflow.StageTemplate.CreateStageTemplate.Execute(ctx, &stagetemplatepb.CreateStageTemplateRequest{
		Data: stageTmpl,
	})
	if err != nil {
		return fmt.Errorf("create stage template: %w", err)
	}
	log.Printf("   ✅ Created: %s", stageTmpl.Id)

	// 3. Create ActivityTemplates
	activities := []ActivityTemplateDef{
		{
			ID:          WorkflowTemplateID + ":stage:0:activity:0",
			Name:        "Read Price Plan",
			OrderIndex:  0,
			UseCaseCode: "subscription.price_plan.read",
			InputSchema: map[string]interface{}{
				"id": map[string]interface{}{
					"source": "price_plan_id",
					"type":   "string",
				},
			},
			OutputSchema: map[string]interface{}{
				"price_plan_amount": map[string]interface{}{
					"source": "amount",
					"type":   "number",
				},
				"price_plan_name": map[string]interface{}{
					"source": "name",
					"type":   "string",
				},
				"price_plan_plan_id": map[string]interface{}{
					"source": "plan_id",
					"type":   "string",
				},
			},
		},
		{
			ID:          WorkflowTemplateID + ":stage:0:activity:1",
			Name:        "Find or Create Client",
			OrderIndex:  1,
			UseCaseCode: "entity.client.find_or_create",
			InputSchema: map[string]interface{}{
				"user.first_name": map[string]interface{}{
					"source": "first_name",
					"type":   "string",
				},
				"user.last_name": map[string]interface{}{
					"source": "last_name",
					"type":   "string",
				},
				"user.email_address": map[string]interface{}{
					"source": "email",
					"type":   "string",
				},
			},
			OutputSchema: map[string]interface{}{
				"client_id": map[string]interface{}{
					"source": "client_id",
					"type":   "string",
				},
			},
		},
		{
			ID:          WorkflowTemplateID + ":stage:0:activity:2",
			Name:        "Create Subscription",
			OrderIndex:  2,
			UseCaseCode: "subscription.subscription.create",
			InputSchema: map[string]interface{}{
				"name": map[string]interface{}{
					"source":  "first_name",
					"type":    "string",
					"default": "Subscription",
				},
				"price_plan_id": map[string]interface{}{
					"source": "price_plan_id",
					"type":   "string",
				},
				"client_id": map[string]interface{}{
					"source": "client_id",
					"type":   "string",
				},
			},
			OutputSchema: map[string]interface{}{
				"subscription_id": map[string]interface{}{
					"source": "subscription_id",
					"type":   "string",
				},
			},
		},
		{
			ID:          WorkflowTemplateID + ":stage:0:activity:3",
			Name:        "Create Payment",
			OrderIndex:  3,
			UseCaseCode: "payment.payment.create",
			InputSchema: map[string]interface{}{
				"name": map[string]interface{}{
					"source":  "first_name",
					"type":    "string",
					"default": "Payment",
				},
				"subscription_id": map[string]interface{}{
					"source": "subscription_id",
					"type":   "string",
				},
				"amount": map[string]interface{}{
					"source": "price_plan_amount",
					"type":   "number",
				},
				"status": map[string]interface{}{
					"source":  "status",
					"type":    "string",
					"default": "pending",
				},
			},
			OutputSchema: map[string]interface{}{
				"payment_id": map[string]interface{}{
					"source": "payment_id",
					"type":   "string",
				},
			},
		},
		{
			ID:          WorkflowTemplateID + ":stage:0:activity:4",
			Name:        "Create Checkout Session",
			OrderIndex:  4,
			UseCaseCode: "integration.payment.create_checkout",
			InputSchema: map[string]interface{}{
				"amount": map[string]interface{}{
					"source": "price_plan_amount",
					"type":   "number",
				},
				"currency": map[string]interface{}{
					"source":  "currency",
					"type":    "string",
					"default": "PHP",
				},
				"description": map[string]interface{}{
					"source":  "first_name",
					"type":    "string",
					"default": "Subscription payment",
				},
				"payment_id": map[string]interface{}{
					"source": "payment_id",
					"type":   "string",
				},
				"subscription_id": map[string]interface{}{
					"source": "subscription_id",
					"type":   "string",
				},
				"client_id": map[string]interface{}{
					"source": "client_id",
					"type":   "string",
				},
				"order_ref": map[string]interface{}{
					"source": "payment_id",
					"type":   "string",
				},
				"success_url": map[string]interface{}{
					"source":  "base_url",
					"type":    "string",
					"default": "/payment/success",
				},
				"failure_url": map[string]interface{}{
					"source":  "base_url",
					"type":    "string",
					"default": "/payment/fail",
				},
				"cancel_url": map[string]interface{}{
					"source":  "base_url",
					"type":    "string",
					"default": "/payment/cancel",
				},
				"customer_email": map[string]interface{}{
					"source": "email",
					"type":   "string",
				},
				"customer_name": map[string]interface{}{
					"source":  "first_name",
					"type":    "string",
					"default": "Customer",
				},
				"customer_phone": map[string]interface{}{
					"source": "contact_number",
					"type":   "string",
				},
			},
			OutputSchema: map[string]interface{}{
				"checkout_session": map[string]interface{}{
					"source": "session",
					"type":   "any",
				},
			},
		},
	}

	for _, actDef := range activities {
		inputJSON, _ := json.Marshal(actDef.InputSchema)
		outputJSON, _ := json.Marshal(actDef.OutputSchema)

		actTmpl := &activitytemplatepb.ActivityTemplate{
			Id:               actDef.ID,
			StageTemplateId:  stageTmpl.Id,
			Name:             actDef.Name,
			OrderIndex:       int32Ptr(int32(actDef.OrderIndex)),
			ActivityType:     "automated",
			IsRequired:       boolPtr(true),
			UseCaseCode:      &actDef.UseCaseCode,
			InputSchemaJson:  stringPtr(string(inputJSON)),
			OutputSchemaJson: stringPtr(string(outputJSON)),
		}

		log.Printf("📝 Creating ActivityTemplate: %s...", actDef.Name)
		_, err := useCases.Workflow.ActivityTemplate.CreateActivityTemplate.Execute(ctx, &activitytemplatepb.CreateActivityTemplateRequest{
			Data: actTmpl,
		})
		if err != nil {
			return fmt.Errorf("create activity template %s: %w", actDef.Name, err)
		}
		log.Printf("   ✅ Created: %s", actTmpl.Id)
	}

	return nil
}

// ActivityTemplateDef defines an activity template
type ActivityTemplateDef struct {
	ID           string
	Name         string
	OrderIndex   int
	UseCaseCode  string
	InputSchema  map[string]interface{}
	OutputSchema map[string]interface{}
}

func int32Ptr(i int32) *int32 {
	return &i
}

func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}
