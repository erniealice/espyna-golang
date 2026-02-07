#!/bin/bash
#
# Build development-optimized Espyna server with enhanced debugging and testing features
#
# This script creates a developer-friendly build optimized for local development with:
# - Gin HTTP framework for hot-reload and middleware flexibility
# - Comprehensive mock providers for offline development
# - Enhanced logging and debugging capabilities
# - Race condition detection enabled by default
# - All business types and test data included
# - Multiple provider support for testing integration scenarios
#
# PARAMETERS:
#   -o, --output OUTPUT          Output binary name (default: espyna-dev-debug)
#   -v, --verbose               Enable verbose build output (default: true for development)
#   -r, --race                  Enable race condition detection (default: true for development)
#   -s, --symbol-table          Include debugging symbols (default: true)
#   -h, --help                  Show this help message
#
# EXAMPLES:
#   ./build-development-debug.sh
#       Development build with all debugging features
#
#   ./build-development-debug.sh --no-symbol-table
#       Development build without debug symbols (smaller size)
#
# NOTES:
#   This build configuration is optimized for:
#   - Local development and testing
#   - API development and debugging
#   - Integration testing with multiple providers
#   - Hot-reload development workflows
#   - Comprehensive test data scenarios

set -euo pipefail

# Default values
OUTPUT="espyna-dev-debug"
VERBOSE_BUILD=true
RACE=true
SYMBOL_TABLE=true

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
    echo "Build development-optimized Espyna server with enhanced debugging and testing features"
    echo ""
    echo "OPTIONS:"
    echo "  -o, --output OUTPUT          Output binary name (default: espyna-dev-debug)"
    echo "  -v, --verbose               Enable verbose build output (default: true)"
    echo "      --no-verbose            Disable verbose build output"
    echo "  -r, --race                  Enable race condition detection (default: true)"
    echo "      --no-race               Disable race condition detection"
    echo "  -s, --symbol-table          Include debugging symbols (default: true)"
    echo "      --no-symbol-table       Exclude debugging symbols (smaller size)"
    echo "  -h, --help                  Show this help message"
    echo ""
    echo "EXAMPLES:"
    echo "  $0"
    echo "  $0 --no-symbol-table"
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
        --no-verbose)
            VERBOSE_BUILD=false
            shift
            ;;
        -r|--race)
            RACE=true
            shift
            ;;
        --no-race)
            RACE=false
            shift
            ;;
        -s|--symbol-table)
            SYMBOL_TABLE=true
            shift
            ;;
        --no-symbol-table)
            SYMBOL_TABLE=false
            shift
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

# Change to espyna directory
cd "$ESPYNA_DIR" || {
    echo -e "${RED}Error: Cannot change to espyna directory${NC}"
    exit 1
}

echo -e "${CYAN}=== Espyna Development Debug Build ===${NC}"
echo -e "${WHITE}Building developer-optimized server with:${NC}"
echo ""
echo -e "${CYAN}🛠️  Development Features:${NC}"
echo -e "${GREEN}  • HTTP Framework: Gin (hot-reload friendly)${NC}"
echo -e "${GREEN}  • Mock Providers: ALL included (offline development)${NC}"
echo -e "${GREEN}  • Business Types: ALL test data scenarios${NC}"
if [[ "$SYMBOL_TABLE" == true ]]; then
    echo -e "${GREEN}  • Debug Symbols: ENABLED${NC}"
else
    echo -e "${YELLOW}  • Debug Symbols: DISABLED${NC}"
fi
if [[ "$RACE" == true ]]; then
    echo -e "${GREEN}  • Race Detection: ENABLED${NC}"
else
    echo -e "${YELLOW}  • Race Detection: DISABLED${NC}"
fi
if [[ "$VERBOSE_BUILD" == true ]]; then
    echo -e "${GREEN}  • Verbose Logging: ENABLED${NC}"
else
    echo -e "${YELLOW}  • Verbose Logging: DISABLED${NC}"
fi
echo ""
echo -e "${CYAN}🧪 Testing Capabilities:${NC}"
echo -e "${GREEN}  • Multiple provider combinations for integration testing${NC}"
echo -e "${GREEN}  • Business type switching at runtime${NC}"
echo -e "${GREEN}  • Mock data for all 40+ business entities${NC}"
echo -e "${GREEN}  • API endpoint testing across all domains${NC}"
echo ""

# Development-optimized build tags - prioritize mock providers and debugging
BUILD_TAGS=(
    "gin" "providers_bootstrap"
    # Mock providers first (primary for development)
    "mock_db" "mock_email" "mock_storage"
    # Essential real providers for integration testing
    "postgres" "postgres_migrations" "local_storage" "jwt_auth"
    # Cloud providers for testing cloud integrations
    "firebase" "firestore" "google" "gmail" "gcp_storage"
    "microsoft" "microsoftgraph" "aws" "s3"
    # Fallback providers
    "noop"
)

echo -e "${GRAY}Working directory: $(pwd)${NC}"
echo -e "${BLUE}Build tags: $(IFS=','; echo "${BUILD_TAGS[*]}")${NC}"
echo -e "${GREEN}Development mode: ENABLED${NC}"
echo ""

# Use the quick-build approach with development-optimized tags
SECONDARY_TAGS_ARRAY=("${BUILD_TAGS[@]:1}")  # Remove "gin" from the array
SECONDARY_TAGS_STRING=$(IFS=','; echo "${SECONDARY_TAGS_ARRAY[*]}")

# Prepare build command with development-specific flags
BUILD_COMMAND="./scripts/build-with-tags.sh --framework gin --secondary-tags '$SECONDARY_TAGS_STRING' --output $OUTPUT"
if [[ "$VERBOSE_BUILD" == true ]]; then
    BUILD_COMMAND+=" --verbose"
fi
if [[ "$RACE" == true ]]; then
    BUILD_COMMAND+=" --race"
fi

echo -e "${MAGENTA}Executing: $BUILD_COMMAND${NC}"
echo ""

# Build arguments for the main build script
BUILD_ARGS=(
    --framework "gin"
    --secondary-tags "$SECONDARY_TAGS_STRING"
    --output "$OUTPUT"
)

if [[ "$VERBOSE_BUILD" == true ]]; then
    BUILD_ARGS+=(--verbose)
fi

if [[ "$RACE" == true ]]; then
    BUILD_ARGS+=(--race)
fi

if [[ "$SYMBOL_TABLE" == true ]]; then
    # Add debug symbols for better debugging experience
    BUILD_ARGS+=(--ldflags "-X main.buildMode=development")
fi

# Execute build using the working build script
if ./scripts/build-with-tags.sh "${BUILD_ARGS[@]}"; then
    echo ""
    echo -e "${GREEN}✅ Development debug build completed!${NC}"
    
    # Show binary info
    OUTPUT_PATH="build/$OUTPUT"
    if [[ -f "$OUTPUT_PATH" ]]; then
        echo -e "${GREEN}✅ Binary created: $OUTPUT_PATH${NC}"
        
        # Calculate binary size
        BINARY_SIZE_BYTES=$(stat -f%z "$OUTPUT_PATH" 2>/dev/null || stat -c%s "$OUTPUT_PATH" 2>/dev/null || echo "0")
        BINARY_SIZE_MB=$((BINARY_SIZE_BYTES / 1024 / 1024))
        
        # Use bc for precise calculations if available, otherwise use bash arithmetic
        if command -v bc >/dev/null 2>&1; then
            BINARY_SIZE_DISPLAY=$(echo "scale=2; $BINARY_SIZE_BYTES / 1024 / 1024" | bc)
            if [[ "$SYMBOL_TABLE" == true ]]; then
                echo -e "${BLUE}✅ Binary size: ${BINARY_SIZE_DISPLAY} MB (includes debug symbols)${NC}"
            else
                echo -e "${BLUE}✅ Binary size: ${BINARY_SIZE_DISPLAY} MB${NC}"
            fi
        else
            # Fallback to simple integer calculations
            if [[ "$SYMBOL_TABLE" == true ]]; then
                echo -e "${BLUE}✅ Binary size: ${BINARY_SIZE_MB} MB (includes debug symbols)${NC}"
            else
                echo -e "${BLUE}✅ Binary size: ${BINARY_SIZE_MB} MB${NC}"
            fi
        fi
    fi
    
    echo ""
    echo -e "${CYAN}🚀 Development Usage Examples:${NC}"
    echo ""
    echo -e "${WHITE}  Quick Start (Mock Data):${NC}"
    echo -e "${GRAY}    MOCK_MODE=true LOG_LEVEL=debug ./$OUTPUT_PATH${NC}"
    echo ""
    echo -e "${WHITE}  Test Different Business Types:${NC}"
    echo -e "${GRAY}    MOCK_MODE=true MOCK_BUSINESS_TYPE=education ./$OUTPUT_PATH${NC}"
    echo -e "${GRAY}    MOCK_MODE=true MOCK_BUSINESS_TYPE=fitness_center ./$OUTPUT_PATH${NC}"
    echo -e "${GRAY}    MOCK_MODE=true MOCK_BUSINESS_TYPE=office_leasing ./$OUTPUT_PATH${NC}"
    echo ""
    echo -e "${WHITE}  API Testing with Real Providers:${NC}"
    echo -e "${GRAY}    SERVER_TYPE=gin DATABASE_PROVIDER=postgres \\${NC}"
    echo -e "${GRAY}    DATABASE_URL=postgres://localhost:5432/dev_db ./$OUTPUT_PATH${NC}"
    echo ""
    echo -e "${WHITE}  Multi-Provider Integration Testing:${NC}"
    echo -e "${GRAY}    DATABASE_PROVIDER=postgres AUTH_PROVIDER=jwt \\${NC}"
    echo -e "${GRAY}    EMAIL_PROVIDER=mock STORAGE_PROVIDER=local ./$OUTPUT_PATH${NC}"
    echo ""
    echo -e "${WHITE}  Hot Development with File Watching:${NC}"
    echo -e "${GRAY}    # Use with air or similar file watcher${NC}"
    echo -e "${GRAY}    air -c .air.toml${NC}"
    echo ""
    echo -e "${CYAN}📋 Development Environment Variables:${NC}"
    echo -e "${WHITE}  # Development Settings${NC}"
    echo -e "${GRAY}  SERVER_TYPE=gin                      # Use Gin for development${NC}"
    echo -e "${GRAY}  SERVER_PORT=3000                     # Development port${NC}"
    echo -e "${GRAY}  LOG_LEVEL=debug                      # Verbose logging${NC}"
    echo -e "${GRAY}  GIN_MODE=debug                       # Gin debug mode${NC}"
    echo ""
    echo -e "${WHITE}  # Mock Data Configuration${NC}"
    echo -e "${GRAY}  MOCK_MODE=true                       # Enable mock providers${NC}"
    echo -e "${GRAY}  MOCK_BUSINESS_TYPE=education         # Business scenario${NC}"
    echo -e "${GRAY}  MOCK_USER_COUNT=50                   # Number of mock users${NC}"
    echo ""
    echo -e "${WHITE}  # Database Development${NC}"
    echo -e "${GRAY}  DATABASE_PROVIDER=mock               # For pure offline dev${NC}"
    echo -e "${GRAY}  DATABASE_URL=postgres://localhost:5432/testdb  # For DB integration testing${NC}"
    echo ""
    echo -e "${WHITE}  # API Testing${NC}"
    echo -e "${GRAY}  CORS_ENABLED=true                    # Enable CORS for frontend dev${NC}"
    echo -e "${GRAY}  API_TIMEOUT=30s                      # Extended timeouts for debugging${NC}"
    echo ""
    echo -e "${CYAN}🔧 Debugging Tips:${NC}"
    echo -e "${GRAY}  • Use LOG_LEVEL=debug for verbose output${NC}"
    echo -e "${GRAY}  • Mock providers allow offline development${NC}"
    echo -e "${GRAY}  • Switch business types to test different data scenarios${NC}"
    echo -e "${GRAY}  • Race detection helps catch concurrency issues early${NC}"
    echo -e "${GRAY}  • Use curl or Postman to test API endpoints${NC}"
    if [[ "$SYMBOL_TABLE" == true ]]; then
        echo -e "${GRAY}  • Debug symbols included for GDB/Delve debugging${NC}"
    fi
    
else
    echo ""
    echo -e "${RED}❌ Build failed!${NC}"
    exit $?
fi

echo ""
echo -e "${CYAN}🎯 Development Build Benefits:${NC}"
echo -e "${GRAY}   • Optimized for rapid development and testing cycles${NC}"
echo -e "${GRAY}   • Comprehensive mock providers for offline development${NC}"
echo -e "${GRAY}   • Multiple business type scenarios for thorough testing${NC}"
echo -e "${GRAY}   • Race condition detection prevents concurrency bugs${NC}"
echo -e "${GRAY}   • Enhanced logging for debugging complex issues${NC}"
echo -e "${GRAY}   • Provider switching allows integration testing scenarios${NC}"
echo -e "${GRAY}   • Hot-reload compatible with development tools${NC}"