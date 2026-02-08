# Espyna Package Server - Developer Guide

## =� Quick Start

The Espyna package provides multiple HTTP server implementations (Vanilla, Gin, Fiber) that can be configured with different database, authentication, and storage providers.

---

## =� Running with Mock Providers (Development)

### Vanilla HTTP Server + All Mock Providers

This is the recommended setup for local development and testing.

#### Step 1: Set Environment Variables

```bash
export CONFIG_DATABASE_PROVIDER=mock_db
export CONFIG_AUTH_PROVIDER=mock_auth
export CONFIG_STORAGE_PROVIDER=mock_storage
export BUSINESS_TYPE=education
```

#### Step 2: Run with Build Tags

```bash
cd packages/espyna/cmd/server

# Run vanilla server with all mock providers (with logging)
go run -tags vanilla,mock_db,mock_auth,mock_storage main_vanilla.go > server.log 2>&1 &

# Check the server logs
tail -f server.log

# Or check startup logs
head -50 server.log
```

**Why use `> server.log 2>&1 &`?**
- `> server.log` - Redirects stdout to server.log
- `2>&1` - Redirects stderr to the same file
- `&` - Runs the process in the background
- Allows you to check logs later without console clutter

#### Expected Output

```
>� Using Default DatabaseTableConfig (Mock Database)
   Sample tables: Client=client, Subscription=subscription, Payment=payment
   Business Type: education
= Calling container.Initialize()...
=� Starting container initialization...
=' Initializing provider manager...
   DatabaseTableConfig pointer: 0xc0000cd888
   DatabaseTableConfig values: Client=client, Manager=manager, Delegate=delegate
2025/11/30 01:54:08  Mock Provider: Initialized with name 'mock'
2025/11/30 01:54:08 [OK] Mock Auth provider initialized
>� Mock Storage provider initialized (using local storage)
 Provider manager initialized
...
=� Espyna API server (vanilla) starting on http://localhost:8080
```

#### Test the API

```bash
# Test health endpoint
curl -X GET http://localhost:8080/health

# Test client list endpoint
curl -X POST http://localhost:8080/api/entity/client/list \
  -H "Content-Type: application/json" \
  -d '{}'

# Test with pretty JSON output
curl -X POST http://localhost:8080/api/entity/client/list \
  -H "Content-Type: application/json" \
  -d '{}' | python -m json.tool
```

---

## =% Running with Firestore Database

### Vanilla Server + Firestore + Mock Auth/Storage

#### Prerequisites

1. Google Cloud project with Firestore enabled
2. Service account credentials JSON file
3. `.env` file configured (see below)

#### Step 1: Create `.env` File

Create a `.env` file in `packages/espyna/cmd/server/`:

```bash
# Database Configuration
CONFIG_DATABASE_PROVIDER=firestore
FIRESTORE_PROJECT_ID=your-project-id
FIRESTORE_CREDENTIALS_PATH=./path/to/service-account-key.json

# Collection Names (optional - defaults shown)
LEAPFOR_DATABASE_FIRESTORE_COLLECTION_CLIENT=client_v1
LEAPFOR_DATABASE_FIRESTORE_COLLECTION_MANAGER=manager_v1
LEAPFOR_DATABASE_FIRESTORE_COLLECTION_SUBSCRIPTION=subscription_v1
LEAPFOR_DATABASE_FIRESTORE_COLLECTION_PAYMENT=payment_v1
# ... add more as needed

# Auth Configuration
CONFIG_AUTH_PROVIDER=mock_auth

# Storage Configuration
CONFIG_STORAGE_PROVIDER=mock_storage
```

#### Step 2: Load Environment Variables

Add environment loading to `main_vanilla.go`:

```go
import "github.com/joho/godotenv"

func main() {
    // Load .env file
    godotenv.Load()

    // ... rest of the code
}
```

#### Step 3: Run with Build Tags

```bash
# Run with Firestore (with logging)
go run -tags vanilla,firestore,mock_auth,mock_storage main_vanilla.go > server.log 2>&1 &

# Check the server logs
tail -f server.log

# Or check if Firestore connected successfully
head -30 server.log
```

---

## <� Available Server Frameworks

### 1. Vanilla HTTP Server

```bash
# Run with logging
go run -tags vanilla,mock_db,mock_auth,mock_storage main_vanilla.go > server.log 2>&1 &

# Check logs
tail -f server.log
```

**Build Tags:**
- `vanilla` - Required for vanilla HTTP server
- `mock_db` - Mock database provider
- `mock_auth` - Mock authentication provider
- `mock_storage` - Mock storage provider

### 2. Gin Framework

```bash
# Run with logging
go run -tags gin,mock_db,mock_auth,mock_storage main_gin.go > server.log 2>&1 &

# Check logs
tail -f server.log
```

**Build Tags:**
- `gin` - Required for Gin framework
- `mock_db`, `mock_auth`, `mock_storage` - Mock providers

### 3. Fiber Framework

```bash
# Run with logging
go run -tags fiber,mock_db,mock_auth,mock_storage main_fiber.go > server.log 2>&1 &

# Check logs
tail -f server.log
```

**Build Tags:**
- `fiber` - Required for Fiber framework
- `mock_db`, `mock_auth`, `mock_storage` - Mock providers

---

## =' Environment Variables Reference

### Database Providers

| Variable | Values | Description |
|----------|--------|-------------|
| `CONFIG_DATABASE_PROVIDER` | `mock_db`, `postgres`, `firestore` | Database provider to use |
| `BUSINESS_TYPE` | `education`, `healthcare`, `ecommerce` | Mock data business domain |

### Mock Database

| Variable | Default | Description |
|----------|---------|-------------|
| `BUSINESS_TYPE` | `education` | Business domain for mock data |

No additional configuration needed - uses default table names.

### PostgreSQL Database

| Variable | Default | Description |
|----------|---------|-------------|
| `POSTGRES_HOST` | `localhost` | Database host |
| `POSTGRES_PORT` | `5432` | Database port |
| `POSTGRES_NAME` | `espyna` | Database name |
| `POSTGRES_USER` | `postgres` | Database user |
| `POSTGRES_PASSWORD` | - | Database password |
| `POSTGRES_URL` | - | Full connection URL (overrides above) |

### Firestore Database

| Variable | Default | Description |
|----------|---------|-------------|
| `FIRESTORE_PROJECT_ID` | - | Google Cloud project ID (required) |
| `FIRESTORE_CREDENTIALS_PATH` | - | Path to service account JSON |
| `FIRESTORE_DATABASE` | `(default)` | Firestore database name |

### Collection/Table Name Overrides

Override default table/collection names using these patterns:

**PostgreSQL:**
```bash
LEAPFOR_DATABASE_POSTGRES_TABLE_CLIENT=custom_clients
LEAPFOR_DATABASE_POSTGRES_TABLE_MANAGER=custom_managers
```

**Firestore:**
```bash
LEAPFOR_DATABASE_FIRESTORE_COLLECTION_CLIENT=client_v1
LEAPFOR_DATABASE_FIRESTORE_COLLECTION_MANAGER=manager_v1
```

### Authentication Providers

| Variable | Values | Description |
|----------|--------|-------------|
| `CONFIG_AUTH_PROVIDER` | `mock_auth`, `firebase_auth` | Auth provider to use |

### Storage Providers

| Variable | Values | Description |
|----------|--------|-------------|
| `CONFIG_STORAGE_PROVIDER` | `mock_storage`, `local`, `gcs`, `s3` | Storage provider to use |

---

## >� Business Types for Mock Data

The mock database generates realistic data based on business type:

### Education Domain

```bash
BUSINESS_TYPE=education
```

**Terminology:**
- Client = Student
- Manager = Staff/Teacher
- Delegate = Parent/Guardian
- Plan = Academic Year
- Product = Subject/Course
- Subscription = Enrollment
- Invoice = Assessment
- Payment = Tuition Payment

### Healthcare Domain

```bash
BUSINESS_TYPE=healthcare
```

**Terminology:**
- Client = Patient
- Manager = Doctor/Nurse
- Subscription = Treatment Plan
- Payment = Medical Bill Payment

### Ecommerce Domain

```bash
BUSINESS_TYPE=ecommerce
```

**Terminology:**
- Client = Customer
- Manager = Vendor
- Subscription = Membership
- Product = Inventory Item

---

## =� Available API Endpoints

All servers expose these endpoint patterns:

### Health Check
```
GET /health
```

### Entity Domain
```
POST /api/entity/client/list
POST /api/entity/client/create
POST /api/entity/client/read
POST /api/entity/client/update
POST /api/entity/client/delete

POST /api/entity/manager/list
POST /api/entity/delegate/list
POST /api/entity/workspace/list
... and more
```

### Subscription Domain
```
POST /api/subscription/subscription/list
POST /api/subscription/plan/list
POST /api/subscription/invoice/list
POST /api/subscription/payment/list
... and more
```

### Product Domain
```
POST /api/product/product/list
POST /api/product/collection/list
POST /api/product/price-product/list
... and more
```

### Payment Domain
```
POST /api/payment/payment/list
POST /api/payment/payment-method/list
... and more
```

---

## =� Troubleshooting

### DatabaseTableConfig is nil (Panic)

**Error:**
```
DatabaseTableConfig pointer: 0x0
panic: runtime error: invalid memory address or nil pointer dereference
```

**Solution:**
The `DatabaseTableConfig` must be set for all providers. This is now handled automatically:
- **Postgres**: Uses `legacyConfig.Database.Postgres.Tables`
- **Firestore**: Uses `legacyConfig.Database.Firebase.Collections`
- **Mock**: Uses `appConfig.DefaultDatabaseTableConfig()`

If you see this error, ensure the container initialization code in `container.go` handles all three cases.

### Build Constraints Exclude All Files

**Error:**
```
build constraints exclude all Go files
```

**Solution:**
Add the required build tags. For mock mode:
```bash
go run -tags vanilla,mock_db,mock_auth,mock_storage main_vanilla.go
```

### Port Already in Use

**Error:**
```
bind: address already in use
```

**Solution:**
Kill the process using the port:
```bash
# Windows
netstat -ano | findstr :8080
taskkill /F /PID <pid>

# Linux/macOS
lsof -ti:8080 | xargs kill -9
```

### Firestore Credentials Not Found

**Error:**
```
credentials: could not find default credentials
```

**Solution:**
Set the credentials path in your `.env` file:
```bash
FIRESTORE_CREDENTIALS_PATH=./service-account-key.json
```

And ensure the file exists in that location.

---

## <� Example Complete Setup

### Development with Mock Everything

```bash
# Set environment variables
export CONFIG_DATABASE_PROVIDER=mock_db
export CONFIG_AUTH_PROVIDER=mock_auth
export CONFIG_STORAGE_PROVIDER=mock_storage
export BUSINESS_TYPE=education

# Run vanilla server with logging
cd packages/espyna/cmd/server
go run -tags vanilla,mock_db,mock_auth,mock_storage main_vanilla.go > server.log 2>&1 &

# Check logs
tail -f server.log
```

### Production with Firestore

```bash
# Create .env file with Firestore config
cat > .env <<EOF
CONFIG_DATABASE_PROVIDER=firestore
FIRESTORE_PROJECT_ID=my-project
FIRESTORE_CREDENTIALS_PATH=./creds.json
CONFIG_AUTH_PROVIDER=firebase_auth
CONFIG_STORAGE_PROVIDER=gcs
EOF

# Run with firestore tags and logging
go run -tags vanilla,firestore,firebase_auth main_vanilla.go > server.log 2>&1 &

# Monitor logs to ensure Firestore connection
tail -f server.log
```

---

## =� Key Architecture Insights

### DatabaseTableConfig Flow

1. **Environment Loading**: Config values read from environment variables
2. **Config Creation**: `CreateConfig()` builds configuration based on `CONFIG_*_PROVIDER` variables
3. **Container Setup**: `NewContainerFromLegacyConfig()` extracts `DatabaseTableConfig`:
   - **Postgres**: From `Tables` field
   - **Firestore**: From `Collections` field
   - **Mock**: From `DefaultDatabaseTableConfig()`
4. **Provider Initialization**: `DatabaseTableConfig` pointer passed to provider manager
5. **Repository Creation**: Each repository uses table/collection names from config

### Build Tag System

Build tags enable conditional compilation:
- Provider implementations only compile when their tag is specified
- Prevents unused providers from being included in binary
- Follows Go standard library pattern (like `database/sql`)

---

**Last Updated:** 2025-11-30
**Espyna Version:** 1.0.0
**Go Version:** 1.25.1+
