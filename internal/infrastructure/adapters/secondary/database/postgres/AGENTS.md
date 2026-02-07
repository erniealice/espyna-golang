# PostgreSQL Database Adapter - Connection Guide

## Overview
This directory contains the PostgreSQL implementation of the database adapter layer in the Espyna hexagonal architecture. The PostgreSQL adapter provides relational database storage for all 40 business entities across 7 domains.

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
│  │   PostgreSQL Adapter (This Layer)   │   │
│  │  • PostgreSQLProvider                │   │
│  │  • PostgresOperations                │   │
│  │  • 40 Repository Implementations     │   │
│  └─────────────────────────────────────┘   │
└─────────────────────────────────────────────┘
                    ↓
        ┌───────────────────────┐
        │    PostgreSQL Server   │
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
            │ -tags="postgres"                             │
            ▼                                              ▼
    ┌──────────────────┐                       ┌─────────────────────┐
    │factory_postgres.go│                       │  ProviderRegistry   │
    │postgresFactory{} │                       │  • postgresql → ✓   │
    └──────────────────┘                       │  • firestore  → ✗   │
            │                                  │  • mock       → ✗   │
            │ CreateDatabaseProvider()         └─────────────────────┘
            ▼
    ┌─────────────────────────────────────────────────────────┐
    │      PostgreSQLProvider.Initialize(config)              │
    │  • Creates connection string                            │
    │  • sql.Open("postgres", connString)                     │
    │  • Configures connection pool                           │
    │  • Creates PostgresOperations wrapper                   │
    └─────────────────────────────────────────────────────────┘
                                    │
                                    │ activateProvider()
                                    ▼
                    ┌───────────────────────────────┐
                    │    provider.IsHealthy()       │
                    │   (pings PostgreSQL server)   │
                    └───────────────────────────────┘
                                    │
                                    │ ✅ Healthy
                                    ▼
        ┌───────────────────────────────────────────────────────┐
        │     provider.CreateRepositories(businessType)         │
        │  • Creates PostgresOperations wrapper                 │
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
            │  • Stores by provider name ("postgresql") │
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

Follow these steps in order to connect your application to PostgreSQL.

### Step 1: Set Environment Variables

Configure the following environment variables before running your application.

#### Required Variables

```bash
# Provider Configuration
PROVIDER_PRIMARY=postgresql                   # Use PostgreSQL as primary database
PROVIDER_BUSINESS_TYPE=education              # Business domain (education, fitness_center, office_leasing)

# PostgreSQL Configuration
DATABASE_HOST=localhost                       # PostgreSQL server host
DATABASE_PORT=5432                           # PostgreSQL server port
DATABASE_NAME=espyna_db                      # Database name
DATABASE_USER=postgres                        # Database user
DATABASE_PASSWORD=your_password_here          # Database password
```

#### Optional Variables

```bash
# Connection Configuration
DATABASE_SSL_MODE=disable                     # SSL mode (disable, require, verify-ca, verify-full)
DATABASE_MAX_CONNECTIONS=10                   # Maximum number of connections in pool

# Migration Configuration
DATABASE_MIGRATIONS_PATH=./migrations         # Path to migration files

# Fallback provider if PostgreSQL fails
PROVIDER_FALLBACK=mock                        # Fall back to mock database if connection fails
```

#### Complete .env File Example

Create a `.env` file in your project root:

```bash
# Provider Configuration
PROVIDER_PRIMARY=postgresql
PROVIDER_FALLBACK=mock
PROVIDER_BUSINESS_TYPE=education

# PostgreSQL Configuration
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_NAME=school_management_db
DATABASE_USER=school_admin
DATABASE_PASSWORD=SecurePassword123!

# Connection Tuning
DATABASE_SSL_MODE=disable
DATABASE_MAX_CONNECTIONS=20

# Migrations
DATABASE_MIGRATIONS_PATH=./packages/espyna/migrations
```

#### Setting Up PostgreSQL Server

**Using Docker (Recommended for Development):**
```bash
# Start PostgreSQL container
docker run --name espyna-postgres \
  -e POSTGRES_PASSWORD=your_password_here \
  -e POSTGRES_DB=espyna_db \
  -p 5432:5432 \
  -d postgres:16

# Verify container is running
docker ps

# Connect to PostgreSQL (optional - for manual verification)
docker exec -it espyna-postgres psql -U postgres -d espyna_db
```

**Using Local PostgreSQL Installation:**
```bash
# macOS (via Homebrew)
brew install postgresql@16
brew services start postgresql@16

# Ubuntu/Debian
sudo apt install postgresql postgresql-contrib
sudo systemctl start postgresql

# Create database
createdb -U postgres espyna_db

# Set password
psql -U postgres
ALTER USER postgres PASSWORD 'your_password_here';
\q
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

    // 3. Initialize all providers (PostgreSQL, Auth, Storage, Email)
    // This single call:
    // - Creates PostgreSQL provider based on build tags
    // - Establishes database connection
    // - Configures connection pool
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

    log.Println("✅ Application ready with PostgreSQL")

    // Use repositories in your application
    // Example: Call use cases with these repositories
}
```

#### What Happens During `Initialize()`

When you call `providerManager.Initialize()`, the system automatically:

1. **Reads environment variables** (DATABASE_HOST, DATABASE_PORT, DATABASE_NAME, etc.)
2. **Selects factory** based on build tags (`-tags="postgres"` → postgresFactory)
3. **Creates PostgreSQL provider** via factory.CreateDatabaseProvider("postgresql")
4. **Builds connection string** from configuration
5. **Opens database connection** using database/sql
6. **Configures connection pool** (max connections, idle connections)
7. **Tests connection** with health check (ping)
8. **Creates PostgresOperations** wrapper for database operations
9. **Creates 40 repositories** (one for each entity)
10. **Maps table names** based on business type (e.g., "students" for education)
11. **Validates** that all 40 repositories are present
12. **Registers** everything in thread-safe registries

**You don't need to do any of this manually.** The Provider Manager orchestrates everything.

---

### Step 3: Use Database Operations Through Repositories

Once initialized, use repositories to interact with PostgreSQL. The repositories automatically handle:
- Data conversion (protobuf ↔ SQL rows)
- Table name mapping (business-specific)
- Audit field management (active, date_created, date_modified)
- SQL query generation
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

#### Generated SQL Examples

The PostgresOperations abstraction generates SQL queries automatically:

**CREATE:**
```sql
INSERT INTO students (id, name, email, workspace_id, active, date_created, date_created_string, date_modified, date_modified_string)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;
```

**READ:**
```sql
SELECT * FROM students
WHERE id = $1 AND active = true;
```

**UPDATE:**
```sql
UPDATE students
SET name = $1, email = $2, date_modified = $3, date_modified_string = $4
WHERE id = $5 AND active = true
RETURNING *;
```

**DELETE (Soft):**
```sql
UPDATE students
SET active = false, date_modified = $1, date_modified_string = $2
WHERE id = $3 AND active = true;
```

**LIST:**
```sql
SELECT * FROM students
WHERE active = true AND workspace_id = $1
ORDER BY date_created DESC;
```

#### Table Name Mapping

Tables are automatically named based on your business type:

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

PostgreSQL support requires specific build tags to include the correct provider implementations.

#### Understanding Build Tags

Build tags are compile-time flags that control which code is included in your binary. The Espyna architecture uses build tags to:

1. **Include only needed providers** (smaller binaries)
2. **Enable conditional compilation** (different builds for different environments)
3. **Activate correct factory** (postgresFactory for PostgreSQL, gcpFactory for Firestore, etc.)

#### Required Build Tag

To use PostgreSQL, you **must** include the `postgres` build tag:

```bash
-tags="postgres"
```

#### Common Build Tag Combinations

```bash
# Production with PostgreSQL
-tags="postgres"

# Production with PostgreSQL + specific auth (if available)
-tags="postgres,jwt"

# Development/Testing with Mock Database
-tags="mock_db,mock_auth"

# Production with Firestore (not PostgreSQL)
-tags="firestore,firebase"
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
# Vet with PostgreSQL provider included
go vet -tags="postgres" ./...

# Vet with Mock provider (for tests)
go vet -tags="mock_db,mock_auth" ./...
```

##### Build Binary
```bash
# Build production binary with PostgreSQL
go build -tags="postgres" -o app ./

# Build with specific output location
go build -tags="postgres" -o bin/espyna-server ./cmd/server

# Build for different platforms
GOOS=linux GOARCH=amd64 go build -tags="postgres" -o app-linux ./
```

##### Run Application
```bash
# Run directly with PostgreSQL
go run -tags="postgres" ./

# Run specific package
go run -tags="postgres" ./cmd/server
```

##### Test
```bash
# Unit tests with Mock database (fastest, no real connections)
go test -tags="mock_db,mock_auth" -v ./...

# Integration tests with real PostgreSQL
go test -tags="postgres" -v ./internal/infrastructure/adapters/secondary/database/postgres/...

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
go vet -tags="postgres" ./...

# 4. Run tests
go test -tags="mock_db,mock_auth" ./...

# 5. Build binary
go build -tags="postgres" -o app ./

# 6. Run application
./app
```

**Or as a one-liner:**
```bash
go fmt ./... && go mod tidy && go vet -tags="postgres" ./... && go test -tags="mock_db,mock_auth" ./... && go build -tags="postgres" -o app ./
```

#### What Happens Without Build Tags?

If you forget the build tags, you'll get an error:

```bash
# ❌ This will fail
go run ./

# Error output:
# "no database provider factory included in build - use build tags like 'postgres', 'firestore', or 'mock_db'"
```

The system needs the build tag to know which factory to compile into your binary.

---

## Database Schema Setup

PostgreSQL requires table schemas to be created before use. The provider includes migration support.

### Creating Tables

#### Option 1: Using Migration Service (Recommended)

The PostgreSQL provider includes a migration service for managing database schema:

```go
// Get migration service from provider
migrationService := provider.GetMigrationService()

// Run migrations
if err := migrationService.Up(); err != nil {
    log.Fatalf("Failed to run migrations: %v", err)
}
```

#### Option 2: Manual SQL Scripts

Create SQL migration files in `migrations/` directory:

**migrations/001_create_clients_table.up.sql:**
```sql
CREATE TABLE IF NOT EXISTS students (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    workspace_id VARCHAR(255) NOT NULL,
    active BOOLEAN DEFAULT true,
    date_created TIMESTAMP NOT NULL DEFAULT NOW(),
    date_created_string VARCHAR(50),
    date_modified TIMESTAMP NOT NULL DEFAULT NOW(),
    date_modified_string VARCHAR(50),

    -- Indexes for common queries
    INDEX idx_students_workspace (workspace_id),
    INDEX idx_students_active (active),
    INDEX idx_students_email (email)
);
```

**migrations/001_create_clients_table.down.sql:**
```sql
DROP TABLE IF EXISTS students;
```

Run migrations using your preferred tool:
- [golang-migrate](https://github.com/golang-migrate/migrate)
- [goose](https://github.com/pressly/goose)
- Custom migration runner

#### Option 3: Docker Initialization Script

If using Docker, add initialization SQL:

**docker-compose.yml:**
```yaml
version: '3.8'
services:
  postgres:
    image: postgres:16
    environment:
      POSTGRES_DB: espyna_db
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: your_password
    ports:
      - "5432:5432"
    volumes:
      - ./migrations:/docker-entrypoint-initdb.d
      - postgres_data:/var/lib/postgresql/data

volumes:
  postgres_data:
```

---

## Troubleshooting

### Common Errors and Solutions

#### Error: "no database provider factory included in build"

**Cause:** Missing build tag

**Solution:** Add `-tags="postgres"` to your go command
```bash
go run -tags="postgres" ./
```

---

#### Error: "failed to connect to PostgreSQL: connection refused"

**Cause:** PostgreSQL server not running or wrong connection details

**Solution:**
1. Check PostgreSQL is running:
   ```bash
   # Docker
   docker ps | grep postgres

   # macOS
   brew services list | grep postgresql

   # Linux
   systemctl status postgresql
   ```

2. Verify connection details:
   ```bash
   psql -h localhost -p 5432 -U postgres -d espyna_db
   ```

3. Check firewall/network settings

---

#### Error: "password authentication failed for user"

**Cause:** Incorrect database password or user doesn't exist

**Solution:**
1. Reset password:
   ```sql
   ALTER USER postgres PASSWORD 'new_password';
   ```

2. Verify user exists:
   ```sql
   SELECT usename FROM pg_user;
   ```

3. Check `DATABASE_USER` and `DATABASE_PASSWORD` environment variables match

---

#### Error: "database 'espyna_db' does not exist"

**Cause:** Database not created

**Solution:**
```bash
# Create database
createdb -U postgres espyna_db

# Or via SQL
psql -U postgres
CREATE DATABASE espyna_db;
\q
```

---

#### Error: "pq: SSL is not enabled on the server"

**Cause:** SSL mode set to `require` but server doesn't support SSL

**Solution:** Set `DATABASE_SSL_MODE=disable` for local development
```bash
export DATABASE_SSL_MODE=disable
```

For production, enable SSL on PostgreSQL server

---

#### Error: "relation 'students' does not exist"

**Cause:** Database tables not created

**Solution:** Run migrations to create tables (see "Database Schema Setup" section above)

---

#### Error: "too many open connections"

**Cause:** Connection pool exhausted

**Solution:**
1. Increase max connections:
   ```bash
   export DATABASE_MAX_CONNECTIONS=50
   ```

2. Check for connection leaks in code

3. Increase PostgreSQL max_connections setting:
   ```sql
   ALTER SYSTEM SET max_connections = 200;
   SELECT pg_reload_conf();
   ```

---

## Connection Pool Management

PostgreSQL provider automatically manages connection pooling:

```go
// Connection pool is configured during initialization
// Based on DATABASE_MAX_CONNECTIONS environment variable

// Default settings:
// - MaxOpenConns: 10
// - MaxIdleConns: 5 (half of MaxOpenConns)
// - ConnMaxLifetime: Unlimited
// - ConnMaxIdleTime: Unlimited

// To customize, set environment variable:
// DATABASE_MAX_CONNECTIONS=20
```

### Monitoring Connection Pool

```go
// Get database connection
db := provider.GetConnection().(*sql.DB)

// Check pool statistics
stats := db.Stats()
log.Printf("Open connections: %d", stats.OpenConnections)
log.Printf("In use: %d", stats.InUse)
log.Printf("Idle: %d", stats.Idle)
log.Printf("Wait count: %d", stats.WaitCount)
log.Printf("Wait duration: %v", stats.WaitDuration)
```

---

## Key Files Reference

| Purpose | Location |
|---------|----------|
| **Provider Manager** | `internal/composition/provider_manager.go` |
| **PostgreSQL Provider** | `internal/infrastructure/providers/postgresql/provider.go` |
| **Postgres Operations** | `internal/infrastructure/adapters/secondary/database/postgres/operations.go` |
| **Repository Example** | `internal/infrastructure/adapters/secondary/database/postgres/entity/client.go` |
| **Factory (Postgres)** | `internal/infrastructure/providers/factory_postgres.go` |
| **Provider Registry** | `internal/infrastructure/registry/provider_registry.go` |
| **Repository Registry** | `internal/infrastructure/registry/repository_registry.go` |
| **Business Config** | `internal/infrastructure/registry/config.go` |
| **Config System** | `internal/infrastructure/config/` |

---

## Summary

Connecting to PostgreSQL in the Espyna hexagonal architecture is straightforward:

### The Three Steps

1. **Set environment variables** (DATABASE_HOST, DATABASE_NAME, DATABASE_USER, DATABASE_PASSWORD, PROVIDER_PRIMARY)
2. **Initialize Provider Manager** (`providerManager.Initialize()` - one line does everything)
3. **Use repositories** (all 40 repositories automatically created and ready)

### The One Build Tag

- Use `-tags="postgres"` for all go commands (vet, build, run)

### The Key Benefit

**You never directly touch PostgreSQL/SQL code.** The hexagonal architecture means:
- Business logic stays clean (no database imports)
- Easy to switch databases (just change environment variables and build tags)
- Consistent interface across all 40 entities
- Automatic SQL generation and query optimization
- Business-specific table naming
- Built-in connection pooling and health checks

The Provider Manager handles all complexity automatically. Your application code just uses repositories through clean interfaces.

---

## Additional Resources

### Official Documentation
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Go database/sql Package](https://pkg.go.dev/database/sql)
- [lib/pq PostgreSQL Driver](https://github.com/lib/pq)
- [Docker PostgreSQL](https://hub.docker.com/_/postgres)

### Internal Documentation
- **Espyna Architecture**: `packages/espyna/AGENTS.md`
- **Provider System**: `packages/espyna/internal/infrastructure/providers/`
- **Registry System**: `packages/espyna/internal/infrastructure/registry/`
- **Use Cases**: `packages/espyna/internal/application/usecases/`