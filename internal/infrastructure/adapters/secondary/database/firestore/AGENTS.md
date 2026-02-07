# Firestore Database Adapter - Connection Guide

## Overview
This directory contains the Firestore implementation of the database adapter layer in the Espyna hexagonal architecture. The Firestore adapter provides NoSQL document-based storage for all 40 business entities across 7 domains.

## Architecture Context

### Hexagonal Architecture Position
```
┌─────────────────────────────────────────────┐
│       Application Layer (Use Cases)        │
│         Domain Logic & Business Rules       │
└─────────────────────────────────────────────┘
                    ↓ Ports (Interfaces)
┌─────────────────────────────────────────────┐
│     Secondary Adapters (Infrastructure)     │
│  ┌─────────────────────────────────────┐   │
│  │   Firestore Adapter (This Layer)    │   │
│  │  • FirestoreProvider                 │   │
│  │  • FirestoreOperations               │   │
│  │  • 40 Repository Implementations     │   │
│  └─────────────────────────────────────┘   │
└─────────────────────────────────────────────┘
                    ↓
        ┌───────────────────────┐
        │ Google Cloud Firestore │
        └───────────────────────┘
```

## Complete Initialization Flow

### Visual Overview

```
┌───────────────────────────────────────────────────────────────────────────┐
│                        USER APPLICATION CODE                              │
│                                                                           │
│  main.go:                                                                 │
│    cfg := config.NewConfig()                                              │
│    pm := composition.NewProviderManager(cfg)  ← YOU START HERE           │
│    pm.Initialize()                            ← THIS DOES EVERYTHING     │
└───────────────────────────────────────────────────────────────────────────┘
                                    │
                  ┌─────────────────┴─────────────────┐
                  │  ProviderManager.Initialize()     │
                  └─────────────────┬─────────────────┘
                                    │
            ┌───────────────────────┼───────────────────────┐
            │                       │                       │
            ▼                       ▼                       ▼
    ┌──────────────┐      ┌────────────────┐     ┌────────────────┐
    │ GetFactory() │      │ CreateProvider │     │ RegisterProvider│
    │ (build tags) │──────▶│   (factory)    │────▶│   (registry)   │
    └──────────────┘      └────────────────┘     └────────────────┘
            │                                              │
            │ -tags="firestore"                            │
            ▼                                              ▼
    ┌──────────────────┐                       ┌─────────────────────┐
    │  factory_gcp.go  │                       │  ProviderRegistry   │
    │  gcpFactory{}    │                       │  • firestore → ✓    │
    └──────────────────┘                       │  • postgres  → ✗    │
            │                                  │  • mock      → ✗    │
            │ CreateDatabaseProvider()         └─────────────────────┘
            ▼
    ┌─────────────────────────────────────────────────────────┐
    │         FirestoreProvider.Initialize(config)            │
    │  • firestore.NewClient(ctx, projectID)                  │
    │  • Stores client in provider.client                     │
    │  • Creates FirestoreOperations(client)                  │
    └─────────────────────────────────────────────────────────┘
                                    │
                                    │ activateProvider()
                                    ▼
                    ┌───────────────────────────────┐
                    │    provider.IsHealthy()       │
                    │  (tests Firestore connection) │
                    └───────────────────────────────┘
                                    │
                                    │ ✅ Healthy
                                    ▼
        ┌───────────────────────────────────────────────────────┐
        │     provider.CreateRepositories(businessType)         │
        │  • Creates FirestoreOperations wrapper                │
        │  • Creates 40 repository instances:                   │
        │    - ClientRepository(ops, "students")                │
        │    - ProductRepository(ops, "subjects")               │
        │    - ... (38 more)                                    │
        │  • Validates all 40 present                           │
        └───────────────────────────────────────────────────────┘
                                    │
                                    │ Returns RepositoryCollection
                                    ▼
            ┌───────────────────────────────────────────┐
            │      RepositoryRegistry.Register()        │
            │  • Validates RepositoryCollection         │
            │  • Stores by provider name ("firestore")  │
            │  • Thread-safe registration               │
            └───────────────────────────────────────────┘
                                    │
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────┐
│                    🎉 READY TO USE                              │
│                                                                 │
│  repos := pm.GetActiveProvider().(ports.RepositoryProvider)     │
│           .CreateRepositories(businessType, connection)         │
│                                                                 │
│  repoCollection := repos.(*registry.RepositoryCollection)       │
│  clientRepo := repoCollection.ClientRepository                 │
│  // ... use all 40 repositories                                │
└─────────────────────────────────────────────────────────────────┘
```

---

## Step-by-Step Setup Guide

Follow these steps in order to connect your application to Firestore.

### Step 1: Set Environment Variables

Configure the following environment variables before running your application.

#### Required Variables

```bash
# Provider Configuration
PROVIDER_PRIMARY=firestore                    # Use Firestore as primary database
PROVIDER_BUSINESS_TYPE=education              # Business domain (education, fitness_center, office_leasing)

# Firebase/Firestore Configuration
FIREBASE_PROJECT_ID=your-project-id           # Your Google Cloud project ID
```

#### Authentication Method 1: Service Account File (Recommended)

```bash
# Point to your service account JSON file
GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account-key.json
```

**How to get a service account key:**
1. Go to Google Cloud Console
2. Navigate to IAM & Admin → Service Accounts
3. Create or select a service account
4. Click "Keys" → "Add Key" → "Create new key" → JSON
5. Download the JSON file and set the path in environment variable

#### Authentication Method 2: Service Account JSON in Environment Variables

If you cannot use a file (e.g., in cloud environments), set individual fields:

```bash
FIREBASE_USE_SERVICE_ACCOUNT=true
FIREBASE_TYPE=service_account
FIREBASE_PROJECT_ID=your-project-id
FIREBASE_PRIVATE_KEY_ID=your-private-key-id
FIREBASE_PRIVATE_KEY="-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----\n"
FIREBASE_CLIENT_EMAIL=your-service-account@project.iam.gserviceaccount.com
FIREBASE_CLIENT_ID=your-client-id
FIREBASE_AUTH_URI=https://accounts.google.com/o/oauth2/auth
FIREBASE_TOKEN_URI=https://oauth2.googleapis.com/token
FIREBASE_AUTH_PROVIDER_CERT_URL=https://www.googleapis.com/oauth2/v1/certs
FIREBASE_CLIENT_CERT_URL=https://www.googleapis.com/robot/v1/metadata/x509/your-service-account@project.iam.gserviceaccount.com
```

#### Optional Variables

```bash
# For multi-database Firestore projects
FIRESTORE_DATABASE=(default)                  # Use a specific database instead of (default)

# Fallback provider if Firestore fails
PROVIDER_FALLBACK=mock                        # Fall back to mock database if connection fails
```

#### Complete .env File Example

Create a `.env` file in your project root:

```bash
# Provider Configuration
PROVIDER_PRIMARY=firestore
PROVIDER_FALLBACK=mock
PROVIDER_BUSINESS_TYPE=education

# Firebase Configuration
FIREBASE_PROJECT_ID=my-school-management-system
GOOGLE_APPLICATION_CREDENTIALS=/home/user/credentials/service-account.json

# Optional
FIRESTORE_DATABASE=(default)
```

---

### Step 2: Initialize Provider Manager

This is the **only initialization code** you need. The Provider Manager handles everything automatically.

#### Complete Working Example

```go
package main

import (
    "log"

    "leapfor.xyz/espyna/internal/composition"
    "leapfor.xyz/espyna/internal/infrastructure/config"
    "leapfor.xyz/espyna/internal/infrastructure/registry"
    "leapfor.xyz/espyna/internal/application/ports"
)

func main() {
    // 1. Load configuration from environment variables
    cfg := config.NewConfig()

    // 2. Create Provider Manager
    providerManager := composition.NewProviderManager(cfg)

    // 3. Initialize all providers (Firestore, Auth, Storage, Email)
    // This single call:
    // - Creates Firestore provider based on build tags
    // - Initializes Firestore client connection
    // - Creates all 40 repositories
    // - Validates repository collection
    // - Registers everything in registries
    // - Performs health checks
    if err := providerManager.Initialize(); err != nil {
        log.Fatalf("❌ Failed to initialize providers: %v", err)
    }
    defer providerManager.Close()

    log.Println("✅ Provider system initialized successfully")

    // 4. Get the active database provider
    activeProvider := providerManager.GetActiveProvider()
    log.Printf("✅ Active database provider: %s", activeProvider.Name())

    // 5. Get repositories for your business type
    businessType := cfg.GetProviderConfig().BusinessType
    connection := activeProvider.GetConnection()

    repos, err := activeProvider.(ports.RepositoryProvider).CreateRepositories(
        businessType,
        connection,
    )
    if err != nil {
        log.Fatalf("❌ Failed to create repositories: %v", err)
    }

    // 6. Access all 40 repositories through the collection
    repoCollection := repos.(*registry.RepositoryCollection)

    // Now you have access to all repositories:
    clientRepo := repoCollection.ClientRepository
    productRepo := repoCollection.ProductRepository
    subscriptionRepo := repoCollection.SubscriptionRepository
    // ... all 40 repositories available

    log.Println("✅ Application ready with Firestore")

    // Use repositories in your application
    // Example: Call use cases with these repositories
}
```

#### What Happens During `Initialize()`

When you call `providerManager.Initialize()`, the system automatically:

1. **Reads environment variables** (FIREBASE_PROJECT_ID, GOOGLE_APPLICATION_CREDENTIALS, etc.)
2. **Selects factory** based on build tags (`-tags="firestore"` → gcpFactory)
3. **Creates Firestore provider** via factory.CreateDatabaseProvider("firestore")
4. **Initializes Firestore client** using Google Cloud credentials
5. **Tests connection** with health check
6. **Creates FirestoreOperations** wrapper for database operations
7. **Creates 40 repositories** (one for each entity)
8. **Maps collection names** based on business type (e.g., "students" for education)
9. **Validates** that all 40 repositories are present
10. **Registers** everything in thread-safe registries

**You don't need to do any of this manually.** The Provider Manager orchestrates everything.

---

### Step 3: Use Database Operations Through Repositories

Once initialized, use repositories to interact with Firestore. The repositories automatically handle:
- Data conversion (protobuf ↔ Firestore documents)
- Collection name mapping (business-specific)
- Audit field management (active, date_created, date_modified)
- Error wrapping

#### Example: Client Repository Operations

```go
import (
    "context"

    clientpb "leapfor.xyz/esqyma/golang/v1/domain/entity/client"
)

func main() {
    // ... initialization code from Step 2 ...

    ctx := context.Background()
    clientRepo := repoCollection.ClientRepository

    // CREATE - Add a new client (student in education context)
    createReq := &clientpb.CreateClientRequest{
        Data: &clientpb.Client{
            Name:        "John Doe",
            Email:       "john.doe@school.edu",
            WorkspaceId: "workspace-123",
            // ID, active, date_created, date_modified are added automatically
        },
    }

    createResp, err := clientRepo.CreateClient(ctx, createReq)
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }

    clientID := createResp.Data[0].Id
    log.Printf("✅ Created client: %s", clientID)

    // READ - Retrieve a client by ID
    readReq := &clientpb.ReadClientRequest{
        Data: &clientpb.Client{
            Id: clientID,
        },
    }

    readResp, err := clientRepo.ReadClient(ctx, readReq)
    if err != nil {
        log.Fatalf("Failed to read client: %v", err)
    }

    client := readResp.Data[0]
    log.Printf("✅ Client name: %s, email: %s", client.Name, client.Email)

    // UPDATE - Modify a client
    updateReq := &clientpb.UpdateClientRequest{
        Data: &clientpb.Client{
            Id:    clientID,
            Name:  "John Smith",
            Email: "john.smith@school.edu",
            // date_modified is updated automatically
        },
    }

    updateResp, err := clientRepo.UpdateClient(ctx, updateReq)
    if err != nil {
        log.Fatalf("Failed to update client: %v", err)
    }

    log.Printf("✅ Updated client: %s", updateResp.Data[0].Name)

    // LIST - Get all active clients
    listReq := &clientpb.ListClientsRequest{}

    listResp, err := clientRepo.ListClients(ctx, listReq)
    if err != nil {
        log.Fatalf("Failed to list clients: %v", err)
    }

    log.Printf("✅ Found %d clients", len(listResp.Data))
    for _, c := range listResp.Data {
        log.Printf("  - %s (%s)", c.Name, c.Email)
    }

    // DELETE - Soft delete (sets active=false)
    deleteReq := &clientpb.DeleteClientRequest{
        Data: &clientpb.Client{
            Id: clientID,
        },
    }

    deleteResp, err := clientRepo.DeleteClient(ctx, deleteReq)
    if err != nil {
        log.Fatalf("Failed to delete client: %v", err)
    }

    log.Printf("✅ Deleted client (soft): %v", deleteResp.Success)
}
```

#### Automatic Data Enrichment

All database operations automatically add/update these fields:

**On CREATE:**
- `id` - Auto-generated UUID if not provided
- `active` - Set to `true`
- `date_created` - Current UTC timestamp
- `date_created_string` - ISO 8601 formatted timestamp
- `date_modified` - Current UTC timestamp
- `date_modified_string` - ISO 8601 formatted timestamp

**On UPDATE:**
- `date_modified` - Updated to current UTC timestamp
- `date_modified_string` - Updated ISO 8601 timestamp
- `date_created` fields are preserved

**On DELETE (soft):**
- `active` - Set to `false`
- `date_modified` - Updated to current UTC timestamp
- `date_modified_string` - Updated ISO 8601 timestamp

#### Collection Name Mapping

Collections are automatically named based on your business type:

**Education Business Type:**
- `client` → `students`
- `manager` → `teachers`
- `delegate` → `parents`
- `plan` → `academic_years`
- `product` → `subjects`
- `framework` → `grading_systems`
- `event` → `schedules`
- `subscription` → `enrollments`
- `invoice` → `assessments`
- `balance` → `statements_of_account`

**Fitness Center Business Type:**
- `client` → `members`
- `manager` → `trainers`
- `plan` → `membership_periods`
- `product` → `classes`
- etc.

**Office Leasing Business Type:**
- `client` → `tenants`
- `manager` → `property_managers`
- `plan` → `lease_periods`
- `product` → `office_spaces`
- etc.

This mapping is handled automatically by the `BusinessSpecificConfig` in the registry system.

---

### Step 4: Build Tags for Compilation

Firestore support requires specific build tags to include the correct provider implementations.

#### Understanding Build Tags

Build tags are compile-time flags that control which code is included in your binary. The Espyna architecture uses build tags to:

1. **Include only needed providers** (smaller binaries)
2. **Enable conditional compilation** (different builds for different environments)
3. **Activate correct factory** (gcpFactory for Firestore, postgresFactory for PostgreSQL, etc.)

#### Required Build Tag

To use Firestore, you **must** include the `firestore` build tag:

```bash
-tags="firestore,firebase"
```

- `firestore` - Includes Firestore database provider
- `firebase` - Includes Firebase Auth provider (optional, use `noop` if you don't need auth)

#### Common Build Tag Combinations

```bash
# Production with Firestore + Firebase Auth
-tags="firestore,firebase"

# Production with Firestore + No Auth
-tags="firestore,noop"

# Development/Testing with Mock Database
-tags="mock_db,mock_auth"

# Production with PostgreSQL (not Firestore)
-tags="postgres,firebase"
```

#### Commands with Build Tags

##### Format Code
```bash
go fmt ./...
```
*Note: No build tags needed for formatting*

##### Tidy Dependencies
```bash
go mod tidy
```
*Note: No build tags needed for tidying*

##### Vet (Check for Issues)
```bash
# Vet with Firestore provider included
go vet -tags="firestore,firebase" ./...

# Vet with Mock provider (for tests)
go vet -tags="mock_db,mock_auth" ./...
```

##### Build Binary
```bash
# Build production binary with Firestore
go build -tags="firestore,firebase" -o app ./

# Build with specific output location
go build -tags="firestore,firebase" -o bin/espyna-server ./cmd/server

# Build for different platforms
GOOS=linux GOARCH=amd64 go build -tags="firestore,firebase" -o app-linux ./
```

##### Run Application
```bash
# Run directly with Firestore
go run -tags="firestore,firebase" ./

# Run specific package
go run -tags="firestore,firebase" ./cmd/server
```

##### Test
```bash
# Unit tests with Mock database (fastest, no real connections)
go test -tags="mock_db,mock_auth" -v ./...

# Integration tests with Firestore emulator
export FIRESTORE_EMULATOR_HOST=localhost:8080
go test -tags="firestore,firebase" -v ./internal/infrastructure/adapters/secondary/database/firestore/...

# Test specific package
go test -tags="mock_db,mock_auth" -v ./internal/application/usecases/entity/client

# Test with coverage
go test -tags="mock_db,mock_auth" -cover ./...
```

##### Complete Development Workflow

After making changes, run these commands in sequence:

```bash
# 1. Format code
go fmt ./...

# 2. Tidy dependencies
go mod tidy

# 3. Vet for issues (with your target build tags)
go vet -tags="firestore,firebase" ./...

# 4. Run tests
go test -tags="mock_db,mock_auth" ./...

# 5. Build binary
go build -tags="firestore,firebase" -o app ./

# 6. Run application
./app
```

**Or as a one-liner:**
```bash
go fmt ./... && go mod tidy && go vet -tags="firestore,firebase" ./... && go test -tags="mock_db,mock_auth" ./... && go build -tags="firestore,firebase" -o app ./
```

#### What Happens Without Build Tags?

If you forget the build tags, you'll get an error:

```bash
# ❌ This will fail
go run ./

# Error output:
# "no database provider factory included in build - use build tags like 'firestore', 'postgres', or 'mock_db'"
```

The system needs the build tag to know which factory to compile into your binary.

---

## Troubleshooting

### Common Errors and Solutions

#### Error: "no database provider factory included in build"

**Cause:** Missing build tag

**Solution:** Add `-tags="firestore,firebase"` to your go command
```bash
go run -tags="firestore,firebase" ./
```

---

#### Error: "could not find default credentials"

**Cause:** `GOOGLE_APPLICATION_CREDENTIALS` not set or file doesn't exist

**Solution:**
1. Check environment variable is set:
   ```bash
   echo $GOOGLE_APPLICATION_CREDENTIALS
   ```

2. Verify file exists:
   ```bash
   ls -la /path/to/service-account.json
   ```

3. Set environment variable:
   ```bash
   export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json
   ```

---

#### Error: "project_id is required in configuration"

**Cause:** `FIREBASE_PROJECT_ID` not set

**Solution:** Set the environment variable:
```bash
export FIREBASE_PROJECT_ID=your-project-id
```

---

#### Error: "provider 'firestore' not found"

**Cause:** Build compiled without Firestore support (wrong or missing build tags)

**Solution:** Rebuild with correct build tags:
```bash
go build -tags="firestore,firebase" ./
```

---

#### Error: "PERMISSION_DENIED: Missing or insufficient permissions"

**Cause:** Service account doesn't have proper IAM roles for Firestore

**Solution:** Add required roles in Google Cloud Console:
1. Go to IAM & Admin → IAM
2. Find your service account
3. Add roles:
   - `Cloud Datastore User` (for read/write)
   - `Cloud Datastore Index Admin` (for index management)
   - Or use `Cloud Datastore Owner` for full access

---

#### Error: "The query requires an index"

**Cause:** Complex Firestore query needs a composite index

**Solution:**
1. Check error message for index creation link
2. Click the link to auto-create index, OR
3. Create `firestore.indexes.json`:
   ```json
   {
     "indexes": [
       {
         "collectionGroup": "students",
         "queryScope": "COLLECTION",
         "fields": [
           { "fieldPath": "workspace_id", "order": "ASCENDING" },
           { "fieldPath": "active", "order": "ASCENDING" },
           { "fieldPath": "date_created", "order": "DESCENDING" }
         ]
       }
     ]
   }
   ```
4. Deploy: `firebase deploy --only firestore:indexes`

---

#### Error: "context deadline exceeded"

**Cause:** Network timeout or slow Firestore query

**Solution:**
1. Check internet connectivity to Google Cloud
2. Increase context timeout in your code:
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
   defer cancel()
   ```
3. Optimize query (add indexes, reduce result size)

---

#### Error: "document not found" immediately after creation

**Cause:** Firestore has eventual consistency; trying to read immediately after write

**Solution:** Use transactions for read-after-write consistency:
```go
// Use TransactionManager for atomic operations
tm := provider.GetTransactionManager()
err := tm.ExecuteTransaction(ctx, func(txCtx context.Context) error {
    // Create document
    _, err := repo.CreateClient(txCtx, createReq)
    if err != nil {
        return err
    }

    // Read document (guaranteed to see the write)
    _, err = repo.ReadClient(txCtx, readReq)
    return err
})
```

---

## Local Development with Firestore Emulator

For local development without connecting to production Firestore:

### 1. Install Firebase Tools
```bash
npm install -g firebase-tools
```

### 2. Initialize Firebase in Your Project
```bash
cd packages/espyna
firebase init firestore
```

### 3. Start Firestore Emulator
```bash
firebase emulators:start --only firestore
```

The emulator runs on `localhost:8080` by default.

### 4. Configure Application to Use Emulator
```bash
# Set emulator host
export FIRESTORE_EMULATOR_HOST=localhost:8080

# Run your application
go run -tags="firestore,firebase" ./
```

### 5. Access Emulator UI

Open browser to `http://localhost:4000` to view Firestore data in the emulator.

**Benefits:**
- No Google Cloud project needed
- Fast development cycle
- Free (no charges)
- Automatic data reset on restart

---

## Key Files Reference

| Purpose | Location |
|---------|----------|
| **Provider Manager** | `internal/composition/provider_manager.go` |
| **Firestore Provider** | `internal/infrastructure/providers/firestore/provider.go` |
| **Firestore Operations** | `internal/infrastructure/adapters/secondary/database/firestore/operations.go` |
| **Repository Example** | `internal/infrastructure/adapters/secondary/database/firestore/entity/client.go` |
| **Factory (GCP)** | `internal/infrastructure/providers/factory_gcp.go` |
| **Provider Registry** | `internal/infrastructure/registry/provider_registry.go` |
| **Repository Registry** | `internal/infrastructure/registry/repository_registry.go` |
| **Business Config** | `internal/infrastructure/registry/config.go` |
| **Config System** | `internal/infrastructure/config/` |

---

## Summary

Connecting to Firestore in the Espyna hexagonal architecture is straightforward:

### The Three Steps

1. **Set environment variables** (FIREBASE_PROJECT_ID, GOOGLE_APPLICATION_CREDENTIALS, PROVIDER_PRIMARY)
2. **Initialize Provider Manager** (`providerManager.Initialize()` - one line does everything)
3. **Use repositories** (all 40 repositories automatically created and ready)

### The One Build Tag

- Use `-tags="firestore,firebase"` for all go commands (vet, build, run)

### The Key Benefit

**You never directly touch Firestore code.** The hexagonal architecture means:
- Business logic stays clean (no database imports)
- Easy to switch databases (just change environment variables and build tags)
- Consistent interface across all 40 entities
- Automatic data enrichment and validation
- Business-specific collection naming

The Provider Manager handles all complexity automatically. Your application code just uses repositories through clean interfaces.

---

## Additional Resources

### Official Documentation
- [Firebase Go SDK](https://firebase.google.com/docs/admin/setup)
- [Firestore Go Client](https://pkg.go.dev/cloud.google.com/go/firestore)
- [Firestore Data Model](https://firebase.google.com/docs/firestore/data-model)
- [Firestore Security Rules](https://firebase.google.com/docs/firestore/security/get-started)

### Internal Documentation
- **Espyna Architecture**: `packages/espyna/AGENTS.md`
- **Provider System**: `packages/espyna/internal/infrastructure/providers/`
- **Registry System**: `packages/espyna/internal/infrastructure/registry/`
- **Use Cases**: `packages/espyna/internal/application/usecases/`