#!/bin/bash
#
# Build ultra-lightweight Espyna server for edge computing and resource-constrained environments
#
# This script creates the smallest possible build with minimal dependencies:
# - Vanilla HTTP framework (Go standard library only)
# - Mock providers for ultra-light deployment
# - Local file storage (no cloud dependencies)
# - JWT authentication (stateless, no external auth services)
# - Optimized for IoT, edge computing, and embedded systems
#
# PARAMETERS:
#   -o, --output OUTPUT          Output binary name (default: espyna-minimal-edge)
#   -v, --verbose               Enable verbose build output
#   -r, --race                  Enable race condition detection
#   -p, --include-postgres      Include PostgreSQL support for local database (default: false)
#   -h, --help                  Show this help message
#
# EXAMPLES:
#   ./build-minimal-edge.sh
#       Ultra-minimal build for edge deployment
#
#   ./build-minimal-edge.sh -p
#       Minimal build with optional PostgreSQL support
#
# NOTES:
#   This build configuration is optimized for:
#   - IoT and edge computing devices
#   - Raspberry Pi and ARM-based systems
#   - Docker containers with minimal base images
#   - Development environments with limited resources
#   - Air-gapped or offline deployments

set -euo pipefail

# Default values
OUTPUT="espyna-minimal-edge"
VERBOSE_BUILD=false
RACE=false
INCLUDE_POSTGRES=false

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
    echo "Build ultra-lightweight Espyna server for edge computing and resource-constrained environments."
    echo ""
    echo "OPTIONS:"
    echo "  -o, --output OUTPUT          Output binary name (default: espyna-minimal-edge)"
    echo "  -v, --verbose               Enable verbose build output"
    echo "  -r, --race                  Enable race condition detection"
    echo "  -p, --include-postgres      Include PostgreSQL support for local database (default: false)"
    echo "  -h, --help                  Show this help message"
    echo ""
    echo "EXAMPLES:"
    echo "  $0                          # Ultra-minimal build for edge deployment"
    echo "  $0 -p                       # Minimal build with optional PostgreSQL support"
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
        -p|--include-postgres)
            INCLUDE_POSTGRES=true
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

echo -e "${CYAN}=== Espyna Minimal Edge Build ===${NC}"
echo -e "${WHITE}Building ultra-lightweight server for edge computing:${NC}"
echo -e "  ${GREEN}• HTTP Framework: Vanilla (Go standard library)${NC}"
echo -e "  ${GREEN}• Database: Mock in-memory (ultra-fast startup)${NC}"
echo -e "  ${GREEN}• Authentication: JWT (no external dependencies)${NC}"
echo -e "  ${GREEN}• Storage: Local filesystem (no cloud services)${NC}"
echo -e "  ${GREEN}• Email: Mock console logging (no SMTP)${NC}"
echo -e "  ${GREEN}• Memory: Minimal footprint design${NC}"

if [[ "$INCLUDE_POSTGRES" == true ]]; then
    echo -e "  ${YELLOW}• Optional: PostgreSQL support included${NC}"
fi
echo ""

# Minimal build tags - only essential components
BUILD_TAGS=(
    "vanilla" "providers_bootstrap" "mock_db" "mock_email" "mock_storage" 
    "local_storage" "jwt_auth" "noop"
)

if [[ "$INCLUDE_POSTGRES" == true ]]; then
    BUILD_TAGS+=("postgres" "postgres_migrations")
    echo -e "${BLUE}📊 PostgreSQL support: ENABLED${NC}"
else
    echo -e "${BLUE}📊 PostgreSQL support: DISABLED (ultra-minimal mode)${NC}"
fi

# Note: Deliberately exclude cloud providers to minimize binary size
echo -e "${BLUE}☁️  Cloud providers: DISABLED (minimal build)${NC}"
echo -e "${BLUE}📦 Dependencies: MINIMAL (edge computing optimized)${NC}"
echo ""

echo -e "${GRAY}Working directory: $(pwd)${NC}"
echo -e "${BLUE}Build tags: $(IFS=','; echo "${BUILD_TAGS[*]}")${NC}"
echo ""

# Use the build-with-tags script with minimal tag set
SECONDARY_TAGS=($(printf '%s\n' "${BUILD_TAGS[@]}" | grep -v "vanilla"))

echo -e "${MAGENTA}Executing: ./scripts/build-with-tags.sh -f vanilla -s '$(IFS=','; echo "${SECONDARY_TAGS[*]}")' -o $OUTPUT${NC}"
echo ""

# Build arguments for the main build script
BUILD_ARGS=(
    "-f" "vanilla"
    "-s" "$(IFS=','; echo "${SECONDARY_TAGS[*]}")"
    "-o" "$OUTPUT"
)

if [[ "$VERBOSE_BUILD" == true ]]; then
    BUILD_ARGS+=("-v")
fi

if [[ "$RACE" == true ]]; then
    BUILD_ARGS+=("-r")
fi

# Execute build using the working build script
if ./scripts/build-with-tags.sh "${BUILD_ARGS[@]}"; then
    echo ""
    echo -e "${GREEN}✅ Minimal edge build completed!${NC}"
    
    # Show binary info with size comparison
    OUTPUT_PATH="build/$OUTPUT"
    if [[ -f "$OUTPUT_PATH" ]]; then
        echo -e "${GREEN}✅ Binary created: $OUTPUT_PATH${NC}"
        BINARY_SIZE_BYTES=$(stat -f%z "$OUTPUT_PATH" 2>/dev/null || stat -c%s "$OUTPUT_PATH" 2>/dev/null || echo "0")
        BINARY_SIZE_MB=$((BINARY_SIZE_BYTES / 1024 / 1024))
        echo -e "${BLUE}✅ Binary size: ${BINARY_SIZE_MB} MB${NC}"
        
        # Size comparison with other builds
        FULL_BUILD_PATH="build/espyna-server"
        if [[ -f "$FULL_BUILD_PATH" ]]; then
            FULL_SIZE_BYTES=$(stat -f%z "$FULL_BUILD_PATH" 2>/dev/null || stat -c%s "$FULL_BUILD_PATH" 2>/dev/null || echo "0")
            FULL_SIZE_MB=$((FULL_SIZE_BYTES / 1024 / 1024))
            if [[ $FULL_SIZE_MB -gt 0 ]]; then
                SIZE_REDUCTION=$(( (FULL_SIZE_MB - BINARY_SIZE_MB) * 100 / FULL_SIZE_MB ))
                echo -e "${GREEN}📉 Size reduction: ${SIZE_REDUCTION}% smaller than full build${NC}"
            fi
        fi
    fi
    
    echo ""
    echo -e "${CYAN}🚀 Edge Deployment Examples:${NC}"
    echo ""
    echo -e "${WHITE}  Raspberry Pi Deployment:${NC}"
    echo -e "${GRAY}    # Copy binary to Pi${NC}"
    echo -e "${GRAY}    scp $OUTPUT_PATH pi@raspberrypi.local:/home/pi/${NC}"
    echo -e "${GRAY}    # Run on Pi${NC}"
    echo -e "${GRAY}    ssh pi@raspberrypi.local${NC}"
    echo -e "${GRAY}    chmod +x $OUTPUT && ./$OUTPUT${NC}"
    echo ""
    echo -e "${WHITE}  Docker Alpine Container:${NC}"
    echo -e "${GRAY}    # Dockerfile${NC}"
    echo -e "${GRAY}    FROM alpine:latest${NC}"
    echo -e "${GRAY}    COPY $OUTPUT /usr/local/bin/${NC}"
    echo -e "${GRAY}    ENTRYPOINT [\"/usr/local/bin/$OUTPUT\"]${NC}"
    echo ""
    echo -e "${WHITE}  IoT Edge Device:${NC}"
    echo -e "${GRAY}    PORT=3000 JWT_SECRET=edge-device-secret ./$OUTPUT_PATH${NC}"
    echo ""
    echo -e "${WHITE}  Local Development:${NC}"
    echo -e "${GRAY}    MOCK_MODE=true MOCK_BUSINESS_TYPE=fitness_center ./$OUTPUT_PATH${NC}"
    echo ""
    echo -e "${CYAN}📋 Environment Variables:${NC}"
    echo -e "${GRAY}  SERVER_PORT=8080                     # Server port${NC}"
    echo -e "${GRAY}  JWT_SECRET=your-edge-secret          # JWT signing key${NC}"
    echo -e "${GRAY}  STORAGE_PATH=/data/uploads           # Local file storage path${NC}"
    echo -e "${GRAY}  LOG_LEVEL=info                       # Logging level (debug|info|warn|error)${NC}"
    echo -e "${GRAY}  MOCK_MODE=true                       # Use in-memory mock data${NC}"
    echo -e "${GRAY}  MOCK_BUSINESS_TYPE=education         # Mock data type${NC}"
    
    if [[ "$INCLUDE_POSTGRES" == true ]]; then
        echo -e "${GRAY}  DATABASE_URL=postgres://...         # PostgreSQL connection (if enabled)${NC}"
    fi
        
else
    echo ""
    echo -e "${RED}❌ Build failed!${NC}"
    exit 1
fi

echo ""
echo -e "${CYAN}⚡ Minimal Edge Stack Benefits:${NC}"
echo -e "${GRAY}   • Ultra-small binary size for resource-constrained devices${NC}"
echo -e "${GRAY}   • Zero cloud dependencies - runs completely offline${NC}"
echo -e "${GRAY}   • Instant startup with in-memory mock database${NC}"
echo -e "${GRAY}   • Perfect for IoT, edge computing, and embedded systems${NC}"
echo -e "${GRAY}   • Docker-optimized for minimal container images${NC}"
echo -e "${GRAY}   • ARM-compatible for Raspberry Pi and similar devices${NC}"
echo -e "${GRAY}   • Self-contained - no external service dependencies${NC}"