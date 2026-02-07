# Optimized Build Scripts

## 🎯 Size-Optimized Builds (5-35 MB)

These build scripts are designed to create **dramatically smaller binaries** (50-85% reduction) compared to enterprise builds, while maintaining full functionality for specific deployment scenarios.

### Available Scripts

#### `build-development.ps1`
**Target Size:** 5-15 MB  
**Best For:** Local development, CI/CD testing, offline work
```powershell
.\build-development.ps1
```
- **Mock providers only** - zero external dependencies
- **Ultra-fast startup** with comprehensive test data
- **All business types** supported (education, fitness, office_leasing)
- **Perfect for onboarding** - runs immediately without setup

#### `build-minimal-api.ps1`
**Target Size:** 10-20 MB  
**Best For:** Production microservices, containers, cost-conscious deployments
```powershell
.\build-minimal-api.ps1
```
- **PostgreSQL + JWT + SMTP** - reliable production stack
- **Self-contained** - no cloud service dependencies
- **Horizontally scalable** with stateless authentication
- **Docker-optimized** for container deployments

#### `build-cloud-native/`
**Target Size:** 25-35 MB  
**Best For:** Single cloud provider deployments
```powershell
# Available in both PowerShell and Bash variants
.\build-cloud-native\pwsh.ps1 -CloudProvider gcp    # Google Cloud
.\build-cloud-native\pwsh.ps1 -CloudProvider aws    # Amazon Web Services  
.\build-cloud-native\pwsh.ps1 -CloudProvider azure  # Microsoft Azure

# Or using Bash
./build-cloud-native/bash.sh gcp
```
- **Single-cloud optimized** - avoids multi-cloud bloat
- **Cloud-managed services** integration
- **Auto-scaling ready** with cloud-native patterns
- **Container-first** architecture

## Size Comparison

| Build Variant | Size | vs Enterprise | Use Case |
|---------------|------|---------------|----------|
| **Development** | 5-15 MB | 85-90% smaller | Local dev, testing |
| **Minimal API** | 10-20 MB | 70-85% smaller | Production microservices |
| **Cloud Native** | 25-35 MB | 50-65% smaller | Cloud deployments |
| Enterprise Complete | 70-80 MB | Baseline | All providers included |

## Quick Start

1. **For Development:**
   ```powershell
   .\optimized\build-development.ps1
   ./build/espyna-development  # Runs immediately, no setup needed
   ```

2. **For Production:**
   ```powershell
   .\optimized\build-minimal-api.ps1
   DATABASE_URL=postgres://... JWT_SECRET=... ./build/espyna-minimal-api
   ```

3. **For Cloud:**
   ```powershell
   .\optimized\build-cloud-native\pwsh.ps1 -CloudProvider gcp
   FIREBASE_PROJECT_ID=... ./build/espyna-cloud-gcp
   ```

## Benefits

- **Faster Startup** - Fewer dependencies to initialize
- **Lower Memory Usage** - Smaller runtime footprint  
- **Reduced Costs** - Smaller cloud resource requirements
- **Better Performance** - Optimized for specific use cases
- **Simplified Deployment** - Single-purpose binaries
- **Container Friendly** - Minimal base image sizes

---
*These optimized builds solve the binary bloat problem while maintaining full Espyna functionality.*