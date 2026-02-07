#!/bin/bash
#
# Build enterprise-grade Espyna server with comprehensive business features and integrations
#
# This script creates a full-featured enterprise build with:
# - Gin HTTP framework for flexible middleware and API development
# - Multiple database support (PostgreSQL primary, with Firestore backup)
# - Multi-provider authentication (Azure AD, Firebase, JWT fallback)
# - Enterprise email integration (Microsoft Graph, Google Workspace)
# - Multi-cloud storage support (Azure Blob, Google Cloud, AWS S3)
# - Enhanced security, monitoring, and compliance features
#
# PARAMETERS:
#   -o, --output OUTPUT         Output binary name (default: espyna-enterprise-complete)
#   -v, --verbose              Enable verbose build output
#   -r, --race                 Enable race condition detection
#   -m, --mock-mode BOOL       Include mock providers for testing (default: true)
#   -h, --help                 Show this help message
#
# EXAMPLES:
#   ./build-enterprise-complete.sh
#       Full enterprise build with all providers
#
#   ./build-enterprise-complete.sh -v -r --mock-mode false
#       Production enterprise build with debugging
#
# NOTES:
#   This build configuration is optimized for:
#   - Large enterprise organizations
#   - Multi-cloud hybrid deployments
#   - High-availability production systems
#   - Comprehensive integration requirements
#   - Advanced security and compliance needs
#

set -euo pipefail

# Default values
OUTPUT="espyna-enterprise-complete"
VERBOSE_BUILD=false
RACE=false
MOCK_MODE=true

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
GRAY='\033[0;90m'
MAGENTA='\033[0;35m'
WHITE='\033[1;37m'
NC='\033[0m' # No Color

# Function to show usage
show_help() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Build enterprise-grade Espyna server with comprehensive business features and integrations."
    echo ""
    echo "OPTIONS:"
    echo "  -o, --output OUTPUT         Output binary name (default: espyna-enterprise-complete)"
    echo "  -v, --verbose              Enable verbose build output"
    echo "  -r, --race                 Enable race condition detection"
    echo "  -m, --mock-mode BOOL       Include mock providers for testing (default: true)"
    echo "  -h, --help                 Show this help message"
    echo ""
    echo "EXAMPLES:"
    echo "  $0"
    echo "  $0 -v -r --mock-mode false"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -o|--output)
            OUTPUT="$2"
            shift 2
            ;;
        -v|--verbose)
            VERBOSE_BUILD=true
            shift
            ;;
        -r|--race)
            RACE=true
            shift
            ;;
        -m|--mock-mode)
            if [[ "$2" == "false" || "$2" == "0" ]]; then
                MOCK_MODE=false
            else
                MOCK_MODE=true
            fi
            shift 2
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            echo "Error: Unknown option $1"
            show_help
            exit 1
            ;;
    esac
done

# Set build directory to packages/espyna
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ESPYNA_DIR="$(dirname "$(dirname "$(dirname "$SCRIPT_DIR")")")"
cd "$ESPYNA_DIR"

echo -e "${CYAN}=== Espyna Enterprise Complete Build ===${NC}"
echo -e "${WHITE}Building full-featured enterprise server with:${NC}"
echo ""
echo -e "${CYAN}🌐 HTTP Framework:${NC}"
echo -e "${GREEN}  • Gin (enterprise API development with middleware)${NC}"
echo ""
echo -e "${CYAN}🗄️  Database Providers:${NC}"
echo -e "${GREEN}  • PostgreSQL (primary enterprise database)${NC}"
echo -e "${GREEN}  • Firestore (cloud-native backup/secondary)${NC}"
echo -e "${GREEN}  • Migration support (database versioning)${NC}"
echo ""
echo -e "${CYAN}🔐 Authentication Providers:${NC}"
echo -e "${GREEN}  • Azure Active Directory (enterprise SSO)${NC}"
echo -e "${GREEN}  • Firebase Auth (modern web/mobile)${NC}"
echo -e "${GREEN}  • JWT (stateless fallback)${NC}"
echo ""
echo -e "${CYAN}📧 Email & Communication:${NC}"
echo -e "${GREEN}  • Microsoft Graph API (Office 365, Teams integration)${NC}"
echo -e "${GREEN}  • Google Workspace API (Gmail, Calendar)${NC}"
echo ""
echo -e "${CYAN}☁️  Storage Providers:${NC}"
echo -e "${GREEN}  • Azure Blob Storage (Microsoft ecosystem)${NC}"
echo -e "${GREEN}  • Google Cloud Storage (Google ecosystem)${NC}"
echo -e "${GREEN}  • AWS S3 (Amazon ecosystem)${NC}"
echo -e "${GREEN}  • Local storage (on-premises fallback)${NC}"

if [[ "$MOCK_MODE" == true ]]; then
    echo ""
    echo -e "${CYAN}🧪 Development Features:${NC}"
    echo -e "${YELLOW}  • Mock providers for comprehensive testing${NC}"
    echo -e "${YELLOW}  • Multi-business-type support${NC}"
fi
echo ""

# Enterprise-complete build tags - include everything
BUILD_TAGS=(
    "gin" "providers_bootstrap"
    # Database providers
    "postgres" "firestore" "postgres_migrations"
    # Authentication providers  
    "firebase" "microsoft" "jwt_auth"
    # Cloud service providers
    "google" "aws" "azure"
    # Storage providers
    "gcp_storage" "s3" "local_storage"
    # Email providers
    "gmail" "microsoftgraph"
    # Mock providers (required dependencies)
    "mock_db" "mock_email" "mock_storage"
    # Essential fallbacks
    "noop"
)

echo -e "${GRAY}Working directory: $(pwd)${NC}"
echo -e "${BLUE}Build tags: $(IFS=','; echo "${BUILD_TAGS[*]}")${NC}"
echo -e "${BLUE}Total providers: ${#BUILD_TAGS[@]}${NC}"
echo ""

# Use the quick-build approach with comprehensive tag set
SECONDARY_TAGS=()
for tag in "${BUILD_TAGS[@]}"; do
    if [[ "$tag" != "gin" ]]; then
        SECONDARY_TAGS+=("$tag")
    fi
done
SECONDARY_TAGS_STRING=$(IFS=','; echo "${SECONDARY_TAGS[*]}")

echo -e "${MAGENTA}Executing: ./scripts/build/build-with-tags.sh -f gin -s '$SECONDARY_TAGS_STRING' -o $OUTPUT${NC}"
echo ""

# Build arguments for the main build script
BUILD_ARGS=(-f gin -s "$SECONDARY_TAGS_STRING" -o "$OUTPUT")
if [[ "$VERBOSE_BUILD" == true ]]; then
    BUILD_ARGS+=(-v)
fi
if [[ "$RACE" == true ]]; then
    BUILD_ARGS+=(-r)
fi

# Execute build using the working build script
if ./scripts/build/build-with-tags.sh "${BUILD_ARGS[@]}"; then
    echo ""
    echo -e "${GREEN}✅ Enterprise complete build finished!${NC}"
    
    # Show binary info
    OUTPUT_PATH="build/$OUTPUT"
    if [[ -f "$OUTPUT_PATH" ]]; then
        echo -e "${GREEN}✅ Binary created: $OUTPUT_PATH${NC}"
        BINARY_SIZE_BYTES=$(stat -f%z "$OUTPUT_PATH" 2>/dev/null || stat -c%s "$OUTPUT_PATH" 2>/dev/null || echo "0")
        BINARY_SIZE_MB=$((BINARY_SIZE_BYTES / 1024 / 1024))
        echo -e "${BLUE}✅ Binary size: ${BINARY_SIZE_MB} MB (comprehensive feature set)${NC}"
    fi
    
    echo ""
    echo -e "${CYAN}🏢 Enterprise Deployment Scenarios:${NC}"
    echo ""
    echo -e "${WHITE}  Microsoft Enterprise Stack:${NC}"
    echo -e "${GRAY}    SERVER_TYPE=gin \\${NC}"
    echo -e "${GRAY}    DATABASE_PROVIDER=postgres \\${NC}"
    echo -e "${GRAY}    AUTH_PROVIDER=microsoft \\${NC}"
    echo -e "${GRAY}    EMAIL_PROVIDER=microsoftgraph \\${NC}"
    echo -e "${GRAY}    STORAGE_PROVIDER=azure_blob \\${NC}"
    echo -e "${GRAY}    ./$OUTPUT_PATH${NC}"
    echo ""
    echo -e "${WHITE}  Google Workspace Stack:${NC}"
    echo -e "${GRAY}    SERVER_TYPE=gin \\${NC}"
    echo -e "${GRAY}    DATABASE_PROVIDER=firestore \\${NC}"
    echo -e "${GRAY}    AUTH_PROVIDER=firebase \\${NC}"
    echo -e "${GRAY}    EMAIL_PROVIDER=gmail \\${NC}"
    echo -e "${GRAY}    STORAGE_PROVIDER=gcs \\${NC}"
    echo -e "${GRAY}    ./$OUTPUT_PATH${NC}"
    echo ""
    echo -e "${WHITE}  Multi-Cloud Hybrid:${NC}"
    echo -e "${GRAY}    SERVER_TYPE=gin \\${NC}"
    echo -e "${GRAY}    DATABASE_PROVIDER=postgres \\${NC}"
    echo -e "${GRAY}    AUTH_PROVIDER=microsoft \\${NC}"
    echo -e "${GRAY}    EMAIL_PROVIDER=gmail \\${NC}"
    echo -e "${GRAY}    STORAGE_PROVIDER=s3 \\${NC}"
    echo -e "${GRAY}    ./$OUTPUT_PATH${NC}"
    echo ""
    echo -e "${WHITE}  High-Availability Setup:${NC}"
    echo -e "${GRAY}    SERVER_TYPE=gin \\${NC}"
    echo -e "${GRAY}    DATABASE_PROVIDER=postgres \\${NC}"
    echo -e "${GRAY}    DATABASE_BACKUP_PROVIDER=firestore \\${NC}"
    echo -e "${GRAY}    AUTH_PROVIDER=microsoft \\${NC}"
    echo -e "${GRAY}    AUTH_FALLBACK_PROVIDER=jwt \\${NC}"
    echo -e "${GRAY}    ./$OUTPUT_PATH${NC}"
    echo ""
    echo -e "${CYAN}📋 Key Environment Variables:${NC}"
    echo -e "${WHITE}  # Core Configuration${NC}"
    echo -e "${GRAY}  SERVER_TYPE=gin${NC}"
    echo -e "${GRAY}  SERVER_PORT=8080${NC}"
    echo -e "${GRAY}  LOG_LEVEL=info${NC}"
    echo ""
    echo -e "${WHITE}  # Provider Selection (runtime switchable)${NC}"
    echo -e "${GRAY}  DATABASE_PROVIDER=postgres|firestore${NC}"
    echo -e "${GRAY}  AUTH_PROVIDER=microsoft|firebase|jwt${NC}"
    echo -e "${GRAY}  EMAIL_PROVIDER=microsoftgraph|gmail${NC}"
    echo -e "${GRAY}  STORAGE_PROVIDER=azure_blob|gcs|s3|local${NC}"
    echo ""
    echo -e "${WHITE}  # Microsoft Integration${NC}"
    echo -e "${GRAY}  AZURE_CLIENT_ID=your-client-id${NC}"
    echo -e "${GRAY}  AZURE_TENANT_ID=your-tenant-id${NC}"
    echo -e "${GRAY}  AZURE_CLIENT_SECRET=your-secret${NC}"
    echo ""
    echo -e "${WHITE}  # Google Integration${NC}"
    echo -e "${GRAY}  GOOGLE_APPLICATION_CREDENTIALS=path/to/service-key.json${NC}"
    echo -e "${GRAY}  FIREBASE_PROJECT_ID=your-project-id${NC}"
    echo ""
    echo -e "${WHITE}  # Database Connections${NC}"
    echo -e "${GRAY}  DATABASE_URL=postgres://...${NC}"
    echo -e "${GRAY}  FIRESTORE_PROJECT_ID=backup-project${NC}"
    echo ""
    echo -e "${WHITE}  # Development/Testing${NC}"
    echo -e "${GRAY}  MOCK_MODE=true${NC}"
    echo -e "${GRAY}  MOCK_BUSINESS_TYPE=office_leasing${NC}"
    
else
    echo ""
    echo -e "${RED}❌ Build failed!${NC}"
    exit 1
fi

echo ""
echo -e "${CYAN}🏆 Enterprise Complete Stack Benefits:${NC}"
echo -e "${GRAY}   • Maximum flexibility - all providers included${NC}"
echo -e "${GRAY}   • Runtime provider switching - no recompilation needed${NC}"
echo -e "${GRAY}   • Multi-cloud deployment support - avoid vendor lock-in${NC}"
echo -e "${GRAY}   • Enterprise SSO integration - Azure AD, Google Workspace${NC}"
echo -e "${GRAY}   • High availability - primary + backup provider configurations${NC}"
echo -e "${GRAY}   • Comprehensive testing - mock providers for all services${NC}"
echo -e "${GRAY}   • Future-proof architecture - easy to add new providers${NC}"