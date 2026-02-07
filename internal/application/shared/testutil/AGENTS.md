# Espyna Test Utilities - Agent Guidelines

## Purpose
Simple shared helpers to reduce duplication and ensure consistency across all 200+ test files in espyna usecases.

## Core Principle
**Keep it simple.** These are basic helper functions, not a complex framework.

## Available Helpers

### 1. Context Helpers (`context.go`, `context_helpers.go`)
```go
// Standard test context creation
ctx := testutil.CreateTestContext()        // Or testutil.CreateStandardTestContext()
businessType := testutil.GetTestBusinessType()

// With specific user
ctx := testutil.CreateTestContextWithUser("custom-user-id")

// Environment-aware context creation
// Respects TEST_USER_ID and TEST_BUSINESS_TYPE environment variables
```

### 2. Service Helpers (`services.go`)
```go
// Create standard mock services (includes idAdapter for ALL operations)
services := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)

// Access individual services
authService := services.AuthorizationService
transactionService := services.TransactionService
translationService := services.TranslationService
idService := services.IDService  // ALWAYS included
```

### 3. Error Assertion Helpers (`errors.go`)
```go
// Simple error validation
testutil.AssertTranslatedError(t, err, "domain.validation.field_required", translationService, ctx)

// Error with context substitution (e.g., ID replacement)
testutil.AssertTranslatedErrorWithContext(t, err, "domain.errors.not_found", "{\"id\": \"123\"}", translationService, ctx)
```

### 4. Repository Creation
```go
// Keep using the simple, direct approach - no helper needed
mockRepo := entity.NewMockAdminRepository(businessType)
mockRepo := subscription.NewMockPlanRepository(businessType)
```

## Standard Import Template

**ALL test files should use these imports (for consistency):**

```go
import (
    "strings"           // ALWAYS - for error message validation
    "testing"           // ALWAYS
    "time"              // ALWAYS - even if not used

    contextutil "leapfor.xyz/espyna/internal/application/shared/context"
    "leapfor.xyz/espyna/internal/application/shared/testutil"
    mockAuth "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/auth/mock"
    mockDb "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock"
    "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/[domain]"
    idAdapter "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/id"
    "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/translation"

    [entitypb] "leapfor.xyz/esqyma/golang/v1/domain/[domain]/[entity]"
)
```

## Transaction Pattern Standard

**Use executeCore pattern everywhere:**

```go
func (uc *UseCase) Execute(ctx context.Context, req *Request) (*Response, error) {
    // Validation, business logic, data enrichment

    // Transaction decision
    if uc.services.TransactionService.SupportsTransactions() {
        return uc.executeWithTransaction(ctx, req, enrichedData)
    }
    return uc.executeCore(ctx, req, enrichedData)
}

func (uc *UseCase) executeCore(ctx context.Context, req *Request, data *Entity) (*Response, error) {
    // Pure business logic - no transaction concerns
}

func (uc *UseCase) executeWithTransaction(ctx context.Context, req *Request, data *Entity) (*Response, error) {
    // Pure transaction management - delegates to executeCore
}
```

## Test Helper Pattern

```go
func createTest[Operation][Entity]UseCase(businessType string, supportsTransaction bool) *[Operation][Entity]UseCase {
    repositories := [Operation][Entity]Repositories{
        [Entity]: [domain].NewMock[Entity]Repository(businessType),  // Simple, direct creation
    }

    standardServices := testutil.CreateStandardServices(supportsTransaction, true)
    services := [Operation][Entity]Services{
        AuthorizationService: standardServices.AuthorizationService,
        TransactionService:   standardServices.TransactionService,
        TranslationService:   standardServices.TranslationService,
        IDService:           standardServices.IDService,  // Include if service struct has it
    }

    return New[Operation][Entity]UseCase(repositories, services)
}
```

## Standard Test Scenarios

**Every use case should have these tests:**

### For CREATE operations:
1. `TestUseCase_Execute_Success` - Basic happy path
2. `TestUseCase_Execute_WithTransaction` - Transaction support
3. `TestUseCase_Execute_NilRequest` - Nil request validation
4. `TestUseCase_Execute_NilData` - Nil data validation
5. `TestUseCase_Execute_ValidationErrors` - Required field validation
6. `TestUseCase_DataEnrichment` - ID generation, timestamps, flags

### For READ operations:
1. `TestUseCase_Execute_Success` - Basic happy path
2. `TestUseCase_Execute_NotFound` - Non-existent ID
3. `TestUseCase_Execute_EmptyId` - Empty ID validation

### For UPDATE operations:
1. `TestUseCase_Execute_Success` - Basic happy path
2. `TestUseCase_Execute_WithTransaction` - Transaction support
3. `TestUseCase_Execute_NonExistentId` - Non-existent ID
4. `TestUseCase_Execute_ValidationErrors` - Validation scenarios

### For DELETE operations:
1. `TestUseCase_Execute_Success` - Basic happy path
2. `TestUseCase_Execute_WithTransaction` - Transaction support
3. `TestUseCase_Execute_NonExistentId` - Non-existent ID
4. `TestUseCase_Execute_EmptyId` - Empty ID validation

### For LIST operations:
1. `TestUseCase_Execute_Success` - Basic happy path
2. `TestUseCase_Execute_EmptyResult` - Empty result handling
3. `TestUseCase_Execute_NilRequest` - Nil request validation

## Usage Guidelines

### DO:
- Use testutil helpers for common patterns
- Include idAdapter in ALL operations
- Use standard import template
- Follow executeCore pattern
- Validate specific error messages (not just boolean)
- Include all standard test scenarios

### DON'T:
- Create complex abstractions
- Over-engineer test patterns
- Skip standard imports for "optimization"
- Use boolean-only error validation
- Mix transaction patterns

## Migration Notes

When updating existing tests:
1. **Fix imports first** - use standard template
2. **Replace repetitive helpers** - use testutil functions
3. **Add missing test scenarios** - ensure all standard tests exist
4. **Update transaction pattern** - migrate to executeCore where needed

## Code Review Checklist

- [ ] Uses standard imports (all required imports present)
- [ ] Uses testutil helpers where applicable
- [ ] Includes idAdapter import and usage
- [ ] Follows executeCore pattern
- [ ] Uses specific error message validation
- [ ] Has all mandatory test scenarios for operation type

## Remember

These helpers exist to **reduce repetition and ensure consistency**. They're not meant to replace good judgment or domain-specific testing needs. Use them where they help, extend them when needed, but keep them simple.