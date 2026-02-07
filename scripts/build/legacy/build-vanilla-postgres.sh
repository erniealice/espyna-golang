#!/bin/bash
#
# Build Espyna server with Vanilla HTTP framework and PostgreSQL + Self-hosted services
#
# This script creates a specialized build with:
# - Vanilla HTTP framework for lightweight, standard library approach
# - PostgreSQL as the primary database provider
# - Self-hosted authentication with JWT
# - Local file storage and SMTP email
# - Minimal dependencies for containerized deployment
#
# PARAMETERS:
#   -o, --output OUTPUT          Output binary name (default: espyna-vanilla-postgres)
#   -v, --verbose               Enable verbose build output
#   -r, --race                  Enable race condition detection
#   -m, --mock-mode             Include mock providers for testing (default: true)
#   -h, --help                  Show this help message
#
# EXAMPLES:
#   ./build-vanilla-postgres.sh
#       Basic build with Vanilla + PostgreSQL stack
#
#   ./build-vanilla-postgres.sh -v -r --no-mock-mode
#       Production build with verbose output and race detection
#
# NOTES:
#   This build configuration is optimized for:
#   - Minimal dependencies and small binary size
#   - Self-hosted deployment with Docker/Kubernetes
#   - PostgreSQL for reliable ACID transactions
#   - JWT-based authentication for stateless scaling

set -euo pipefail

# Default values
OUTPUT="espyna-vanilla-postgres"
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
    echo "Build Espyna server with Vanilla HTTP framework and PostgreSQL + Self-hosted services."
    echo ""
    echo "OPTIONS:"
    echo "  -o, --output OUTPUT          Output binary name (default: espyna-vanilla-postgres)"
    echo "  -v, --verbose               Enable verbose build output"
    echo "  -r, --race                  Enable race condition detection"
    echo "  -m, --mock-mode             Include mock providers for testing (default: true)"
    echo "      --no-mock-mode          Disable mock providers (production mode)"
    echo "  -h, --help                  Show this help message"
    echo ""
    echo "EXAMPLES:"
    echo "  $0                          # Basic build with Vanilla + PostgreSQL stack"
    echo "  $0 -v -r --no-mock-mode     # Production build with verbose output and race detection"
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
            MOCK_MODE=true
            shift
            ;;
        --no-mock-mode)
            MOCK_MODE=false
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

echo -e "${CYAN}=== Espyna Vanilla + PostgreSQL Build ===${NC}"
echo -e "${WHITE}Building specialized server with:${NC}"
echo -e "  ${GREEN}• HTTP Framework: Vanilla (standard library)${NC}"
echo -e "  ${GREEN}• Database: PostgreSQL (ACID transactions)${NC}"
echo -e "  ${GREEN}• Authentication: JWT (stateless)${NC}"
echo -e "  ${GREEN}• Email Provider: SMTP (configurable)${NC}"
echo -e "  ${GREEN}• Storage Provider: Local filesystem${NC}"

if [[ "$MOCK_MODE" == true ]]; then
    echo -e "  ${YELLOW}• Mock providers included for testing${NC}"
fi
echo ""

# Build tags configuration - comprehensive working tag set
BUILD_TAGS=("vanilla" "providers_bootstrap" "postgres" "jwt_auth" "postgres_migrations")

if [[ "$MOCK_MODE" == true ]]; then
    BUILD_TAGS+=("mock_db" "mock_email" "mock_storage")
fi

# Always include essential providers for complete functionality
BUILD_TAGS+=("local_storage" "noop" "google" "firebase" "firestore" "microsoft")

echo -e "${GRAY}Working directory: $(pwd)${NC}"
echo -e "${BLUE}Build tags: $(IFS=','; echo "${BUILD_TAGS[*]}")${NC}"

# Prepare build arguments
BUILD_ARGS=()
BUILD_ARGS+=("-tags" "$(IFS=','; echo "${BUILD_TAGS[*]}")")

if [[ "$VERBOSE_BUILD" == true ]]; then
    BUILD_ARGS+=("-v")
    echo -e "${YELLOW}Verbose build enabled${NC}"
fi

if [[ "$RACE" == true ]]; then
    BUILD_ARGS+=("-race")
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
BUILD_ARGS+=("-o" "$OUTPUT_PATH")
BUILD_ARGS+=("./cmd/server")

echo -e "${MAGENTA}Executing: go build $(printf '%s ' "${BUILD_ARGS[@]}")${NC}"
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
    echo -e "${GRAY}    MOCK_MODE=true MOCK_BUSINESS_TYPE=fitness_center ./$OUTPUT_PATH${NC}"
    echo ""
    echo -e "${WHITE}  Production (self-hosted):${NC}"
    echo -e "${GRAY}    DATABASE_URL='postgres://user:pass@localhost/db' ./$OUTPUT_PATH${NC}"
    echo -e "${GRAY}    JWT_SECRET=your-secret-key ./$OUTPUT_PATH${NC}"
    echo -e "${GRAY}    SMTP_HOST=mail.example.com ./$OUTPUT_PATH${NC}"
    echo ""
    echo -e "${CYAN}Environment Variables:${NC}"
    echo -e "${GRAY}  SERVER_PORT=8080                    # Server port${NC}"
    echo -e "${GRAY}  DATABASE_URL=postgres://...         # PostgreSQL connection${NC}"
    echo -e "${GRAY}  JWT_SECRET=secret-key               # JWT signing key${NC}"
    echo -e "${GRAY}  SMTP_HOST=mail.example.com          # Email server${NC}"
    echo -e "${GRAY}  SMTP_PORT=587                       # Email port${NC}"
    echo -e "${GRAY}  STORAGE_PATH=/app/uploads           # File storage path${NC}"
    echo -e "${GRAY}  MOCK_MODE=true                      # Enable mock providers${NC}"
    echo -e "${GRAY}  MOCK_BUSINESS_TYPE=fitness_center   # Mock data type${NC}"
    
else
    echo -e "${RED}✗ Build failed with exit code: $?${NC}"
    exit 1
fi

echo ""
echo -e "${CYAN}Vanilla + PostgreSQL Stack Benefits:${NC}"
echo -e "${GRAY}  • Minimal binary size with standard library HTTP${NC}"
echo -e "${GRAY}  • Rock-solid ACID transactions with PostgreSQL${NC}"
echo -e "${GRAY}  • Stateless scaling with JWT authentication${NC}"
echo -e "${GRAY}  • Self-hosted deployment with Docker/K8s${NC}"
echo -e "${GRAY}  • No vendor lock-in with open-source stack${NC}"