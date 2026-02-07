#!/bin/bash
#
# Display all Espyna builds with their sizes and capabilities

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
GRAY='\033[0;90m'
WHITE='\033[1;37m'
NC='\033[0m' # No Color

# Set build directory to packages/espyna
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ESPYNA_DIR="$(dirname "$(dirname "$(dirname "$SCRIPT_DIR")")")"
cd "$ESPYNA_DIR"

echo ""
echo -e "${CYAN}🏭 Espyna Build Summary${NC}"
echo "================================================================================"
echo ""

if [[ -d "build" ]]; then
    # Get all builds sorted by name
    BUILD_FILES=($(find build -maxdepth 1 -type f -executable | sort))
    
    if [[ ${#BUILD_FILES[@]} -eq 0 ]]; then
        echo -e "${YELLOW}No executable builds found in build directory.${NC}"
        echo ""
        exit 0
    fi
    
    echo -e "${WHITE}📦 Legacy Short Names:${NC}"
    for build_file in "${BUILD_FILES[@]}"; do
        build_name=$(basename "$build_file")
        if [[ ! "$build_name" == *"tags"* ]]; then
            if [[ -f "$build_file" ]]; then
                size_bytes=$(stat -f%z "$build_file" 2>/dev/null || stat -c%s "$build_file" 2>/dev/null || echo "0")
                size_mb=$((size_bytes / 1024 / 1024))
                printf "   %-25s %6s MB\n" "$build_name" "$size_mb"
            fi
        fi
    done
    
    echo ""
    echo -e "${WHITE}🏷️  Descriptive Tag-based Names:${NC}"
    for build_file in "${BUILD_FILES[@]}"; do
        build_name=$(basename "$build_file")
        if [[ "$build_name" == *"tags"* ]]; then
            if [[ -f "$build_file" ]]; then
                size_bytes=$(stat -f%z "$build_file" 2>/dev/null || stat -c%s "$build_file" 2>/dev/null || echo "0")
                size_mb=$((size_bytes / 1024 / 1024))
                printf "${GREEN}   %-70s %6s MB${NC}\n" "$build_name" "$size_mb"
            fi
        fi
    done
    
    echo ""
    echo -e "${CYAN}📊 Build Statistics:${NC}"
    total_size=0
    build_count=${#BUILD_FILES[@]}
    
    for build_file in "${BUILD_FILES[@]}"; do
        if [[ -f "$build_file" ]]; then
            size_bytes=$(stat -f%z "$build_file" 2>/dev/null || stat -c%s "$build_file" 2>/dev/null || echo "0")
            size_mb=$((size_bytes / 1024 / 1024))
            total_size=$((total_size + size_mb))
        fi
    done
    
    if [[ $build_count -gt 0 ]]; then
        average_size=$((total_size / build_count))
        echo -e "${WHITE}   Total builds: $build_count${NC}"
        echo -e "${WHITE}   Total size: ${total_size} MB${NC}"
        echo -e "${WHITE}   Average size: ${average_size} MB${NC}"
    fi
    
    echo ""
    echo -e "${CYAN}🔍 Tag Analysis:${NC}"
    
    # Count tag builds
    tag_build_count=0
    for build_file in "${BUILD_FILES[@]}"; do
        build_name=$(basename "$build_file")
        if [[ "$build_name" == *"tags"* ]]; then
            ((tag_build_count++))
        fi
    done
    
    echo -e "${GREEN}   Builds with descriptive tags: $tag_build_count${NC}"
    echo -e "${WHITE}   Framework coverage:${NC}"
    
    # Count framework types
    vanilla_count=0
    gin_count=0
    fiber_count=0
    
    for build_file in "${BUILD_FILES[@]}"; do
        build_name=$(basename "$build_file")
        if [[ "$build_name" == *"tags"* ]]; then
            if [[ "$build_name" == *"vanilla"* ]]; then
                ((vanilla_count++))
            fi
            if [[ "$build_name" == *"gin"* ]]; then
                ((gin_count++))
            fi
            if [[ "$build_name" == *"fiber"* ]]; then
                ((fiber_count++))
            fi
        fi
    done
    
    echo -e "${GRAY}     • Vanilla: $vanilla_count builds${NC}"
    echo -e "${GRAY}     • Gin: $gin_count builds${NC}"
    echo -e "${GRAY}     • Fiber: $fiber_count builds${NC}"
    
    echo ""
    echo -e "${CYAN}💡 Usage Tips:${NC}"
    echo -e "${GRAY}   • Tag-based names clearly show included providers${NC}"
    echo -e "${GRAY}   • Choose builds based on your deployment requirements${NC}"
    echo -e "${GRAY}   • All builds include mock providers for testing${NC}"
    
else
    echo -e "${RED}❌ No build directory found. Run some builds first!${NC}"
fi

echo ""