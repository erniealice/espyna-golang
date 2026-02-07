# Cloud-Specific Build Scripts

## ☁️ Single Cloud Provider Builds (30-50 MB)

These build scripts create **cloud-optimized binaries** tailored for specific cloud platforms. Each script includes only the providers and services for one cloud ecosystem, avoiding multi-cloud bloat while maintaining full integration capabilities.

### Available Scripts

#### `build-fiber-firebase.ps1` - Google Cloud Platform
**Target Size:** 30-40 MB  
**Best For:** Google Cloud Platform deployments
```powershell
.\build-fiber-firebase.ps1
```
**Technology Stack:**
- **HTTP Framework**: Fiber (high performance for cloud workloads)
- **Database**: Firestore (NoSQL, real-time capabilities)
- **Authentication**: Firebase Auth (Google identity platform)
- **Email**: Gmail API (G Suite integration)
- **Storage**: Google Cloud Storage (GCS)
- **Monitoring**: Cloud Logging and Monitoring integration

**Deployment Example:**
```bash
# Google Cloud Run
gcloud run deploy espyna-api \
  --image gcr.io/PROJECT_ID/espyna:latest \
  --platform managed \
  --set-env-vars FIREBASE_PROJECT_ID=your-project
```

#### `build-gin-microsoft.ps1` - Microsoft Azure
**Target Size:** 35-45 MB  
**Best For:** Microsoft Azure and Office 365 integration
```powershell
.\build-gin-microsoft.ps1
```
**Technology Stack:**
- **HTTP Framework**: Gin (enterprise middleware support)
- **Database**: Azure SQL Database (managed SQL Server)
- **Authentication**: Azure Active Directory (enterprise SSO)
- **Email**: Microsoft Graph API (Office 365, Teams integration)
- **Storage**: Azure Blob Storage
- **Integration**: Microsoft Graph, Teams, Office 365

**Deployment Example:**
```bash
# Azure Container Instances
az container create \
  --resource-group espyna-rg \
  --name espyna-api \
  --image espyna.azurecr.io/espyna:latest \
  --environment-variables AZURE_CLIENT_ID=... AZURE_TENANT_ID=...
```

#### `build-fiber-aws.ps1` - Amazon Web Services
**Target Size:** 30-40 MB  
**Best For:** AWS cloud-native deployments
```powershell
.\build-fiber-aws.ps1
```
**Technology Stack:**
- **HTTP Framework**: Fiber (optimized for AWS Lambda/ECS)
- **Database**: RDS PostgreSQL (managed relational database)
- **Authentication**: JWT (stateless, auto-scaling friendly)
- **Email**: Amazon SES (Simple Email Service) via SMTP
- **Storage**: Amazon S3 (infinite scalability)
- **Integration**: AWS CloudWatch, X-Ray, SSM Parameter Store

**Deployment Example:**
```bash
# AWS ECS Fargate
aws ecs create-service \
  --cluster espyna-cluster \
  --service-name espyna-api \
  --task-definition espyna-api:1 \
  --launch-type FARGATE
```

## Cloud Integration Benefits

### Single-Cloud Optimization
- **Reduced Binary Size** - Only includes required cloud SDKs
- **Faster Startup** - Fewer unused dependencies to initialize
- **Lower Memory Usage** - Single cloud SDK in memory
- **Simplified Configuration** - Fewer environment variables needed
- **Better Performance** - Optimized for specific cloud patterns

### vs Multi-Cloud Builds
| Aspect | Single-Cloud | Multi-Cloud |
|--------|-------------|-------------|
| **Binary Size** | 30-45 MB | 70-80 MB |
| **SDKs Included** | 1 cloud platform | All cloud platforms |
| **Configuration** | Simple | Complex |
| **Performance** | Optimized | General purpose |
| **Vendor Lock-in** | Single provider | Provider agnostic |

## Environment Configuration

### Google Cloud Platform (Firebase)
```bash
# Core configuration
FIREBASE_PROJECT_ID=your-gcp-project
GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json
GCS_BUCKET_NAME=your-storage-bucket

# Optional configuration
FIRESTORE_EMULATOR_HOST=localhost:8080  # Development
FIRESTORE_DATABASE_ID=(default)         # Multi-database projects
```

### Microsoft Azure
```bash
# Core configuration  
AZURE_CLIENT_ID=your-application-id
AZURE_TENANT_ID=your-directory-id
AZURE_CLIENT_SECRET=your-client-secret

# Database configuration
SQLSERVER_CONNECTION_STRING="server=...;database=...;user id=...;password=..."

# Storage configuration
AZURE_STORAGE_ACCOUNT=yourstorageaccount
AZURE_STORAGE_ACCESS_KEY=your-access-key
```

### Amazon Web Services
```bash
# Core configuration
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=your-access-key
AWS_SECRET_ACCESS_KEY=your-secret-key

# Database configuration
DATABASE_URL=postgres://user:pass@rds-endpoint:5432/database

# Storage configuration  
S3_BUCKET_NAME=your-s3-bucket
S3_REGION=us-east-1

# Email configuration (SES)
SMTP_HOST=email-smtp.us-east-1.amazonaws.com
SMTP_PORT=587
SMTP_USERNAME=your-ses-smtp-username
SMTP_PASSWORD=your-ses-smtp-password
```

## Deployment Strategies

### Container Optimization
```dockerfile
# Google Cloud optimized
FROM golang:1.24-alpine AS builder
RUN ./build-fiber-firebase.ps1
FROM gcr.io/distroless/base
COPY --from=builder /app/build/espyna-fiber-firebase /espyna
EXPOSE 8080
CMD ["/espyna"]
```

### Serverless Deployment
```bash
# AWS Lambda (requires additional Lambda adapter)
./build-fiber-aws.ps1 -LdFlags "-s -w"  # Minimize for Lambda

# Google Cloud Run (auto-scaling containers)
./build-fiber-firebase.ps1 -LdFlags "-s -w"

# Azure Container Instances
./build-gin-microsoft.ps1 -LdFlags "-s -w"
```

### Kubernetes Deployment
```yaml
# Cloud-specific Kubernetes manifests
apiVersion: apps/v1
kind: Deployment
metadata:
  name: espyna-gcp
spec:
  template:
    spec:
      containers:
      - name: espyna
        image: gcr.io/PROJECT_ID/espyna-firebase:latest
        env:
        - name: FIREBASE_PROJECT_ID
          value: "your-project"
```

## Migration Between Clouds

### From Multi-Cloud to Single-Cloud
1. **Identify current cloud provider** usage in production
2. **Build cloud-specific variant** for your primary cloud
3. **Test thoroughly** with production-like data
4. **Deploy gradually** with canary releases
5. **Monitor performance** improvements (startup time, memory usage)

### Cloud Provider Switching
To switch between cloud providers:
1. **Build new cloud variant** - `./build-gin-microsoft.ps1` → `./build-fiber-firebase.ps1`
2. **Update environment variables** - Azure config → GCP config  
3. **Migrate data** - Azure SQL → Firestore (separate migration process)
4. **Update deployment scripts** - Azure Container Instances → Cloud Run
5. **Test integration** - Ensure all business logic works with new providers

## Performance Characteristics

### Google Cloud Platform (Fiber + Firebase)
- **Cold Start**: ~200ms (Firestore initialization)
- **Memory Usage**: 25-35MB runtime
- **Throughput**: High (Fiber framework optimization)
- **Scaling**: Automatic with Cloud Run
- **Integration**: Native GCP service mesh

### Microsoft Azure (Gin + Azure Services)  
- **Cold Start**: ~300ms (Azure AD token initialization)
- **Memory Usage**: 30-40MB runtime  
- **Throughput**: High (Gin middleware efficiency)
- **Scaling**: Azure App Service auto-scaling
- **Integration**: Enterprise Active Directory

### Amazon Web Services (Fiber + AWS Services)
- **Cold Start**: ~150ms (S3 SDK is lightweight)  
- **Memory Usage**: 25-35MB runtime
- **Throughput**: Very High (Fiber + RDS optimization)
- **Scaling**: ECS Fargate auto-scaling
- **Integration**: AWS CloudWatch, X-Ray tracing

## Best Practices

### Cloud Selection Criteria
- **Existing Infrastructure** - Use cloud where you already have services
- **Team Expertise** - Choose cloud your team knows best
- **Cost Optimization** - Single-cloud avoids data transfer costs
- **Compliance Requirements** - Some clouds better for specific regulations
- **Performance Requirements** - Different clouds excel in different areas

### Optimization Tips
1. **Use cloud-native services** - Managed databases, storage, authentication
2. **Configure health checks** - Proper readiness and liveness probes
3. **Enable auto-scaling** - Configure based on CPU/memory metrics
4. **Monitor performance** - Use cloud-native monitoring tools
5. **Secure service communication** - Use cloud IAM and service meshes

---
*Cloud-specific builds optimize for single cloud platforms, providing better performance and simpler configuration than multi-cloud alternatives.*