#!/bin/bash
#
# Build Espyna server with multiple HTTP frameworks and hybrid cloud services
#
# This script creates a comprehensive build with:
# - All HTTP frameworks: Vanilla, Gin, and Fiber
# - Multiple database providers: PostgreSQL, Firestore, SQL Server
# - Multi-cloud authentication: Firebase, Azure AD, JWT
# - Hybrid email/storage providers: Google, Microsoft, AWS, local
# - Complete flexibility for runtime provider switching
#
# PARAMETERS:
#   -o, --output OUTPUT         Output binary name (default: espyna-multi-hybrid)
#   -v, --verbose              Enable verbose build output
#   -r, --race                 Enable race condition detection
#   -m, --mock-mode BOOL       Include mock providers for testing (default: true)
#   -h, --help                 Show this help message
#
# EXAMPLES:
#   ./build-multi-hybrid.sh
#       Full-featured build with all frameworks and providers
#
#   ./build-multi-hybrid.sh -v -r
#       Production build with all options and debugging
#
# NOTES:
#   This build configuration provides maximum flexibility:
#   - Runtime switching between HTTP frameworks
#   - Multiple database and auth providers
#   - Multi-cloud service support
#   - Comprehensive testing capabilities
#   Warning: This creates a larger binary but offers complete flexibility
#

set -euo pipefail

# Default values
OUTPUT="espyna-multi-hybrid"
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
    echo "Build Espyna server with multiple HTTP frameworks and hybrid cloud services."
    echo ""
    echo "OPTIONS:"
    echo "  -o, --output OUTPUT         Output binary name (default: espyna-multi-hybrid)"
    echo "  -v, --verbose              Enable verbose build output"
    echo "  -r, --race                 Enable race condition detection"
    echo "  -m, --mock-mode BOOL       Include mock providers for testing (default: true)"
    echo "  -h, --help                 Show this help message"
    echo ""
    echo "EXAMPLES:"
    echo "  $0"
    echo "  $0 -v -r"
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

echo -e "${CYAN}=== Espyna Multi-Framework Hybrid Build ===${NC}"
echo -e "${WHITE}Building comprehensive server with ALL provider capabilities:${NC}"
echo ""
echo -e "${CYAN}HTTP Framework (Runtime Configurable):${NC}"
echo -e "${GREEN}  • Gin (middleware-rich) - chosen for maximum flexibility${NC}"
echo -e "${YELLOW}  • Note: Use framework-specific builds for Vanilla or Fiber${NC}"
echo ""
echo -e "${CYAN}Database Providers:${NC}"
echo -e "${GREEN}  • PostgreSQL - ACID transactions, relational${NC}"
echo -e "${GREEN}  • Firestore - NoSQL, real-time, cloud-native${NC}"
echo -e "${GREEN}  • SQL Server - enterprise, Microsoft ecosystem${NC}"
echo ""
echo -e "${CYAN}Authentication Providers:${NC}"
echo -e "${GREEN}  • Firebase Auth - Google identity platform${NC}"
echo -e "${GREEN}  • Azure AD - Microsoft enterprise auth${NC}"
echo -e "${GREEN}  • JWT - stateless, self-hosted tokens${NC}"
echo ""
echo -e "${CYAN}Email/Communication:${NC}"
echo -e "${GREEN}  • Google Gmail API - G Suite integration${NC}"
echo -e "${GREEN}  • Microsoft Graph - Office 365 integration${NC}"
echo -e "${GREEN}  • SMTP - traditional email servers${NC}"
echo ""
echo -e "${CYAN}Storage Providers:${NC}"
echo -e "${GREEN}  • Google Cloud Storage - GCP object storage${NC}"
echo -e "${GREEN}  • Azure Blob Storage - Azure cloud storage${NC}"
echo -e "${GREEN}  • AWS S3 - Amazon object storage${NC}"
echo -e "${GREEN}  • Local Filesystem - self-hosted storage${NC}"

if [[ "$MOCK_MODE" == true ]]; then
    echo ""
    echo -e "${YELLOW}  • Mock providers included for comprehensive testing${NC}"
fi
echo ""

# Multi-hybrid build tags - include all secondary providers
SECONDARY_BUILD_TAGS=(
    # Bootstrap provider system
    "providers_bootstrap"
    # Database providers
    "postgres" "firestore" "postgres_migrations"
    # Auth providers
    "firebase" "jwt_auth" "microsoft"
    # Cloud providers
    "google" "azure" "aws"
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
echo -e "${BLUE}Secondary build tags: $(IFS=','; echo "${SECONDARY_BUILD_TAGS[*]}")${NC}"
echo ""

# Use the gin framework with comprehensive provider support
SECONDARY_TAGS_STRING=$(IFS=','; echo "${SECONDARY_BUILD_TAGS[*]}")

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
    echo -e "${GREEN}✓ Build completed successfully!${NC}"
    echo -e "${GREEN}✓ Binary created: $OUTPUT_PATH${NC}"
    
    if [[ -f "$OUTPUT_PATH" ]]; then
        BINARY_SIZE_BYTES=$(stat -f%z "$OUTPUT_PATH" 2>/dev/null || stat -c%s "$OUTPUT_PATH" 2>/dev/null || echo "0")
        BINARY_SIZE_MB=$((BINARY_SIZE_BYTES / 1024 / 1024))
        echo -e "${BLUE}✓ Binary size: ${BINARY_SIZE_MB} MB${NC}"
        echo -e "${YELLOW}  Note: Larger binary due to comprehensive provider support${NC}"
    fi
    
    echo ""
    echo -e "${CYAN}Runtime Configuration Examples:${NC}"
    echo ""
    echo -e "${WHITE}  Gin Framework (Current Build):${NC}"
    echo -e "${GRAY}    SERVER_TYPE=gin ./$OUTPUT_PATH${NC}"
    echo ""
    echo -e "${WHITE}  Google Cloud Stack:${NC}"
    echo -e "${GRAY}    SERVER_TYPE=gin DATABASE_PROVIDER=firestore${NC}"
    echo -e "${GRAY}    AUTH_PROVIDER=firebase EMAIL_PROVIDER=gmail${NC}"
    echo -e "${GRAY}    STORAGE_PROVIDER=gcs ./$OUTPUT_PATH${NC}"
    echo ""
    echo -e "${WHITE}  Microsoft Enterprise Stack:${NC}"
    echo -e "${GRAY}    SERVER_TYPE=gin DATABASE_PROVIDER=postgres${NC}"
    echo -e "${GRAY}    AUTH_PROVIDER=microsoft EMAIL_PROVIDER=microsoftgraph${NC}"
    echo -e "${GRAY}    STORAGE_PROVIDER=azure_blob ./$OUTPUT_PATH${NC}"
    echo ""
    echo -e "${WHITE}  Self-hosted Stack:${NC}"
    echo -e "${GRAY}    SERVER_TYPE=gin DATABASE_PROVIDER=postgres${NC}"
    echo -e "${GRAY}    AUTH_PROVIDER=jwt_auth EMAIL_PROVIDER=smtp${NC}"
    echo -e "${GRAY}    STORAGE_PROVIDER=local ./$OUTPUT_PATH${NC}"
    echo ""
    echo -e "${WHITE}  Hybrid Cloud:${NC}"
    echo -e "${GRAY}    DATABASE_PROVIDER=postgres AUTH_PROVIDER=firebase${NC}"
    echo -e "${GRAY}    EMAIL_PROVIDER=microsoftgraph STORAGE_PROVIDER=s3 ./$OUTPUT_PATH${NC}"
    
else
    echo -e "${RED}✗ Build failed with exit code: $?${NC}"
    exit 1
fi

echo ""
echo -e "${CYAN}Multi-Hybrid Stack Benefits:${NC}"
echo -e "${GRAY}  • Complete provider flexibility - choose providers at runtime${NC}"
echo -e "${GRAY}  • Multi-cloud deployment strategies${NC}"
echo -e "${GRAY}  • Gradual migration between providers${NC}"
echo -e "${GRAY}  • A/B testing different technology stacks${NC}"
echo -e "${GRAY}  • Comprehensive testing with all provider combinations${NC}"
echo -e "${GRAY}  • Future-proof architecture with provider agnostic design${NC}"
echo -e "${YELLOW}  • Note: Build separate binaries for Vanilla/Fiber frameworks${NC}"