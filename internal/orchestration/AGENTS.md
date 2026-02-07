# Orchestration Layer

## Purpose

The `orchestration/` directory contains **coordination and workflow execution infrastructure**. Unlike adapters (which connect to external systems) or providers (which manage infrastructure resources), orchestration handles the **dynamic composition and execution of use cases**.

This layer is conceptually distinct because it:
1. Coordinates **internal components** rather than external systems
2. Implements **saga patterns** for multi-step processes
3. Provides **runtime flexibility** through use case code mapping
4. Manages **workflow lifecycle** from creation to completion

## Directory Structure

```
orchestration/
└── engine/                              # Workflow engine use cases
    ├── usecases.go                      # Aggregates use cases, implements WorkflowEngineService
    ├── schema.go                        # Schema processing for input/output mapping
    ├── start_workflow_from_template.go  # Creates workflow instance from template
    ├── execute_activity.go              # Executes a single activity via registry
    ├── advance_workflow.go              # Moves workflow to next stage
    ├── continue_workflow.go             # Handles human input for paused workflows
    └── get_workflow_status.go           # Retrieves current workflow state
```

## Architectural Position

```
+-------------------------------------------------------------+
|                    Hexagonal Architecture                    |
+-------------------------------------------------------------+
|                                                             |
|  +-----------+    +-------------------+    +---------+      |
|  |  Primary  |--->|    Application    |<---|Secondary|      |
|  | Adapters  |    |    (Use Cases)    |    | Adapters|      |
|  |(HTTP/gRPC)|    +-------------------+    |(DB/Email)|     |
|  +-----------+            |                +---------+      |
|                           |                                 |
|                +----------v----------+                      |
|                |    Orchestration    | <-- ENGINE LAYER     |
|                |   (Workflow Engine) |                      |
|                +---------------------+                      |
|                                                             |
+-------------------------------------------------------------+
```

## Why Orchestration is Separate

| Aspect | Adapters | Providers | Orchestration |
|--------|----------|-----------|---------------|
| **Purpose** | Connect to external world | Create adapters dynamically | Coordinate use cases |
| **Direction** | Inbound/Outbound | Factory pattern | Internal coordination |
| **Lifecycle** | Request-scoped | App startup | Runtime dynamic |
| **Examples** | HTTP handlers, DB repos | PostgreSQL, Firebase | Workflow engine |

## Engine Architecture

### Core Components

The engine is built around **dependency injection** with two main structs:

```go
// EngineRepositories groups all repository dependencies
type EngineRepositories struct {
    Workflow         workflowpb.WorkflowDomainServiceServer
    WorkflowTemplate workflowtemplatepb.WorkflowTemplateDomainServiceServer
    Stage            stagepb.StageDomainServiceServer
    StageTemplate    stagetemplatepb.StageTemplateDomainServiceServer
    Activity         activitypb.ActivityDomainServiceServer
    ActivityTemplate activitytemplatepb.ActivityTemplateDomainServiceServer
}

// EngineServices groups all business service dependencies
type EngineServices struct {
    AuthorizationService ports.AuthorizationService
    TransactionService   ports.TransactionService
    TranslationService   ports.TranslationService
    IDService            ports.IDService
    ExecutorRegistry     ports.ExecutorRegistry  // Maps use_case_code -> executor
}
```

### EngineUseCases Aggregate

The `EngineUseCases` struct implements `ports.WorkflowEngineService`:

```go
type EngineUseCases struct {
    startWorkflowUC    *StartWorkflowFromTemplateUseCase
    executeActivityUC  *ExecuteActivityUseCase
    advanceWorkflowUC  *AdvanceWorkflowUseCase
    getStatusUC        *GetWorkflowStatusUseCase
    continueWorkflowUC *ContinueWorkflowUseCase
}
```

## Use Cases

### 1. StartWorkflowFromTemplate

Creates a workflow instance from a template with input validation.

**Flow:**
1. Fetch WorkflowTemplate by ID
2. Validate input against `input_schema_json`
3. Create Workflow instance with validated context
4. Create first Stage instance (lazy instantiation)

```go
func (uc *StartWorkflowFromTemplateUseCase) Execute(
    ctx context.Context,
    req *enginepb.StartWorkflowRequest,
) (*enginepb.StartWorkflowResponse, error)
```

### 2. ExecuteActivity

Executes a single workflow activity using the executor registry.

**Flow:**
1. Fetch Activity and ActivityTemplate
2. Fetch Workflow for context
3. Resolve inputs via SchemaProcessor
4. Look up executor by `use_case_code`
5. Execute and map outputs back to context
6. Update Activity status (in_progress → completed/failed)

```go
func (uc *ExecuteActivityUseCase) Execute(
    ctx context.Context,
    req *enginepb.ExecuteActivityRequest,
) (*enginepb.ExecuteActivityResponse, error)
```

### 3. AdvanceWorkflow

Checks current stage completion and moves to next stage if ready.

**Flow:**
1. Fetch Workflow and current Stage
2. Check if all activities are completed/skipped
3. If complete, mark stage as completed
4. Find next StageTemplate by order_index
5. Create next Stage or mark workflow as completed

```go
func (uc *AdvanceWorkflowUseCase) Execute(
    ctx context.Context,
    req *enginepb.AdvanceWorkflowRequest,
) (*enginepb.AdvanceWorkflowResponse, error)
```

### 4. ContinueWorkflow

Handles human input to continue a paused workflow (human_task, approval).

**Flow:**
1. Validate activity is in "pending" state
2. Validate input against ActivityTemplate schema
3. Merge input into workflow context
4. Execute use case if defined
5. Mark activity as completed
6. Auto-advance if stage is complete

```go
func (uc *ContinueWorkflowUseCase) Execute(
    ctx context.Context,
    req *enginepb.ContinueWorkflowRequest,
) (*enginepb.ContinueWorkflowResponse, error)
```

### 5. GetWorkflowStatus

Retrieves the current state of a workflow including pending activities.

**Flow:**
1. Fetch Workflow by ID
2. Find current (non-completed) Stage
3. List activities for current stage
4. Identify pending human_task/approval activities

```go
func (uc *GetWorkflowStatusUseCase) Execute(
    ctx context.Context,
    req *enginepb.GetWorkflowStatusRequest,
) (*enginepb.GetWorkflowStatusResponse, error)
```

## Schema Processing

The `SchemaProcessor` handles input/output schema resolution and validation.

### Schema Field Definition

```go
type SchemaField struct {
    Source   string `json:"source"`   // Field to read from context
    Type     string `json:"type"`     // Target type: string, int, bool
    Default  any    `json:"default"`  // Default value if missing
    Required bool   `json:"required"` // Validation flag
}
```

### Key Methods

```go
// Resolve maps workflow context to activity input/output
func (p *SchemaProcessor) Resolve(
    workflowContext map[string]any,
    mappingJson string,
) (map[string]any, error)

// ValidateInput validates and enriches input against schema
func (p *SchemaProcessor) ValidateInput(
    inputJson string,
    schemaJson string,
) (map[string]any, error)
```

### Type Coercion

Supports automatic conversion for:
- `string`: Any value → string via `fmt.Sprintf`
- `int`: int, float64, string → int
- `bool`: bool, string ("true"), int (non-zero) → bool

## Executor Registry

The engine uses `ExecutorRegistry` (injected via services) to map use case codes to executors:

```go
// Use case code format: {domain}.{resource}.{operation}
"entity.client.create"       -> CreateClientUseCase.Execute
"subscription.plan.list"     -> ListPlansUseCase.Execute
"integration.email.send"     -> SendEmailUseCase.Execute
```

The registry is defined in `application/ports` and implemented in the composition layer.

## Workflow Lifecycle

```
┌──────────────────────────────────────────────────────────────────┐
│                      WORKFLOW LIFECYCLE                          │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  StartWorkflow ──► Stage[0] ──► Activities ──► AdvanceWorkflow   │
│       │                              │               │           │
│       │                              │               ▼           │
│       │                         ExecuteActivity   Stage[n]       │
│       │                              │               │           │
│       │                              │               ▼           │
│       │                         ContinueWorkflow  Complete       │
│       │                         (human tasks)                    │
│       │                                                          │
│       └──────────► GetWorkflowStatus (anytime) ◄─────────────────┘
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

## Activity States

| State | Description |
|-------|-------------|
| `pending` | Activity created, waiting for execution or human input |
| `in_progress` | Activity is currently being executed |
| `completed` | Activity finished successfully |
| `failed` | Activity execution failed |
| `skipped` | Activity was skipped (treated as complete for advancement) |

## Guiding Principles

### Observability
The `orchestration` layer is a critical component for coordinating complex business flows. To ensure effective debugging and monitoring, it must be observable.

**Principle**: The engine should emit structured logs at key lifecycle points (e.g., workflow start, activity execution, workflow end). This is a low-effort, high-reward feature.

### Error Handling
**Principle**: The engine's primary role is orchestration, not error transformation. Errors originating from the executed activities (use cases, integrations) should be propagated upward without modification. The engine is responsible for halting the workflow on failure, not for interpreting the business-level error.

### Lazy Instantiation
**Principle**: Stages and activities are created lazily as the workflow progresses, rather than all at once at workflow start. This allows for dynamic workflow paths based on runtime context.

### Schema-Driven Data Flow
**Principle**: Input/output schemas in templates define how data flows between activities. The SchemaProcessor resolves values from workflow context, enabling loose coupling between activities.

## Related Directories

- `application/ports/` - Defines `WorkflowEngineService` interface and `ExecutorRegistry`
- `composition/routing/config/orchestration/` - Wires engine with HTTP routes
- `composition/core/` - Initializes engine with dependencies
