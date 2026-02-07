# Enterprise Build Scripts

## 🏢 Comprehensive Enterprise Builds (70-80 MB)

These build scripts create full-featured enterprise binaries with **maximum flexibility** and **all providers included**. Use when you need runtime provider switching or comprehensive integration capabilities.

### Available Scripts

#### `build-enterprise-complete.ps1`
**Target Size:** 70-80 MB  
**Best For:** Large enterprises, multi-cloud hybrid, high-availability systems
```powershell
.\build-enterprise-complete.ps1
```
- **All HTTP frameworks** - Gin with enterprise middleware
- **Multi-database support** - PostgreSQL + Firestore backup
- **All authentication providers** - Azure AD, Firebase, JWT
- **All email providers** - Microsoft Graph + Google Workspace
- **Multi-cloud storage** - Azure Blob, GCS, S3, Local
- **Runtime provider switching** - no recompilation needed

#### `build-multi-hybrid.ps1`
**Target Size:** 75-85 MB  
**Best For:** Maximum deployment flexibility, A/B testing different stacks
```powershell
.\build-multi-hybrid.ps1
```
- **All HTTP frameworks** - Vanilla, Gin, Fiber (runtime switchable)
- **All database providers** - PostgreSQL, Firestore, SQL Server
- **All authentication methods** - Firebase, Azure AD, JWT
- **All cloud services** - Google, Microsoft, AWS
- **Provider-agnostic** architecture for gradual migration

## Enterprise Deployment Scenarios

### Microsoft Enterprise Stack
```powershell
.\enterprise\build-enterprise-complete.ps1

# Runtime configuration
SERVER_TYPE=gin
DATABASE_PROVIDER=postgres
AUTH_PROVIDER=microsoft
EMAIL_PROVIDER=microsoftgraph
STORAGE_PROVIDER=azure_blob
./build/espyna-enterprise-complete
```

### Google Workspace Stack
```powershell
.\enterprise\build-enterprise-complete.ps1

# Runtime configuration  
SERVER_TYPE=gin
DATABASE_PROVIDER=firestore
AUTH_PROVIDER=firebase
EMAIL_PROVIDER=gmail
STORAGE_PROVIDER=gcs
./build/espyna-enterprise-complete
```

### Multi-Cloud Hybrid
```powershell
.\enterprise\build-multi-hybrid.ps1

# Mix and match providers at runtime
SERVER_TYPE=gin
DATABASE_PROVIDER=postgres
AUTH_PROVIDER=microsoft
EMAIL_PROVIDER=gmail
STORAGE_PROVIDER=s3
./build/espyna-multi-hybrid
```

## When to Use Enterprise Builds

✅ **Use When:**
- **Large enterprise** organizations with complex requirements
- **Multi-cloud** hybrid deployments required
- **Runtime provider switching** needed without recompilation
- **High-availability** production systems with backup providers
- **Comprehensive integration** requirements across multiple cloud platforms
- **A/B testing** different technology stacks
- **Gradual migration** between cloud providers

❌ **Don't Use When:**
- **Single cloud** deployment (use `optimized/build-cloud-native` instead)
- **Microservices** architecture (use `optimized/build-minimal-api` instead)
- **Development** environment (use `optimized/build-development` instead)
- **Cost-sensitive** deployments where binary size matters
- **Container** deployments where startup time is critical

## Size vs Flexibility Trade-off

| Build Type | Size | Providers | Runtime Switching | Best For |
|------------|------|-----------|-------------------|----------|
| **Optimized** | 5-35 MB | 1-3 providers | No | Specific use cases |
| **Enterprise** | 70-80 MB | All providers | Yes | Maximum flexibility |

## Performance Considerations

Enterprise builds include **ALL providers simultaneously**, which means:

- **Larger Memory Footprint** - All SDKs loaded even if unused
- **Slower Startup** - More dependencies to initialize
- **Higher Resource Usage** - More cloud service clients active
- **Complex Configuration** - More environment variables needed

**💡 Recommendation:** Use enterprise builds only when you truly need the flexibility. For most deployments, optimized builds offer better performance and lower costs.

## Migration Strategy

**From Enterprise to Optimized:**
1. Identify which providers you actually use in production
2. Build with `optimized/build-cloud-native` for your specific cloud
3. Test thoroughly in staging environment
4. Deploy optimized build for better performance and lower costs

**Enterprise Build Benefits:**
- **Zero Downtime Provider Switching** - Change providers via environment variables
- **Disaster Recovery** - Failover between cloud providers
- **Vendor Negotiation** - Avoid vendor lock-in with multi-cloud support

---
*Enterprise builds provide maximum flexibility at the cost of binary size and performance.*