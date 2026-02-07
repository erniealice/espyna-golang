#!/bin/bash
#
# Build ultra-lightweight Espyna server for development with mock providers only
#
# This script creates the smallest possible build with:
# - Vanilla HTTP framework (Go standard library)
# - Mock database provider (in-memory, no external database required)
# - Mock authentication (test tokens, no auth service required)
# - Mock email provider (console logging, no email service required)
# - Mock storage provider (temporary files, no storage service required)
# - All business type mock data included (education, fitness_center, office_leasing)
# - Zero external dependencies for complete offline development
#
# PARAMETERS:
#   -o, --output OUTPUT         Output binary name (default: espyna-development)
#   -v, --verbose              Enable verbose build output
#   -r, --race                 Enable race condition detection
#   -b, --all-business-types   Include mock data for all business types (default: true)
#   -h, --help                 Show this help message
#
# EXAMPLES:
#   ./build-development.sh
#       Ultra-minimal development server with all mock providers
#
#   ./build-development.sh -v -r
#       Development build with debugging options
#
# NOTES:
#   This build configuration is optimized for:
#   - Ultra-small binary size (target: 5-15MB)
#   - Instant startup with no external dependencies
#   - Complete offline development capability
#   - Comprehensive test data for all business scenarios
#   - Hot-reload friendly development workflow
#   - CI/CD pipeline testing without external services

set -euo pipefail

# Default values
OUTPUT="espyna-development"
VERBOSE_BUILD=false
RACE=false
ALL_BUSINESS_TYPES=true

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
    echo "Build ultra-lightweight Espyna server for development with mock providers only"
    echo ""
    echo "OPTIONS:"
    echo "  -o, --output OUTPUT         Output binary name (default: espyna-development)"
    echo "  -v, --verbose              Enable verbose build output"
    echo "  -r, --race                 Enable race condition detection"
    echo "  -b, --all-business-types   Include mock data for all business types (default: true)"
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
        -b|--all-business-types)
            ALL_BUSINESS_TYPES=true
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

echo -e "${MAGENTA}=== Espyna Development Build ===${NC}"
echo -e "${WHITE}Building ultra-lightweight server for development:${NC}"
echo -e "${CYAN}  • Zero external dependencies (100% offline capable)${NC}"
echo -e "${CYAN}  • Ultra-small binary size (target: 5-15MB)${NC}"
echo -e "${CYAN}  • Instant startup with comprehensive mock data${NC}"
echo -e "${CYAN}  • Perfect for development, testing, and CI/CD${NC}"
echo ""
echo -e "${CYAN}🌐 HTTP Framework:${NC}"
echo -e "${GREEN}  • Vanilla (Go standard library - zero dependencies)${NC}"
echo ""
echo -e "${CYAN}🧪 Mock Providers (Development Only):${NC}"
echo -e "${YELLOW}  • Mock Database (in-memory with full business logic)${NC}"
echo -e "${YELLOW}  • Mock Authentication (test tokens and user sessions)${NC}"
echo -e "${YELLOW}  • Mock Email (console logging with realistic templates)${NC}"
echo -e "${YELLOW}  • Mock Storage (temporary files with full API support)${NC}"
echo ""

if [[ "$ALL_BUSINESS_TYPES" == true ]]; then
    echo -e "${CYAN}Business Type Support:${NC}"
    echo -e "${GREEN}  • Education (students, teachers, courses, grades)${NC}"
    echo -e "${GREEN}  • Fitness Center (members, trainers, classes, equipment)${NC}"
    echo -e "${GREEN}  • Office Leasing (tenants, properties, leases, maintenance)${NC}"
    echo ""
fi

# Development build tags - only mock providers
BUILD_TAGS=(
    "vanilla" "providers_bootstrap"
    # Mock providers only - no real external dependencies
    "mock_db" "mock_email" "mock_storage" "mock_auth"
    # Essential system components
    "noop"
)

echo -e "${GRAY}Working directory: $(pwd)${NC}"
echo -e "${MAGENTA}Build tags: $(IFS=','; echo "${BUILD_TAGS[*]}")${NC}"
echo -e "${MAGENTA}Total components: ${#BUILD_TAGS[@]} (minimal)${NC}"
echo ""

# Build with development-only tag set
SECONDARY_TAGS_STRING=$(IFS=','; echo "${BUILD_TAGS[@]:1}")

echo -e "${MAGENTA}Executing development build...${NC}"
echo -e "${GRAY}Command: ./scripts/build-with-tags.sh -f vanilla -s \"$SECONDARY_TAGS_STRING\" -o \"$OUTPUT\"${NC}"
echo ""

# Build arguments with development optimizations
BUILD_ARGS=(
    -f vanilla
    -s "$SECONDARY_TAGS_STRING"
    -o "$OUTPUT"
    -l "-s -w"  # Strip for minimal size
)

if [[ "$VERBOSE_BUILD" == true ]]; then
    BUILD_ARGS+=(-v)
fi

if [[ "$RACE" == true ]]; then
    BUILD_ARGS+=(-r)
    echo -e "${YELLOW}Race detection enabled - great for development debugging!${NC}"
fi

# Execute development build
if ./scripts/build-with-tags.sh "${BUILD_ARGS[@]}"; then
    echo ""
    echo -e "${GREEN}Development build finished!${NC}"
    
    # Show binary info with size comparison
    OUTPUT_PATH="build/$OUTPUT"
    if [[ -f "$OUTPUT_PATH" ]]; then
        echo -e "${GREEN}Binary created: $OUTPUT_PATH${NC}"
        
        # Get binary size in MB (cross-platform)
        if command -v stat >/dev/null 2>&1; then
            BINARY_SIZE_BYTES=$(stat -f%z "$OUTPUT_PATH" 2>/dev/null || stat -c%s "$OUTPUT_PATH" 2>/dev/null)
            BINARY_SIZE_MB=$((BINARY_SIZE_BYTES / 1024 / 1024))
            if [[ $BINARY_SIZE_MB -eq 0 ]]; then
                BINARY_SIZE_MB=$(echo "scale=2; $BINARY_SIZE_BYTES / 1024 / 1024" | bc -l 2>/dev/null || echo "1")
            fi
        else
            BINARY_SIZE_MB="~10"
        fi
        
        echo -e "${MAGENTA}Binary size: ${BINARY_SIZE_MB} MB (development-optimized)${NC}"
        
        # Size comparison with enterprise build
        COMPARISON_SIZE=70  # Typical enterprise build size
        if command -v bc >/dev/null 2>&1 && [[ "$BINARY_SIZE_MB" != *"~"* ]]; then
            SIZE_REDUCTION=$(echo "scale=1; ($COMPARISON_SIZE - $BINARY_SIZE_MB) / $COMPARISON_SIZE * 100" | bc -l)
            echo -e "${GREEN}Size reduction: ${SIZE_REDUCTION}% smaller than enterprise builds${NC}"
            
            # Compare with minimal API build
            MINIMAL_SIZE=15  # Expected minimal API build size
            if (( $(echo "$BINARY_SIZE_MB < $MINIMAL_SIZE" | bc -l) )); then
                DEV_ADVANTAGE=$(echo "scale=1; ($MINIMAL_SIZE - $BINARY_SIZE_MB) / $MINIMAL_SIZE * 100" | bc -l)
                echo -e "${CYAN}Even smaller: ${DEV_ADVANTAGE}% smaller than minimal API builds${NC}"
            fi
        fi
    fi
    
    echo ""
    echo -e "${MAGENTA}🚀 Development Usage Examples:${NC}"
    echo ""
    echo -e "${WHITE}  Instant Development Server:${NC}"
    echo -e "${GRAY}    ./$OUTPUT${NC}"
    echo -e "${GREEN}    # No configuration needed - runs immediately with mock data${NC}"
    echo ""
    echo -e "${WHITE}  Business Type Testing:${NC}"
    echo -e "${GRAY}    MOCK_BUSINESS_TYPE=education ./$OUTPUT    # Student/teacher data${NC}"
    echo -e "${GRAY}    MOCK_BUSINESS_TYPE=fitness_center ./$OUTPUT  # Gym/trainer data${NC}"
    echo -e "${GRAY}    MOCK_BUSINESS_TYPE=office_leasing ./$OUTPUT  # Tenant/property data${NC}"
    echo ""
    echo -e "${WHITE}  Development with Debugging:${NC}"
    echo -e "${GRAY}    LOG_LEVEL=debug \\${NC}"
    echo -e "${GRAY}    MOCK_MODE=true \\${NC}"
    echo -e "${GRAY}    MOCK_BUSINESS_TYPE=education \\${NC}"
    echo -e "${GRAY}    ./$OUTPUT${NC}"
    echo ""
    echo -e "${WHITE}  CI/CD Pipeline Testing:${NC}"
    echo -e "${GREEN}    # Perfect for automated testing - no external dependencies${NC}"
    echo -e "${GRAY}    docker run --rm -p 8080:8080 espyna-dev:latest${NC}"
    echo ""
    echo -e "${WHITE}  API Development and Testing:${NC}"
    echo -e "${GREEN}    # All 40 entities with realistic mock data${NC}"
    echo -e "${GRAY}    curl http://localhost:8080/api/entities/client${NC}"
    echo -e "${GRAY}    curl http://localhost:8080/api/entities/subscription${NC}"
    echo -e "${GRAY}    curl http://localhost:8080/api/entities/event${NC}"
    echo ""
    echo -e "${CYAN}📋 Environment Variables (All Optional):${NC}"
    echo -e "${WHITE}  # Core Configuration${NC}"
    echo -e "${GRAY}  SERVER_TYPE=vanilla  # Always vanilla for development${NC}"
    echo -e "${GRAY}  SERVER_PORT=8080     # HTTP port (default: 8080)${NC}"
    echo -e "${GRAY}  LOG_LEVEL=debug      # Verbose logging for development${NC}"
    echo ""
    echo -e "${WHITE}  # Business Type Selection${NC}"
    echo -e "${GRAY}  MOCK_BUSINESS_TYPE=education      # Default business scenario${NC}"
    echo -e "${GRAY}  MOCK_BUSINESS_TYPE=fitness_center # Alternative scenario${NC}"
    echo -e "${GRAY}  MOCK_BUSINESS_TYPE=office_leasing # Alternative scenario${NC}"
    echo ""
    echo -e "${WHITE}  # Development Features${NC}"
    echo -e "${GRAY}  MOCK_MODE=true              # Always true for this build${NC}"
    echo -e "${GRAY}  MOCK_DELAY_MS=100           # Add realistic API delays${NC}"
    echo -e "${GRAY}  MOCK_ERROR_RATE=0.05        # Simulate 5% error rate for testing${NC}"
    
else
    echo ""
    echo -e "${RED}Build failed!${NC}"
    exit 1
fi

echo ""
echo -e "${CYAN}Development Build Benefits:${NC}"
echo -e "${GRAY}   • Zero external dependencies - runs anywhere instantly${NC}"
echo -e "${GRAY}   • Ultra-fast startup - perfect for development iteration${NC}"
echo -e "${GRAY}   • Comprehensive mock data - all 40 entities with realistic content${NC}"
echo -e "${GRAY}   • Complete business scenarios - education, fitness, office leasing${NC}"
echo -e "${GRAY}   • CI/CD friendly - no database setup or external services needed${NC}"
echo -e "${GRAY}   • Hot-reload ready - minimal resource usage for development loops${NC}"
echo -e "${GRAY}   • Perfect for onboarding - new developers can start immediately${NC}"
echo ""
echo -e "${CYAN}Perfect for:${NC}"
echo -e "${GRAY}   • Local development and prototyping${NC}"
echo -e "${GRAY}   • API development and testing${NC}"
echo -e "${GRAY}   • CI/CD pipeline testing${NC}"
echo -e "${GRAY}   • Frontend development with realistic backend${NC}"
echo -e "${GRAY}   • Demo and presentation environments${NC}"
echo -e "${GRAY}   • New developer onboarding${NC}"
echo -e "${GRAY}   • Offline development environments${NC}"
echo ""
echo -e "${GREEN}🏃‍♂️ Quick Start:${NC}"
echo -e "${WHITE}   1. Run: ./$OUTPUT${NC}"
echo -e "${WHITE}   2. Open: http://localhost:8080/health${NC}"
echo -e "${WHITE}   3. Test API: http://localhost:8080/api/entities/client${NC}"
echo -e "${WHITE}   4. That's it! No setup required.${NC}"