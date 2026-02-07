#!/bin/bash
#
# Build Espyna server with Gin HTTP framework and Microsoft ecosystem (SQL Server + Microsoft Graph)
#
# DESCRIPTION:
#   This script creates a specialized build with:
#   - Gin HTTP framework for flexible web API development
#   - Microsoft SQL Server as the primary database provider
#   - Microsoft Graph API for email, calendar, and user management
#   - Azure services integration (storage, authentication, etc.)
#
# PARAMETERS:
#   -o, --output OUTPUT       Output binary name (default: espyna-gin-microsoft)
#   -v, --verbose            Enable verbose build output
#   -r, --race               Enable race condition detection
#   -m, --mock-mode BOOL     Include mock providers for testing (default: true)
#   -h, --help               Show this help message
#
# EXAMPLES:
#   ./build-gin-microsoft.sh
#       Basic build with Gin + Microsoft stack
#
#   ./build-gin-microsoft.sh -v -r --mock-mode=false
#       Production build with verbose output and race detection
#
# NOTES:
#   This build configuration is optimized for:
#   - Enterprise REST API development with Gin
#   - Microsoft 365 integration with Graph API  
#   - Enterprise database with SQL Server
#   - Azure cloud services integration

set -euo pipefail

# Default values
OUTPUT="espyna-gin-microsoft"
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
    echo "Build Espyna server with Gin HTTP framework and Microsoft ecosystem"
    echo ""
    echo "OPTIONS:"
    echo "  -o, --output OUTPUT       Output binary name (default: espyna-gin-microsoft)"
    echo "  -v, --verbose            Enable verbose build output"
    echo "  -r, --race               Enable race condition detection"
    echo "  -m, --mock-mode BOOL     Include mock providers for testing (default: true)"
    echo "  -h, --help               Show this help message"
    echo ""
    echo "EXAMPLES:"
    echo "  $0"
    echo "  $0 -v -r --mock-mode=false"
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
            if [[ "$2" == "true" ]]; then
                MOCK_MODE=true
            elif [[ "$2" == "false" ]]; then
                MOCK_MODE=false
            else
                echo -e "${RED}Error: Invalid mock-mode value '$2'. Use 'true' or 'false'${NC}"
                exit 1
            fi
            shift 2
            ;;
        --mock-mode=*)
            VALUE="${1#*=}"
            if [[ "$VALUE" == "true" ]]; then
                MOCK_MODE=true
            elif [[ "$VALUE" == "false" ]]; then
                MOCK_MODE=false
            else
                echo -e "${RED}Error: Invalid mock-mode value '$VALUE'. Use 'true' or 'false'${NC}"
                exit 1
            fi
            shift
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            echo -e "${RED}Error: Unknown option $1${NC}"
            show_help
            exit 1
            ;;
    esac
done

# Set build directory to packages/espyna
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ESPYNA_DIR="$(dirname "$(dirname "$(dirname "$SCRIPT_DIR")")")"
cd "$ESPYNA_DIR"

echo -e "${CYAN}=== Espyna Gin + Microsoft Build ===${NC}"
echo -e "${WHITE}Building specialized server with:${NC}"
echo -e "${GREEN}  • HTTP Framework: Gin (flexible API development)${NC}"
echo -e "${GREEN}  • Database: Microsoft SQL Server (enterprise)${NC}"
echo -e "${GREEN}  • Email/Calendar: Microsoft Graph API${NC}"
echo -e "${GREEN}  • Authentication: Azure Active Directory${NC}"
echo -e "${GREEN}  • Storage Provider: Azure Blob Storage${NC}"
if [[ "$MOCK_MODE" == true ]]; then
    echo -e "${YELLOW}  • Mock providers included for testing${NC}"
fi
echo ""

# Build tags configuration - comprehensive working tag set
BUILD_TAGS=("gin" "providers_bootstrap" "postgres" "microsoft" "microsoftgraph" "postgres_migrations")
if [[ "$MOCK_MODE" == true ]]; then
    BUILD_TAGS+=("mock_db" "mock_email" "mock_storage")
fi
# Always include essential providers for complete functionality
BUILD_TAGS+=("local_storage" "noop" "google" "firebase" "firestore")

echo -e "${GRAY}Working directory: $(pwd)${NC}"
echo -e "${BLUE}Build tags: $(IFS=','; echo "${BUILD_TAGS[*]}")${NC}"

# Prepare build arguments
TAGS_STRING=$(IFS=','; echo "${BUILD_TAGS[*]}")
BUILD_ARGS=()
BUILD_ARGS+=(-tags "$TAGS_STRING")

if [[ "$VERBOSE_BUILD" == true ]]; then
    BUILD_ARGS+=(-v)
    echo -e "${YELLOW}Verbose build enabled${NC}"
fi

if [[ "$RACE" == true ]]; then
    BUILD_ARGS+=(-race)
    echo -e "${YELLOW}Race detection enabled${NC}"
fi

# Ensure build directory exists
BUILD_DIR="build"
if [[ ! -d "$BUILD_DIR" ]]; then
    mkdir -p "$BUILD_DIR"
    echo -e "${BLUE}Created build directory: $BUILD_DIR${NC}"
fi

# Set output path
OUTPUT_PATH="$BUILD_DIR/$OUTPUT"
BUILD_ARGS+=(-o "$OUTPUT_PATH")
BUILD_ARGS+=(./cmd/server)

echo -e "${MAGENTA}Executing: go build ${BUILD_ARGS[*]}${NC}"
echo ""

# Execute build
if go build "${BUILD_ARGS[@]}"; then
    echo -e "${GREEN}✓ Build completed successfully!${NC}"
    echo -e "${GREEN}✓ Binary created: $OUTPUT_PATH${NC}"
    
    if [[ -f "$OUTPUT_PATH" ]]; then
        BINARY_SIZE_BYTES=$(stat -f%z "$OUTPUT_PATH" 2>/dev/null || stat -c%s "$OUTPUT_PATH" 2>/dev/null || echo "0")
        BINARY_SIZE_MB=$((BINARY_SIZE_BYTES / 1024 / 1024))
        echo -e "${BLUE}✓ Binary size: ${BINARY_SIZE_MB} MB${NC}"
    fi
    
    echo ""
    echo -e "${CYAN}Usage Examples:${NC}"
    echo -e "${WHITE}  Development (with mock data):${NC}"
    echo -e "${GRAY}    MOCK_MODE=true MOCK_BUSINESS_TYPE=office_leasing ./$OUTPUT_PATH${NC}"
    echo ""
    echo -e "${WHITE}  Production (Microsoft services):${NC}"
    echo -e "${GRAY}    AZURE_CLIENT_ID=your-client-id ./$OUTPUT_PATH${NC}"
    echo -e "${GRAY}    AZURE_TENANT_ID=your-tenant-id ./$OUTPUT_PATH${NC}"
    echo -e "${GRAY}    SQLSERVER_CONNECTION_STRING='server=...;database=...' ./$OUTPUT_PATH${NC}"
    echo ""
    echo -e "${CYAN}Environment Variables:${NC}"
    echo -e "${GRAY}  SERVER_TYPE=gin                        # HTTP framework${NC}"
    echo -e "${GRAY}  SERVER_PORT=8081                       # Server port${NC}"
    echo -e "${GRAY}  AZURE_CLIENT_ID=client-id              # Azure app registration${NC}"
    echo -e "${GRAY}  AZURE_TENANT_ID=tenant-id              # Azure tenant${NC}"
    echo -e "${GRAY}  AZURE_CLIENT_SECRET=secret              # Azure app secret${NC}"
    echo -e "${GRAY}  SQLSERVER_CONNECTION_STRING=conn-str   # SQL Server connection${NC}"
    echo -e "${GRAY}  MOCK_MODE=true                          # Enable mock providers${NC}"
    echo -e "${GRAY}  MOCK_BUSINESS_TYPE=office_leasing       # Mock data type${NC}"
    
else
    echo -e "${RED}✗ Build failed${NC}"
    exit 1
fi

echo ""
echo -e "${CYAN}Gin + Microsoft Stack Benefits:${NC}"
echo -e "${GRAY}  • Flexible REST API development with Gin middleware${NC}"
echo -e "${GRAY}  • Enterprise-grade database with SQL Server${NC}"
echo -e "${GRAY}  • Microsoft 365 integration with Graph API${NC}"
echo -e "${GRAY}  • Azure Active Directory for enterprise auth${NC}"
echo -e "${GRAY}  • Seamless Office 365 and Teams integration${NC}"