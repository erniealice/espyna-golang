# Development Build Scripts

## 🛠️ Development-Focused Builds (5-20 MB)

These build scripts are designed for **development workflows**, featuring comprehensive debugging capabilities, mock data, and rapid iteration support.

### Available Scripts

#### `build-development.ps1`
**Target Size:** 5-15 MB  
**Best For:** Ultra-lightweight local development
```powershell
.\build-development.ps1
```
- **Mock providers only** - zero external dependencies required
- **Comprehensive test data** - all 40 business entities with realistic data
- **All business types** - education, fitness_center, office_leasing scenarios
- **Instant startup** - no database setup or external services needed
- **Perfect for onboarding** - new developers can run immediately

#### `build-development-debug.ps1`  
**Target Size:** 15-20 MB  
**Best For:** Debugging and integration testing
```powershell
.\build-development-debug.ps1 -VerboseBuild -Race
```
- **Gin framework** with debug middleware and hot-reload support
- **Race detection** enabled for concurrency debugging
- **Debug symbols** included for stack traces and profiling
- **All provider types** for comprehensive integration testing
- **Verbose logging** with request/response tracing

#### `build-development-clean.ps1`
**Target Size:** 5-15 MB  
**Best For:** Development builds without emoji encoding issues
```powershell
.\build-development-clean.ps1
```
- **Same as build-development.ps1** but with emoji characters removed
- **Windows PowerShell compatible** - fixes encoding issues
- **CI/CD friendly** - works in automated environments

## Development Workflow Integration

### Local Development Setup
```powershell
# 1. Build development server
.\development\build-development.ps1

# 2. Run immediately (no configuration needed)
./build/espyna-development

# 3. Access development endpoints
# Health: http://localhost:8080/health
# API: http://localhost:8080/api/entities/client
```

### Debugging Workflow
```powershell
# 1. Build with debugging enabled
.\development\build-development-debug.ps1 -Race -VerboseBuild

# 2. Run with debug environment
LOG_LEVEL=debug MOCK_MODE=true GIN_MODE=debug ./build/espyna-dev-debug

# 3. Debugging features available:
# - Race condition detection
# - Verbose request logging  
# - Debug symbols for profiling
# - Hot-reload friendly configuration
```

### Business Type Testing
```powershell
# Education business scenario (students, teachers, courses)
MOCK_BUSINESS_TYPE=education ./build/espyna-development

# Fitness center scenario (members, trainers, classes) 
MOCK_BUSINESS_TYPE=fitness_center ./build/espyna-development

# Office leasing scenario (tenants, properties, leases)
MOCK_BUSINESS_TYPE=office_leasing ./build/espyna-development
```

## Mock Data Capabilities

### Available Business Entities (40 total)
- **Entity Domain (17)**: Admin, Client, ClientAttribute, Delegate, etc.
- **Event Domain (2)**: Event, EventClient
- **Framework Domain (3)**: Framework, Objective, Task
- **Payment Domain (3)**: Payment, PaymentMethod, PaymentProfile
- **Product Domain (8)**: Product, Collection, ProductPlan, etc.
- **Subscription Domain (6)**: Balance, Invoice, Plan, Subscription, etc.

### Mock Provider Features
- **In-memory database** - No PostgreSQL/Firestore setup required
- **Test authentication** - Predefined users and tokens
- **Console email logging** - Email operations logged to stdout
- **Temporary file storage** - File uploads handled in temp directories

## Development Environment Variables

### Core Configuration (All Optional)
```bash
SERVER_TYPE=vanilla           # HTTP framework (vanilla/gin/fiber)
SERVER_PORT=8080             # HTTP port (default: 8080)
LOG_LEVEL=debug              # Logging verbosity
MOCK_MODE=true               # Always true for development builds
```

### Business Type Selection
```bash
MOCK_BUSINESS_TYPE=education      # Student/teacher/course data
MOCK_BUSINESS_TYPE=fitness_center # Member/trainer/class data  
MOCK_BUSINESS_TYPE=office_leasing # Tenant/property/lease data
```

### Development Features
```bash
MOCK_DELAY_MS=100            # Add realistic API response delays
MOCK_ERROR_RATE=0.05         # Simulate 5% error rate for testing
GIN_MODE=debug               # Gin framework debug mode (debug build)
```

## CI/CD Integration

### Automated Testing
```bash
# Perfect for CI pipelines - no external dependencies
docker run --rm -p 8080:8080 espyna-dev:latest

# Health check for readiness
curl -f http://localhost:8080/health || exit 1

# API testing with mock data  
curl http://localhost:8080/api/entities/client
```

### Build Matrix Testing
```powershell
# Test all business types in CI
foreach ($businessType in @('education', 'fitness_center', 'office_leasing')) {
    $env:MOCK_BUSINESS_TYPE = $businessType
    ./build/espyna-development
    # Run integration tests...
}
```

## Development vs Production

| Aspect | Development Builds | Production Builds |
|--------|-------------------|-------------------|
| **Size** | 5-20 MB | 25-80 MB |
| **Dependencies** | Mock only | Real providers |
| **Startup** | Instant | Requires configuration |
| **Data** | Comprehensive test data | Real business data |
| **Setup** | Zero configuration | Database, auth, etc. |
| **Debugging** | Full debug support | Optimized performance |
| **Networking** | Offline capable | Cloud service integration |

## Best Practices

### For New Developers
1. **Start with development build** - `./development/build-development.ps1`
2. **Run immediately** - No setup required
3. **Explore all entities** - 40 business entities with mock data
4. **Test all business types** - Switch between education/fitness/office scenarios

### For API Development  
1. **Use debug build** - `./development/build-development-debug.ps1 -Race`
2. **Enable verbose logging** - `LOG_LEVEL=debug`
3. **Add realistic delays** - `MOCK_DELAY_MS=100`
4. **Test error handling** - `MOCK_ERROR_RATE=0.05`

### For Integration Testing
1. **Use comprehensive mock data** - All 40 entities available
2. **Test business rule validation** - Foreign key constraints enforced
3. **Verify all endpoints** - Complete CRUD operations for each entity
4. **Cross-business-type testing** - Ensure consistency across domains

---
*Development builds enable rapid iteration with comprehensive mock data and zero external dependencies.*