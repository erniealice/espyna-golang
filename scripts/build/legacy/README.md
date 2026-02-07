# Legacy Build Scripts

## 📦 Legacy & Utility Build Scripts

These build scripts represent **older build patterns**, **utility functions**, and **specialized deployment scenarios**. While functional, consider using the newer organized build groups for better maintainability.

### Available Scripts

#### `build-vanilla-postgres.ps1` - Traditional Self-Hosted Stack
**Target Size:** 40-50 MB  
**Legacy Pattern**: Traditional monolithic deployment approach
```powershell
.\build-vanilla-postgres.ps1
```
- **HTTP Framework**: Vanilla Go (standard library)
- **Database**: PostgreSQL (ACID transactions)  
- **Authentication**: JWT (stateless tokens)
- **Email**: SMTP (configurable servers)
- **Storage**: Local filesystem

**Migration Path**: Use `optimized/build-minimal-api.ps1` instead for better organization and smaller size.

#### `build-container-k8s.ps1` - Container Optimization
**Target Size:** 45-55 MB  
**Specialized Use**: Docker and Kubernetes optimized builds
```powershell
.\build-container-k8s.ps1
```
- **HTTP Framework**: Fiber (container performance)
- **Features**: Health checks, graceful shutdown, 12-factor app compliance
- **Multi-database support**: PostgreSQL, Firestore
- **Storage**: Multi-cloud (S3, GCS, Azure Blob)

**Migration Path**: Use `cloud-specific/` builds with proper Dockerfile optimization instead.

#### `build-minimal-edge.ps1` - Edge Computing
**Target Size:** 20-30 MB  
**Specialized Use**: IoT, edge devices, minimal resource environments
```powershell
.\build-minimal-edge.ps1
```
- **HTTP Framework**: Vanilla (minimal dependencies)
- **Database**: Mock in-memory (ultra-fast startup)
- **Authentication**: JWT (no external dependencies)
- **Storage**: Local filesystem (self-contained)
- **Email**: Mock console logging (offline capable)

**Migration Path**: Consider `optimized/build-development.ps1` for development or create new edge-optimized group.

### Utility Scripts

#### `quick-build.ps1` - Rapid Development Builds
**Purpose**: Quick builds for development testing
```powershell
.\quick-build.ps1 -Framework gin
.\quick-build.ps1 -Framework fiber -SecondaryTags "postgres,jwt"
```
**Migration Path**: Use `optimized/` group builds with proper configuration instead.

#### `show-builds.ps1` - Build Information
**Purpose**: Display available build variants and their sizes
```powershell
.\show-builds.ps1
```
**Output**: Lists all build outputs with sizes and descriptions
**Migration Path**: Create dedicated build management utilities.

## Why These Are Legacy

### 🔄 **Superseded by Better Alternatives**

| Legacy Script | Better Alternative | Why Better |
|---------------|-------------------|------------|
| `build-vanilla-postgres.ps1` | `optimized/build-minimal-api.ps1` | Smaller size, better organization |
| `build-container-k8s.ps1` | `cloud-specific/build-*-*.ps1` | Single-cloud optimization |
| `build-minimal-edge.ps1` | `optimized/build-development.ps1` | More comprehensive mock data |
| `quick-build.ps1` | `optimized/` + `cloud-specific/` | Purpose-built variants |

### 📁 **Organization Issues**
- **Scattered patterns** - Mixed deployment scenarios in single directory  
- **Inconsistent naming** - No clear grouping or hierarchy
- **Redundant functionality** - Multiple scripts doing similar things
- **Maintenance burden** - Harder to update and maintain

## Migration Recommendations

### For Production Deployments
```powershell
# OLD (Legacy)
.\legacy\build-vanilla-postgres.ps1

# NEW (Optimized)  
.\optimized\build-minimal-api.ps1    # 30-50% smaller
```

### For Cloud Deployments
```powershell
# OLD (Legacy)
.\legacy\build-container-k8s.ps1

# NEW (Cloud-Specific)
.\cloud-specific\build-fiber-firebase.ps1   # Google Cloud
.\cloud-specific\build-gin-microsoft.ps1    # Microsoft Azure  
.\cloud-specific\build-fiber-aws.ps1        # Amazon AWS
```

### For Development
```powershell
# OLD (Legacy)  
.\legacy\build-minimal-edge.ps1

# NEW (Development-Focused)
.\development\build-development.ps1          # Comprehensive mock data
.\development\build-development-debug.ps1    # With debugging features
```

## When to Still Use Legacy Scripts

### ✅ **Valid Use Cases**
- **Temporary compatibility** - While migrating to new build structure
- **Specialized requirements** - Very specific deployment scenarios not covered by new groups
- **Research and comparison** - Understanding different build approaches
- **Emergency fallback** - If new builds have issues (temporary measure)

### ❌ **Avoid for New Projects**
- **New deployments** - Use organized build groups instead
- **Production systems** - Prefer optimized builds for better performance
- **Team onboarding** - New developers should learn the organized structure
- **CI/CD pipelines** - Use predictable, organized build paths

## Deprecation Timeline

### Phase 1: Migration Period (Current)
- **Legacy scripts available** for backward compatibility
- **Documentation guides** users to better alternatives  
- **New development** should use organized groups

### Phase 2: Deprecation Warnings (Future)
- **Console warnings** when running legacy scripts
- **Documentation updates** emphasizing migration paths
- **Team training** on new build organization

### Phase 3: Retirement (Future)
- **Legacy scripts moved** to `archived/` directory
- **Automated migration tools** to convert existing usage
- **Full transition** to organized build groups

## Legacy Script Maintenance

### ⚠️ **Limited Support**
- **Bug fixes only** - No new features added to legacy scripts
- **Security updates** - Critical security issues will be patched
- **Compatibility** - Maintained for existing deployments
- **Documentation** - Updated only for critical changes

### 🔧 **Self-Service**
If you need to continue using legacy scripts:
1. **Understand the risks** - Less optimized, harder to maintain
2. **Plan migration** - Set timeline to move to organized builds  
3. **Document usage** - Why legacy script is needed vs new alternatives
4. **Monitor performance** - Compare with optimized builds regularly

## Organized Build Structure Benefits

The new organized structure provides:

- **Clear categorization** - optimized/, enterprise/, development/, cloud-specific/
- **Consistent documentation** - Each group has comprehensive README
- **Size optimization** - 30-85% smaller binaries than legacy builds
- **Maintenance efficiency** - Easier to update and extend
- **Team productivity** - Faster onboarding and decision making
- **Future extensibility** - Easy to add new specialized groups

---
*Legacy scripts remain available for compatibility but consider migrating to the organized build groups for better performance, smaller binaries, and easier maintenance.*