# Example: Run Espyna with PostgreSQL database and Gin framework
# This demonstrates the new environment-based configuration system

# Set provider configuration
$env:CONFIG_SERVER_FRAMEWORK = "gin"
$env:CONFIG_SERVER_PORT = "8081"
$env:CONFIG_DATABASE_PROVIDER = "postgres"
$env:CONFIG_AUTH_PROVIDER = "firebase_auth"
$env:CONFIG_EMAIL_PROVIDER = "gmail"
$env:CONFIG_STORAGE_PROVIDER = "gcp_storage"

# Set business type (not part of build tags)
$env:BUSINESS_TYPE = "education"

# Set PostgreSQL connection details
$env:POSTGRES_HOST = "localhost"
$env:POSTGRES_PORT = "5432"
$env:POSTGRES_NAME = "espyna_dev"
$env:POSTGRES_USER = "postgres"
$env:POSTGRES_PASSWORD = "password123"

# Set Firebase Auth details
$env:FIREBASE_AUTH_PROJECT_ID = "your-project-id"
$env:FIREBASE_AUTH_CREDENTIALS_PATH = "./credentials/firebase-admin.json"

Write-Host "🔧 Starting Espyna with PostgreSQL + Gin + Firebase Auth + Gmail + GCP Storage" -ForegroundColor Green

# Run the dynamic configuration script
& ".\scripts\serve\run-dynamic-config.ps1" -Mode development -HotReload