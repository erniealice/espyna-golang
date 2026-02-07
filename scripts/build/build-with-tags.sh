#!/bin/bash
#
# Build Espyna server with specific framework and secondary adapter tags.
#
# This script builds the Espyna server using Go build tags to conditionally compile
# only the specified HTTP framework adapters (vanilla, gin, fiber) and secondary adapters.
# 
# The build tags are already implemented in the source files:
# - packages/espyna/internal/infrastructure/adapters/primary/http/vanilla/server.go (//go:build vanilla)
# - packages/espyna/internal/infrastructure/adapters/primary/http/gin/server.go (//go:build gin)  
# - packages/espyna/internal/infrastructure/adapters/primary/http/fiber/server.go (//go:build fiber)
# - Secondary adapters now also have build tags (e.g., //go:build firestore, //go:build google && gcp_storage).
#
# PARAMETERS:
#   -f, --framework FRAMEWORK    HTTP framework to build with. Valid values: vanilla, gin, fiber, all (default: vanilla)
#   -s, --secondary-tags TAGS    Comma-separated list of secondary adapter build tags (e.g., "firestore,google,aws")
#   -o, --output OUTPUT          Output binary name (default: espyna-server)
#   -v, --verbose               Enable verbose build output
#   -r, --race                  Enable race condition detection
#   -l, --ldflags LDFLAGS       Additional linker flags
#   -h, --help                  Show this help message
#
# EXAMPLES:
#   ./build-with-tags.sh -f fiber -s "firestore,google"
#       Builds server with Fiber framework and Firestore/Google Cloud adapters.
#
#   ./build-with-tags.sh -f gin -o espyna-gin -v -s "postgres,mock_email"
#       Builds Gin-only server with verbose output, PostgreSQL, and mock email adapters.
#
#   ./build-with-tags.sh -f all -r -s "firestore,aws,microsoft"
#       Builds server with all frameworks, race detection, and Firestore, AWS, and Microsoft adapters.
#
# NOTES:
#   Build tags reduce binary size and eliminate unused dependencies.
#   The main.go file will conditionally import only the tagged frameworks.
#

set -euo pipefail

# Default values
FRAMEWORK="vanilla"
SECONDARY_TAGS=()
OUTPUT="espyna-server"
VERBOSE_BUILD=false
RACE=false
LDFLAGS=""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
GRAY='\033[0;90m'
MAGENTA='\033[0;35m'
NC='\033[0m' # No Color

# Function to show usage
show_help() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Build Espyna server with specific framework and secondary adapter tags."
    echo ""
    echo "OPTIONS:"
    echo "  -f, --framework FRAMEWORK    HTTP framework to build with [vanilla|gin|fiber|all] (default: vanilla)"
    echo "  -s, --secondary-tags TAGS    Comma-separated secondary adapter build tags"
    echo "  -o, --output OUTPUT          Output binary name (default: espyna-server)"
    echo "  -v, --verbose               Enable verbose build output"
    echo "  -r, --race                  Enable race condition detection"
    echo "  -l, --ldflags LDFLAGS       Additional linker flags"
    echo "  -h, --help                  Show this help message"
    echo ""
    echo "EXAMPLES:"
    echo "  $0 -f fiber -s \"firestore,google\""
    echo "  $0 -f gin -o espyna-gin -v -s \"postgres,mock_email\""
    echo "  $0 -f all -r -s \"firestore,aws,microsoft\""
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -f|--framework)
            FRAMEWORK="$2"
            shift 2
            ;;
        -s|--secondary-tags)
            IFS=',' read -ra SECONDARY_TAGS <<< "$2"
            shift 2
            ;;
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
        -l|--ldflags)
            LDFLAGS="$2"
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

# Validate framework parameter
case $FRAMEWORK in
    vanilla|gin|fiber|all)
        ;;
    *)
        echo -e "${RED}Error: Invalid framework '$FRAMEWORK'. Valid values: vanilla, gin, fiber, all${NC}"
        exit 1
        ;;
esac

# Set build directory to packages/espyna
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ESPYNA_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"
cd "$ESPYNA_DIR"

echo -e "${CYAN}Building Espyna Server with framework: $FRAMEWORK${NC}"
echo -e "${GRAY}Working directory: $(pwd)${NC}"

# Prepare build command components
BUILD_ARGS=()

# Collect all tags
ALL_TAGS=()

# Add framework tags
if [[ "$FRAMEWORK" == "all" ]]; then
    ALL_TAGS+=(vanilla gin fiber)
    echo -e "${GREEN}Framework tags: vanilla,gin,fiber (all frameworks)${NC}"
else
    ALL_TAGS+=("$FRAMEWORK")
    echo -e "${GREEN}Framework tags: $FRAMEWORK${NC}"
fi

# Add secondary tags
if [[ ${#SECONDARY_TAGS[@]} -gt 0 ]]; then
    ALL_TAGS+=("${SECONDARY_TAGS[@]}")
    echo -e "${GREEN}Secondary tags: $(IFS=','; echo "${SECONDARY_TAGS[*]}")${NC}"
fi

# Join all tags with commas
if [[ ${#ALL_TAGS[@]} -gt 0 ]]; then
    TAGS_STRING=$(IFS=','; echo "${ALL_TAGS[*]}")
    BUILD_ARGS+=(-tags "$TAGS_STRING")
fi

# Add verbose flag if requested
if [[ "$VERBOSE_BUILD" == true ]]; then
    BUILD_ARGS+=(-v)
    echo -e "${YELLOW}Verbose output enabled${NC}"
fi

# Add race detection if requested
if [[ "$RACE" == true ]]; then
    BUILD_ARGS+=(-race)
    echo -e "${YELLOW}Race condition detection enabled${NC}"
fi

# Add linker flags if provided
if [[ -n "$LDFLAGS" ]]; then
    BUILD_ARGS+=(-ldflags "$LDFLAGS")
    echo -e "${YELLOW}Linker flags: $LDFLAGS${NC}"
else
    # Suggest optimization flags if none provided
    echo -e "${BLUE}Tip: Add -l '-s -w' for smaller binaries (strips debug symbols)${NC}"
fi

# Ensure build directory exists
BUILD_DIR="build"
if [[ ! -d "$BUILD_DIR" ]]; then
    mkdir -p "$BUILD_DIR"
    echo -e "${BLUE}Created build directory: $BUILD_DIR${NC}"
fi

# Set output path to build directory
OUTPUT_PATH="$BUILD_DIR/$OUTPUT"
BUILD_ARGS+=(-o "$OUTPUT_PATH")

# Add main package path
BUILD_ARGS+=(./cmd/server)

echo -e "${MAGENTA}Executing: go build ${BUILD_ARGS[*]}${NC}"
echo ""

# Execute the build command
if go build "${BUILD_ARGS[@]}"; then
    echo -e "${GREEN}Build completed successfully!${NC}"
    echo -e "${GREEN}Binary created: $OUTPUT_PATH${NC}"
    
    # Show enhanced binary info with size analysis
    if [[ -f "$OUTPUT_PATH" ]]; then
        BINARY_SIZE_BYTES=$(stat -f%z "$OUTPUT_PATH" 2>/dev/null || stat -c%s "$OUTPUT_PATH" 2>/dev/null || echo "0")
        BINARY_SIZE_MB=$((BINARY_SIZE_BYTES / 1024 / 1024))
        
        echo -e "${BLUE}Binary size: ${BINARY_SIZE_MB} MB${NC}"
        
        # Size category analysis
        if [[ $BINARY_SIZE_MB -lt 15 ]]; then
            echo -e "${GREEN}Size category: Optimized (< 15MB)${NC}"
        elif [[ $BINARY_SIZE_MB -lt 35 ]]; then
            echo -e "${YELLOW}Size category: Moderate (15-35MB)${NC}"
        elif [[ $BINARY_SIZE_MB -lt 60 ]]; then
            echo -e "${MAGENTA}Size category: Large (35-60MB)${NC}"
        else
            echo -e "${RED}Size category: Very Large (> 60MB)${NC}"
            echo -e "${YELLOW}💡 Consider using targeted builds (build-minimal-api.ps1 or build-cloud-native.ps1) for smaller size${NC}"
        fi
        
        # Provider count estimation
        ESTIMATED_PROVIDERS=0
        for tag in "${ALL_TAGS[@]}"; do
            if [[ $tag =~ ^(postgres|firestore|firebase|google|aws|azure|microsoft|mock_) ]]; then
                ((ESTIMATED_PROVIDERS++))
            fi
        done
        
        if [[ $ESTIMATED_PROVIDERS -gt 0 ]]; then
            echo -e "${GRAY}Active providers: ~$ESTIMATED_PROVIDERS (build tags: $(IFS=','; echo "${ALL_TAGS[*]}"))${NC}"
        fi
        
        # Size optimization suggestions
        if [[ $BINARY_SIZE_MB -gt 50 && -z "$LDFLAGS" ]]; then
            echo ""
            echo -e "${CYAN}Size optimization suggestions:${NC}"
            echo -e "${YELLOW}   • Add -l \"-s -w\" to strip debug symbols (~10-15% reduction)${NC}"
            echo -e "${YELLOW}   • Use build-minimal-api.ps1 for essential features only${NC}"
            echo -e "${YELLOW}   • Use build-cloud-native.ps1 -CloudProvider [gcp|aws|azure] for single-cloud builds${NC}"
        fi
    fi
    
    echo ""
    echo -e "${CYAN}Usage examples:${NC}"
    case $FRAMEWORK in
        vanilla)
            echo -e "${NC}   ./$OUTPUT_PATH${NC}"
            echo -e "${NC}   SERVER_PORT=8080 ./$OUTPUT_PATH${NC}"
            ;;
        gin)
            echo -e "${NC}   ./$OUTPUT_PATH${NC}"
            echo -e "${NC}   SERVER_TYPE=gin SERVER_PORT=8081 ./$OUTPUT_PATH${NC}"
            ;;
        fiber)
            echo -e "${NC}   ./$OUTPUT_PATH${NC}"
            echo -e "${NC}   SERVER_TYPE=fiber SERVER_PORT=8082 ./$OUTPUT_PATH${NC}"
            ;;
        all)
            echo -e "${NC}   SERVER_TYPE=vanilla ./$OUTPUT_PATH${NC}"
            echo -e "${NC}   SERVER_TYPE=gin ./$OUTPUT_PATH${NC}"
            echo -e "${NC}   SERVER_TYPE=fiber ./$OUTPUT_PATH${NC}"
            echo -e "${NC}   SERVER_TYPE=multi ./$OUTPUT_PATH${NC}"
            ;;
    esac
    
else
    echo -e "${RED}Build failed${NC}"
    exit 1
fi

echo ""
echo -e "${CYAN}Build Tag Benefits:${NC}"
echo -e "${GRAY}   • Smaller binary size (excludes unused frameworks)${NC}"
echo -e "${GRAY}   • Faster compilation (fewer dependencies to build)${NC}"
echo -e "${GRAY}   • Reduced memory footprint at runtime${NC}"
echo -e "${GRAY}   • Eliminates unused framework dependencies${NC}"