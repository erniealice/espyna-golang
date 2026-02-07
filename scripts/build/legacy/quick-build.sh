#!/bin/bash
#
# Quick build script for Espyna server with working build tag combinations
#
# This script provides pre-tested build tag combinations that are known to work.
# It uses the comprehensive tag set discovered through testing to ensure successful builds.
#
# PARAMETERS:
#   -f, --framework FRAMEWORK    The HTTP framework to build (default: vanilla)
#                                Valid values: vanilla, gin, fiber
#   -o, --output OUTPUT          Custom output binary name (optional)
#   -h, --help                  Show this help message
#
# EXAMPLES:
#   ./quick-build.sh -f fiber
#       Builds Fiber server with all providers
#
#   ./quick-build.sh -f gin -o my-gin-server
#       Builds Gin server with custom name

set -euo pipefail

# Default values
FRAMEWORK="vanilla"
OUTPUT=""

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
    echo "Quick build script for Espyna server with working build tag combinations."
    echo ""
    echo "OPTIONS:"
    echo "  -f, --framework FRAMEWORK    HTTP framework to build [vanilla|gin|fiber] (default: vanilla)"
    echo "  -o, --output OUTPUT          Custom output binary name (optional)"
    echo "  -h, --help                  Show this help message"
    echo ""
    echo "EXAMPLES:"
    echo "  $0 -f fiber                 # Builds Fiber server with all providers"
    echo "  $0 -f gin -o my-gin-server  # Builds Gin server with custom name"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -f|--framework)
            FRAMEWORK="$2"
            shift 2
            ;;
        -o|--output)
            OUTPUT="$2"
            shift 2
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

# Validate framework parameter
case $FRAMEWORK in
    vanilla|gin|fiber)
        ;;
    *)
        echo -e "${RED}Error: Invalid framework '$FRAMEWORK'. Valid values: vanilla, gin, fiber${NC}"
        exit 1
        ;;
esac

# Set build directory to packages/espyna
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ESPYNA_DIR="$(dirname "$(dirname "$(dirname "$SCRIPT_DIR")")")"
cd "$ESPYNA_DIR"

echo ""
echo -e "${CYAN}🚀 Espyna Quick Build${NC}"
echo -e "${GREEN}Framework: $FRAMEWORK${NC}"
echo -e "${GRAY}Working directory: $(pwd)${NC}"
echo ""

# Comprehensive working tag set (tested and verified)
WORKING_TAGS=(
    "$FRAMEWORK"
    "providers_bootstrap"
    "mock_db" "mock_email" "mock_storage"
    "local_storage" "google" "aws" "s3" 
    "microsoft" "microsoftgraph" "gmail" "gcp_storage"
    "noop" "postgres" "firestore" "firebase" 
    "postgres_migrations"
)

# Set default output name with tags if not provided
if [[ -z "$OUTPUT" ]]; then
    # Create a descriptive filename based on key tags
    KEY_TAGS=("$FRAMEWORK")
    
    # Add major provider types to filename
    for tag in "${WORKING_TAGS[@]}"; do
        case $tag in
            firestore) KEY_TAGS+=("firestore") ;;
            postgres) KEY_TAGS+=("postgres") ;;
            firebase) KEY_TAGS+=("firebase") ;;
            microsoft) KEY_TAGS+=("microsoft") ;;
            google) KEY_TAGS+=("google") ;;
            aws) KEY_TAGS+=("aws") ;;
            mock_db) KEY_TAGS+=("mock") ;;
        esac
    done
    
    OUTPUT="espyna-$(IFS='-'; echo "${KEY_TAGS[*]}")-tags"
    echo -e "${BLUE}Generated filename: $OUTPUT${NC}"
fi

echo -e "${BLUE}Build tags: $(IFS=','; echo "${WORKING_TAGS[*]}")${NC}"
echo ""

# Use the original working build script with our tested tags
BUILD_SCRIPT_PATH="./scripts/build-with-tags.sh"
SECONDARY_TAGS=($(printf '%s\n' "${WORKING_TAGS[@]}" | grep -v "^$FRAMEWORK$"))

echo -e "${MAGENTA}Executing: $BUILD_SCRIPT_PATH -f $FRAMEWORK -s '$(IFS=','; echo "${SECONDARY_TAGS[*]}")' -o $OUTPUT${NC}"
echo ""

# Execute the build
if "$BUILD_SCRIPT_PATH" -f "$FRAMEWORK" -s "$(IFS=','; echo "${SECONDARY_TAGS[*]}")" -o "$OUTPUT"; then
    echo ""
    echo -e "${GREEN}✅ Quick build completed!${NC}"
    echo ""
    
    # Show binary location
    BINARY_PATH="build/$OUTPUT"
    if [[ -f "$BINARY_PATH" ]]; then
        echo -e "${BLUE}📁 Binary location: $BINARY_PATH${NC}"
        BINARY_SIZE_BYTES=$(stat -f%z "$BINARY_PATH" 2>/dev/null || stat -c%s "$BINARY_PATH" 2>/dev/null || echo "0")
        BINARY_SIZE_MB=$((BINARY_SIZE_BYTES / 1024 / 1024))
        echo -e "${BLUE}📏 Binary size: ${BINARY_SIZE_MB} MB${NC}"
    fi
    
    echo ""
    echo -e "${CYAN}🧪 Test your build:${NC}"
    case $FRAMEWORK in
        vanilla)
            echo -e "${WHITE}   ./$BINARY_PATH${NC}"
            echo -e "${GRAY}   MOCK_MODE=true ./$BINARY_PATH${NC}"
            ;;
        gin)
            echo -e "${WHITE}   SERVER_TYPE=gin ./$BINARY_PATH${NC}"
            echo -e "${GRAY}   SERVER_TYPE=gin MOCK_MODE=true ./$BINARY_PATH${NC}"
            ;;
        fiber)
            echo -e "${WHITE}   SERVER_TYPE=fiber ./$BINARY_PATH${NC}"
            echo -e "${GRAY}   SERVER_TYPE=fiber MOCK_MODE=true ./$BINARY_PATH${NC}"
            ;;
    esac
    
else
    echo ""
    echo -e "${RED}❌ Build failed!${NC}"
    exit 1
fi

echo ""
echo -e "${CYAN}💡 Quick Build Benefits:${NC}"
echo -e "${GRAY}   • Uses pre-tested working build tag combinations${NC}"
echo -e "${GRAY}   • Includes all essential providers for maximum functionality${NC}"
echo -e "${GRAY}   • Reliable builds without tag dependency issues${NC}"
echo -e "${GRAY}   • Mock providers included for development and testing${NC}"