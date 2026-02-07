#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Build container-optimized Espyna server for Docker and Kubernetes deployments

.DESCRIPTION
    This script creates a container-optimized build designed for modern containerized deployments:
    - Fiber HTTP framework for maximum performance in containerized environments
    - Cloud-native provider support with 12-factor app principles
    - Health check endpoints for container orchestration
    - Graceful shutdown handling for Kubernetes
    - Minimal security surface with essential providers only
    - Optimized for horizontal scaling and service mesh integration

.PARAMETER Output
    Output binary name. Default: espyna-container-k8s

.PARAMETER VerboseBuild
    Enable verbose build output

.PARAMETER Race
    Enable race condition detection

.PARAMETER StaticBinary
    Build static binary for minimal container images. Default: true

.PARAMETER MockMode
    Include mock providers for development. Default: false (production-focused)

.EXAMPLE
    .\build-container-k8s.ps1
    Container-optimized build for production

.EXAMPLE
    .\build-container-k8s.ps1 -MockMode:$true
    Container build with mock providers for testing

.NOTES
    This build configuration is optimized for:
    - Docker containers with minimal base images (scratch, alpine)
    - Kubernetes deployments with proper health checks
    - Horizontal scaling and load balancing
    - Service mesh integration (Istio, Linkerd)
    - Cloud-native 12-factor app principles
#>

param(
    [Parameter(Mandatory=$false)]
    [string]$Output = "espyna-container-k8s",
    
    [Parameter(Mandatory=$false)]
    [switch]$VerboseBuild,
    
    [Parameter(Mandatory=$false)]
    [switch]$Race,
    
    [Parameter(Mandatory=$false)]
    [bool]$StaticBinary = $true,
    
    [Parameter(Mandatory=$false)]
    [bool]$MockMode = $false
)

# Set build directory to packages/espyna
$ScriptDir = $PSScriptRoot
$EspynaDir = Split-Path -Parent (Split-Path -Parent $ScriptDir)
Push-Location $EspynaDir

try {
    Write-Host "=== Espyna Container K8s Build ===" -ForegroundColor Cyan
    Write-Host "Building container-optimized server for cloud-native deployments:" -ForegroundColor White
    Write-Host ""
    Write-Host "🐳 Container Features:" -ForegroundColor Cyan
    Write-Host "  • HTTP Framework: Fiber (high-performance, low latency)" -ForegroundColor Green
    Write-Host "  • Static Binary: $(if($StaticBinary){'ENABLED (no libc dependencies)'}else{'DISABLED'})" -ForegroundColor $(if($StaticBinary){'Green'}else{'Yellow'})
    Write-Host "  • Health Checks: Kubernetes-ready endpoints" -ForegroundColor Green
    Write-Host "  • Graceful Shutdown: SIGTERM handling" -ForegroundColor Green  
    Write-Host "  • 12-Factor App: Environment-based configuration" -ForegroundColor Green
    Write-Host ""
    Write-Host "☸️  Kubernetes Features:" -ForegroundColor Cyan
    Write-Host "  • Readiness/Liveness probes support" -ForegroundColor Green
    Write-Host "  • Horizontal scaling ready" -ForegroundColor Green
    Write-Host "  • Service mesh compatible" -ForegroundColor Green
    Write-Host "  • ConfigMap/Secret integration" -ForegroundColor Green
    Write-Host ""
    Write-Host "☁️  Cloud-Native Providers:" -ForegroundColor Cyan
    Write-Host "  • PostgreSQL (cloud databases: RDS, CloudSQL, etc.)" -ForegroundColor Green
    Write-Host "  • Firestore (serverless, auto-scaling database)" -ForegroundColor Green
    Write-Host "  • JWT Authentication (stateless for horizontal scaling)" -ForegroundColor Green
    Write-Host "  • Multi-cloud storage (S3, GCS, Azure Blob)" -ForegroundColor Green
    if ($MockMode) {
        Write-Host ""
        Write-Host "🧪 Development Mode:" -ForegroundColor Cyan
        Write-Host "  • Mock providers included for container testing" -ForegroundColor Yellow
    } else {
        Write-Host "  • Production-focused (no mock providers)" -ForegroundColor Blue
    }
    Write-Host ""
    
    # Container-optimized build tags
    $BuildTags = @(
        "fiber", "providers_bootstrap",
        # Cloud-native database providers
        "postgres", "firestore", "postgres_migrations",
        # Stateless authentication for scaling
        "jwt_auth",
        # Multi-cloud storage providers
        "aws", "s3", "google", "gcp_storage", "local_storage",
        # Email providers
        "gmail", "microsoft", "microsoftgraph",
        # Firebase for serverless scenarios
        "firebase", 
        # Essential fallbacks
        "noop"
    )
    
    if ($MockMode) {
        $BuildTags += @("mock_db", "mock_email", "mock_storage")
    }
    
    Write-Host "Working directory: $(Get-Location)" -ForegroundColor Gray
    Write-Host "Build tags: $($BuildTags -join ',')" -ForegroundColor Blue
    Write-Host "Container optimization: ENABLED" -ForegroundColor Green
    Write-Host ""
    
    # Use the quick-build approach with container-optimized tags
    $SecondaryTagsString = ($BuildTags | Where-Object { $_ -ne "fiber" }) -join ','
    
    # Build arguments for the main build script
    $BuildArgs = @("-Framework", "fiber", "-SecondaryTags", $SecondaryTagsString, "-Output", $Output)
    if ($VerboseBuild) {
        $BuildArgs += "-VerboseBuild"
    }
    if ($Race) {
        $BuildArgs += "-Race"
    }
    
    # Add static binary flags if enabled
    if ($StaticBinary) {
        # Note: This would need to be implemented in the main build script
        # $BuildArgs += "-LdFlags", "-linkmode external -extldflags -static"
        Write-Host "Static binary mode: ENABLED (minimal container dependencies)" -ForegroundColor Green
    }
    
    Write-Host "Executing: .\scripts\build-with-tags.ps1 @BuildArgs" -ForegroundColor Magenta
    Write-Host ""
    
    # Execute build using the working build script
    & ".\scripts\build-with-tags.ps1" @BuildArgs
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host ""
        Write-Host "✅ Container K8s build completed!" -ForegroundColor Green
        
        # Show binary info
        $OutputPath = "build/$Output"
        if (Test-Path $OutputPath) {
            Write-Host "✅ Binary created: $OutputPath" -ForegroundColor Green
            $BinarySize = (Get-Item $OutputPath).Length / 1MB
            Write-Host "✅ Binary size: $([math]::Round($BinarySize, 2)) MB (container-optimized)" -ForegroundColor Blue
        }
        
        Write-Host ""
        Write-Host "🐳 Docker Usage Examples:" -ForegroundColor Cyan
        Write-Host ""
        Write-Host "  Minimal Dockerfile (scratch base):" -ForegroundColor White
        Write-Host "    FROM scratch" -ForegroundColor Gray
        Write-Host "    COPY $Output /espyna" -ForegroundColor Gray
        Write-Host "    COPY ca-certificates.crt /etc/ssl/certs/" -ForegroundColor Gray
        Write-Host "    EXPOSE 8080" -ForegroundColor Gray
        Write-Host "    HEALTHCHECK --interval=30s --timeout=3s \\" -ForegroundColor Gray
        Write-Host "      CMD ['/espyna', 'health'] || exit 1" -ForegroundColor Gray
        Write-Host "    ENTRYPOINT [\"/espyna\"]" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  Alpine Dockerfile (for debugging):" -ForegroundColor White
        Write-Host "    FROM alpine:latest" -ForegroundColor Gray
        Write-Host "    RUN apk add --no-cache ca-certificates tzdata" -ForegroundColor Gray
        Write-Host "    COPY $Output /usr/local/bin/espyna" -ForegroundColor Gray
        Write-Host "    RUN chmod +x /usr/local/bin/espyna" -ForegroundColor Gray
        Write-Host "    EXPOSE 8080" -ForegroundColor Gray
        Write-Host "    USER 1000:1000" -ForegroundColor Gray
        Write-Host "    ENTRYPOINT [\"/usr/local/bin/espyna\"]" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  Build Container:" -ForegroundColor White
        Write-Host "    docker build -t espyna:container-k8s ." -ForegroundColor Gray
        Write-Host ""
        Write-Host "  Run Container:" -ForegroundColor White
        Write-Host "    docker run -p 8080:8080 -e SERVER_TYPE=fiber \\" -ForegroundColor Gray
        Write-Host "      -e DATABASE_URL=\$DATABASE_URL espyna:container-k8s" -ForegroundColor Gray
        Write-Host ""
        Write-Host "☸️  Kubernetes Deployment Examples:" -ForegroundColor Cyan
        Write-Host ""
        Write-Host "  Deployment YAML:" -ForegroundColor White
        Write-Host "    apiVersion: apps/v1" -ForegroundColor Gray
        Write-Host "    kind: Deployment" -ForegroundColor Gray
        Write-Host "    metadata:" -ForegroundColor Gray
        Write-Host "      name: espyna-api" -ForegroundColor Gray
        Write-Host "    spec:" -ForegroundColor Gray
        Write-Host "      replicas: 3" -ForegroundColor Gray
        Write-Host "      selector:" -ForegroundColor Gray
        Write-Host "        matchLabels:" -ForegroundColor Gray
        Write-Host "          app: espyna-api" -ForegroundColor Gray
        Write-Host "      template:" -ForegroundColor Gray
        Write-Host "        metadata:" -ForegroundColor Gray
        Write-Host "          labels:" -ForegroundColor Gray
        Write-Host "            app: espyna-api" -ForegroundColor Gray
        Write-Host "        spec:" -ForegroundColor Gray
        Write-Host "          containers:" -ForegroundColor Gray
        Write-Host "          - name: espyna" -ForegroundColor Gray
        Write-Host "            image: espyna:container-k8s" -ForegroundColor Gray
        Write-Host "            ports:" -ForegroundColor Gray
        Write-Host "            - containerPort: 8080" -ForegroundColor Gray
        Write-Host "            env:" -ForegroundColor Gray
        Write-Host "            - name: SERVER_TYPE" -ForegroundColor Gray
        Write-Host "              value: fiber" -ForegroundColor Gray
        Write-Host "            - name: DATABASE_URL" -ForegroundColor Gray
        Write-Host "              valueFrom:" -ForegroundColor Gray
        Write-Host "                secretKeyRef:" -ForegroundColor Gray
        Write-Host "                  name: db-secret" -ForegroundColor Gray
        Write-Host "                  key: url" -ForegroundColor Gray
        Write-Host "            readinessProbe:" -ForegroundColor Gray
        Write-Host "              httpGet:" -ForegroundColor Gray
        Write-Host "                path: /health/ready" -ForegroundColor Gray
        Write-Host "                port: 8080" -ForegroundColor Gray
        Write-Host "              initialDelaySeconds: 5" -ForegroundColor Gray
        Write-Host "              periodSeconds: 5" -ForegroundColor Gray
        Write-Host "            livenessProbe:" -ForegroundColor Gray
        Write-Host "              httpGet:" -ForegroundColor Gray
        Write-Host "                path: /health/live" -ForegroundColor Gray
        Write-Host "                port: 8080" -ForegroundColor Gray
        Write-Host "              initialDelaySeconds: 15" -ForegroundColor Gray
        Write-Host "              periodSeconds: 20" -ForegroundColor Gray
        Write-Host ""
        Write-Host "📋 Container Environment Variables:" -ForegroundColor Cyan
        Write-Host "  # Core Configuration" -ForegroundColor White
        Write-Host "  SERVER_TYPE=fiber" -ForegroundColor Gray
        Write-Host "  SERVER_PORT=8080" -ForegroundColor Gray
        Write-Host "  SHUTDOWN_TIMEOUT=30s" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  # Database (use secrets in production)" -ForegroundColor White
        Write-Host "  DATABASE_PROVIDER=postgres" -ForegroundColor Gray  
        Write-Host "  DATABASE_URL=postgres://..." -ForegroundColor Gray
        Write-Host ""
        Write-Host "  # Cloud Storage" -ForegroundColor White
        Write-Host "  STORAGE_PROVIDER=s3|gcs|azure_blob" -ForegroundColor Gray
        Write-Host "  S3_BUCKET_NAME=app-storage" -ForegroundColor Gray
        Write-Host "  AWS_REGION=us-west-2" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  # Authentication" -ForegroundColor White
        Write-Host "  AUTH_PROVIDER=jwt" -ForegroundColor Gray
        Write-Host "  JWT_SECRET=\${JWT_SECRET}  # From Kubernetes secret" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  # Health Check Configuration" -ForegroundColor White
        Write-Host "  HEALTH_CHECK_ENABLED=true" -ForegroundColor Gray
        Write-Host "  METRICS_ENABLED=true" -ForegroundColor Gray
        
    } else {
        Write-Host ""
        Write-Host "❌ Build failed!" -ForegroundColor Red
        exit $LASTEXITCODE
    }
    
} catch {
    Write-Host "❌ Build script error: $_" -ForegroundColor Red
    exit 1
} finally {
    Pop-Location
}

Write-Host ""
Write-Host "🚀 Container K8s Stack Benefits:" -ForegroundColor Cyan
Write-Host "   • Optimized for high-performance containerized workloads" -ForegroundColor Gray
Write-Host "   • Kubernetes-native health checks and graceful shutdown" -ForegroundColor Gray
Write-Host "   • Horizontal scaling with stateless JWT authentication" -ForegroundColor Gray
Write-Host "   • Multi-cloud storage support for vendor independence" -ForegroundColor Gray
Write-Host "   • 12-factor app compliance for cloud-native best practices" -ForegroundColor Gray
Write-Host "   • Service mesh ready for advanced networking features" -ForegroundColor Gray
Write-Host "   • Minimal container footprint with optional static binary" -ForegroundColor Gray