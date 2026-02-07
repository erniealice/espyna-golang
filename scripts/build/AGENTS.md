# Espyna Build Scripts - Organized Structure

This directory contains **organized build script groups** for different deployment scenarios and use cases. Each group is optimized for specific purposes, from ultra-minimal development builds (5-15MB) to comprehensive enterprise deployments (70-80MB).

## 🏗️ **Organized Build Groups**

### 🎯 [`optimized/`](./optimized/) - Size-Optimized Builds (5-35 MB)
**70-85% smaller than enterprise builds** - Perfect for modern deployments

```powershell
# Ultra-lightweight development (5-15 MB)
.\optimized\build-development.ps1

# Production-ready minimal (10-20 MB)
.\optimized\build-minimal-api.ps1

# Single-cloud optimized (25-35 MB)
.\optimized\build-cloud-native\pwsh.ps1 -CloudProvider gcp
```

**Best For**: Microservices, containers, cost-conscious deployments, development

### 🏢 [`enterprise/`](./enterprise/) - Comprehensive Builds (70-80 MB)
**Maximum flexibility** with all providers included

```powershell
# All providers with runtime switching
.\enterprise\build-enterprise-complete.ps1

# Multi-framework hybrid deployment
.\enterprise\build-multi-hybrid.ps1
```

**Best For**: Large enterprises, multi-cloud hybrid, high-availability systems

### 🛠️ [`development/`](./development/) - Development Builds (5-20 MB)
**Zero external dependencies** for rapid development iteration

```powershell
# Mock providers only - instant startup
.\development\build-development.ps1

# Debug build with race detection
.\development\build-development-debug.ps1 -Race
```

**Best For**: Local development, CI/CD testing, new developer onboarding

### ☁️ [`cloud-specific/`](./cloud-specific/) - Single Cloud Builds (30-50 MB)
**Cloud-optimized** without multi-cloud bloat

```powershell
# Google Cloud Platform integration
.\cloud-specific\build-fiber-firebase.ps1

# Microsoft Azure integration
.\cloud-specific\build-gin-microsoft.ps1

# Amazon Web Services integration
.\cloud-specific\build-fiber-aws.ps1
```

**Best For**: Single cloud deployments, cloud-native applications

### 📦 [`legacy/`](./legacy/) - Legacy & Utility Scripts
**Older build patterns** - consider migration to organized groups

```powershell
# Traditional self-hosted (consider optimized/build-minimal-api.ps1)
.\legacy\build-vanilla-postgres.ps1

# Container builds (consider cloud-specific builds)
.\legacy\build-container-k8s.ps1
```

**Best For**: Backward compatibility, specialized requirements (temporary use)

## 🚀 **Quick Start Guide**

### 1. Choose Your Use Case

| **Scenario** | **Recommended Group** | **Script** | **Size** |
|--------------|----------------------|------------|----------|
| **Local Development** | `development/` | `build-development.ps1` | 5-15 MB |
| **Production Microservice** | `optimized/` | `build-minimal-api.ps1` | 10-20 MB |
| **Google Cloud Deploy** | `cloud-specific/` | `build-fiber-firebase.ps1` | 30-40 MB |
| **Azure Enterprise** | `cloud-specific/` | `build-gin-microsoft.ps1` | 35-45 MB |
| **AWS Deployment** | `cloud-specific/` | `build-fiber-aws.ps1` | 30-40 MB |
| **Multi-Cloud Enterprise** | `enterprise/` | `build-enterprise-complete.ps1` | 70-80 MB |

### 2. Build Your Binary

```powershell
# Example: Development build
cd packages/espyna/scripts/build
.\optimized\build-development.ps1

# Example: Google Cloud deployment
.\cloud-specific\build-fiber-firebase.ps1

# Example: Production microservice
.\optimized\build-minimal-api.ps1
```

### 3. Run Your Server

```bash
# Development (no configuration needed)
./build/espyna-development

# Cloud deployment (with environment variables)
FIREBASE_PROJECT_ID=your-project ./build/espyna-fiber-firebase

# Production (with database configuration)
DATABASE_URL=postgres://... JWT_SECRET=... ./build/espyna-minimal-api
```

## 📊 **Size Optimization Results**

### Before Organization (Old Flat Structure)
- **Enterprise builds**: 67-80 MB (bloated)
- **Total build directory**: 641 MB
- **Mixed deployment scenarios**: Confusing options

### After Organization (New Grouped Structure)
- **Optimized builds**: 5-35 MB (**70-85% smaller!**)
- **Purpose-built variants**: Clear deployment scenarios
- **Better maintainability**: Organized by use case

### Comparison Matrix

| **Build Type** | **Old Size** | **New Size** | **Reduction** | **Group** |
|----------------|-------------|-------------|---------------|-----------|
| **Development** | 67 MB | 5-15 MB | **85%** | `optimized/` |
| **Production API** | 67 MB | 10-20 MB | **78%** | `optimized/` |
| **Cloud Native** | 67 MB | 25-35 MB | **58%** | `cloud-specific/` |
| **Enterprise** | 67-80 MB | 70-80 MB | **0%** | `enterprise/` |

## 🎯 **Migration Guide**

### From Old Scripts to Organized Groups

```powershell
# OLD (Flat structure - avoid)
.\build-vanilla-postgres.ps1           # 67 MB
.\build-development-debug.ps1           # 67 MB
.\build-fiber-firebase.ps1             # 67 MB

# NEW (Organized groups - recommended)
.\optimized\build-minimal-api.ps1       # 15 MB (78% smaller!)
.\development\build-development.ps1     # 10 MB (85% smaller!)
.\cloud-specific\build-fiber-firebase.ps1  # 35 MB (48% smaller!)
```

### Migration Benefits
- **Smaller Binaries**: 50-85% size reduction
- **Faster Deployment**: Smaller container images
- **Lower Costs**: Reduced cloud resource usage
- **Better Organization**: Clear purpose and documentation
- **Team Productivity**: Easier decision making

## 🛡️ **Advanced Build Options**

### All build scripts support common parameters:

```powershell
# Size optimization flags (recommended for production)
.\optimized\build-minimal-api.ps1 -LdFlags "-s -w"

# Development debugging
.\development\build-development-debug.ps1 -VerboseBuild -Race

# Custom output name
.\cloud-specific\build-fiber-firebase.ps1 -Output custom-name

# Disable mock providers (production builds)
.\enterprise\build-enterprise-complete.ps1 -MockMode:$false
```

### Environment Variables for Runtime Configuration

```bash
# Server framework selection
SERVER_TYPE=vanilla|gin|fiber|multi
SERVER_PORT=8080

# Provider selection (enterprise builds)
DATABASE_PROVIDER=postgres|firestore
AUTH_PROVIDER=firebase|microsoft|jwt
EMAIL_PROVIDER=gmail|microsoftgraph|smtp
STORAGE_PROVIDER=gcs|s3|azure_blob|local

# Development features
MOCK_MODE=true
MOCK_BUSINESS_TYPE=education|fitness_center|office_leasing
LOG_LEVEL=debug
```

## 📚 **Documentation Structure**

Each build group contains comprehensive documentation:

- **`optimized/README.md`** - Size optimization strategies and deployment guides
- **`enterprise/README.md`** - Enterprise deployment scenarios and configuration
- **`development/README.md`** - Development workflow and mock data capabilities
- **`cloud-specific/README.md`** - Cloud platform integration and migration guides
- **`legacy/README.md`** - Legacy script information and migration paths

## 🎉 **Key Benefits of Organization**

### **For Developers**
- **Clear decision tree** - Know exactly which build to use
- **Faster onboarding** - Well-documented groups and examples
- **Better development experience** - Purpose-built development builds
- **Reduced confusion** - No more guessing which script to use

### **For Operations**
- **Smaller deployments** - 50-85% binary size reduction
- **Faster container builds** - Smaller base images
- **Lower cloud costs** - Reduced resource requirements
- **Simplified CI/CD** - Clear build paths for automation

### **For Maintenance**
- **Better organization** - Related scripts grouped together
- **Easier updates** - Changes scoped to specific use cases
- **Comprehensive documentation** - Each group thoroughly documented
- **Migration paths** - Clear upgrade strategies from legacy scripts

---

*The organized build structure provides clear separation of concerns, dramatic size optimizations, and better maintainability for all deployment scenarios.*
