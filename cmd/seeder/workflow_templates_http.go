//go:build workflow_http
// +build workflow_http

// Package main implements a workflow template seeder for the subscription checkout flow.
//
// This seeder directly uses Firestore without going through the consumer package
// to avoid build tag conflicts with HTTP adapters.
//
// Usage:
//
//	cd apps/tph-unlock-golang-v2
//	go run -tags "workflow_http,gin,firestore,google_uuidv7" ../packages/espyna/cmd/seeder/workflow_templates_http.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
)

const (
	// WorkflowTemplateID is the unique identifier for the subscription checkout workflow
	WorkflowTemplateID = "generic:submit_subscription_checkout:v1"

	// Firestore collection names (must match database.go config)
	WorkflowTemplateCollection = "workflow_template"
	StageTemplateCollection    = "stage_template"
	ActivityTemplateCollection = "activity_template"
)

func main() {
	// Load .env file from parent app directory
	if err := godotenv.Load("../../apps/tph-unlock-golang-v2/.env"); err != nil {
		log.Println("⚠️  Warning: .env file not found, using system environment variables")
	}

	log.Println("🌱 Workflow Template Seeder (Direct Firestore)")
	log.Println("============================================")

	// Get Firestore credentials from environment
	projectID := os.Getenv("FIRESTORE_PROJECT_ID")
	credentialsPath := os.Getenv("FIRESTORE_CREDENTIALS_PATH")
	databaseID := os.Getenv("FIRESTORE_DATABASE")

	if projectID == "" {
		log.Fatal("❌ FIRESTORE_PROJECT_ID environment variable not set")
	}

	log.Printf("📦 Project: %s", projectID)
	if databaseID != "" {
		log.Printf("📦 Database: %s", databaseID)
	}

	// Create Firestore client
	ctx := context.Background()
	var client *firestore.Client
	var err error

	if credentialsPath != "" {
		client, err = firestore.NewClient(ctx, projectID, option.WithCredentialsFile(credentialsPath))
	} else {
		client, err = firestore.NewClient(ctx, projectID)
	}
	if err != nil {
		log.Fatalf("❌ Failed to create Firestore client: %v", err)
	}
	defer client.Close()

	log.Println("✅ Firestore client initialized")

	// Seed the workflow template
	if err := seedWorkflowTemplate(ctx, client); err != nil {
		log.Fatalf("❌ Failed to seed workflow template: %v", err)
	}

	log.Println("")
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

func seedWorkflowTemplate(ctx context.Context, client *firestore.Client) error {
	now := time.Now().UnixMilli()

	// 1. Create WorkflowTemplate
	// Workflow input schema uses simple SchemaField format
	inputSchema := map[string]interface{}{
		"first_name":      map[string]interface{}{"type": "string", "required": true},
		"last_name":       map[string]interface{}{"type": "string", "required": true},
		"email":           map[string]interface{}{"type": "string", "required": true},
		"contact_number":  map[string]interface{}{"type": "string", "required": true},
		"price_plan_id":   map[string]interface{}{"type": "string", "required": true},
		"affiliate_code":  map[string]interface{}{"type": "string"},
		"conversion_rate": map[string]interface{}{"type": "number", "default": 1.0},
		"base_url":        map[string]interface{}{"type": "string", "default": "http://localhost:8080"},
	}

	// Output schema is not used by SchemaProcessor, just for documentation
	_ = map[string]interface{}{
		"subscription_id":  map[string]interface{}{"type": "string"},
		"payment_id":       map[string]interface{}{"type": "string"},
		"checkout_session": map[string]interface{}{"type": "object"},
	}

	inputSchemaJSON, _ := json.Marshal(inputSchema)

	workflowTmpl := map[string]interface{}{
		"id":                WorkflowTemplateID,
		"name":              "Submit Subscription with Checkout",
		"description":       "Create subscription, payment, and checkout session in one automated flow",
		"business_type":     "generic",
		"version":           1,
		"is_system":         true,
		"status":            "active",
		"input_schema_json": string(inputSchemaJSON),
		"active":            true,
		"date_created":      now,
		"date_modified":     now,
	}

	log.Println("📝 Creating WorkflowTemplate...")
	_, err := client.Collection(WorkflowTemplateCollection).Doc(WorkflowTemplateID).Set(ctx, workflowTmpl)
	if err != nil {
		return fmt.Errorf("create workflow template: %w", err)
	}
	log.Printf("   ✅ Created: %s", WorkflowTemplateID)

	// 2. Create StageTemplate
	stageTmplID := WorkflowTemplateID + ":stage:0"
	stageTmpl := map[string]interface{}{
		"id":                   stageTmplID,
		"workflow_template_id": WorkflowTemplateID,
		"name":                 "Subscription Checkout",
		"order_index":          0,
		"stage_type":           "automated",
		"is_required":          true,
		"active":               true, // REQUIRED for List queries to work
		"date_created":         now,
		"date_modified":        now,
	}

	log.Println("📝 Creating StageTemplate...")
	_, err = client.Collection(StageTemplateCollection).Doc(stageTmplID).Set(ctx, stageTmpl)
	if err != nil {
		return fmt.Errorf("create stage template: %w", err)
	}
	log.Printf("   ✅ Created: %s", stageTmplID)

	// 3. Create ActivityTemplates
	activities := []struct {
		ID           string
		Name         string
		OrderIndex   int
		UseCaseCode  string
		InputSchema  map[string]interface{}
		OutputSchema map[string]interface{}
	}{
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
					"type":   "float64",
				},
				"price_plan_name": map[string]interface{}{
					"source": "name",
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
					"source": "id",
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
					"source": "id",
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
					"type":   "float64",
				},
				"status": map[string]interface{}{
					"type":    "string",
					"default": "pending",
				},
			},
			OutputSchema: map[string]interface{}{
				"payment_id": map[string]interface{}{
					"source": "id",
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
					"source":  "price_plan_amount",
					"type":    "float64",
					"default": 0.0,
				},
				"currency": map[string]interface{}{
					"type":    "string",
					"default": "PHP",
				},
				"description": map[string]interface{}{
					"source":  "first_name",
					"type":    "string",
					"default": "Subscription Checkout",
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
					"type":    "string",
					"default": "/payment/success",
				},
				"failure_url": map[string]interface{}{
					"type":    "string",
					"default": "/payment/fail",
				},
				"cancel_url": map[string]interface{}{
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
					"source": "checkout_url",
					"type":   "string",
				},
			},
		},
	}

	for _, actDef := range activities {
		inputJSON, _ := json.Marshal(actDef.InputSchema)
		outputJSON, _ := json.Marshal(actDef.OutputSchema)

		actTmpl := map[string]interface{}{
			"id":                 actDef.ID,
			"stage_template_id":  stageTmplID,
			"name":               actDef.Name,
			"order_index":        actDef.OrderIndex,
			"activity_type":      "automated",
			"is_required":        true,
			"active":             true, // REQUIRED for List queries to work
			"use_case_code":      actDef.UseCaseCode,
			"input_schema_json":  string(inputJSON),
			"output_schema_json": string(outputJSON),
			"date_created":       now,
			"date_modified":      now,
		}

		log.Printf("📝 Creating ActivityTemplate: %s...", actDef.Name)
		_, err := client.Collection(ActivityTemplateCollection).Doc(actDef.ID).Set(ctx, actTmpl)
		if err != nil {
			return fmt.Errorf("create activity template %s: %w", actDef.Name, err)
		}
		log.Printf("   ✅ Created: %s", actDef.ID)
	}

	return nil
}
