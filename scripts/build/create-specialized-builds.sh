#!/bin/bash
#
# Create multiple specialized Espyna builds with descriptive tag-based names
#
# This script creates several specialized builds using the working tag combinations
# but with descriptive filenames that clearly indicate what capabilities each build has.
#

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
ESPYNA_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"
cd "$ESPYNA_DIR"

echo -e "${CYAN}🏭 Creating Specialized Espyna Builds${NC}"
echo -e "${WHITE}Each build will have a descriptive name showing its capabilities${NC}"
echo ""

# Working tag set that we know compiles successfully
BASE_WORKING_TAGS=(
    "providers_bootstrap"
    "mock_db" "mock_email" "mock_storage"
    "local_storage" "google" "aws" "s3"
    "microsoft" "microsoftgraph" "gmail" "gcp_storage"
    "noop" "postgres" "firestore" "firebase"
    "postgres_migrations"
)

# Function to join array elements with a delimiter
join_by() {
    local IFS="$1"
    shift
    echo "$*"
}

# Function to remove duplicates from an array
remove_duplicates() {
    local -a unique_array=()
    local seen=()
    
    for item in "$@"; do
        local found=false
        for seen_item in "${seen[@]:-}"; do
            if [[ "$item" == "$seen_item" ]]; then
                found=true
                break
            fi
        done
        if [[ "$found" == false ]]; then
            unique_array+=("$item")
            seen+=("$item")
        fi
    done
    
    echo "${unique_array[@]}"
}

# Define specialized builds with their intended focus
declare -A builds

# Build 1: Vanilla Self-hosted
builds[vanilla]="espyna-vanilla-selfhosted-postgres-jwt-local-tags|vanilla|Self-hosted vanilla server with PostgreSQL and JWT|postgres,jwt_auth,local_storage"

# Build 2: Gin Enterprise
builds[gin_enterprise]="espyna-gin-enterprise-microsoft-azure-graph-tags|gin|Enterprise Gin server with Microsoft ecosystem|microsoft,microsoftgraph"

# Build 3: Fiber Google Cloud
builds[fiber_google]="espyna-fiber-cloud-google-firebase-firestore-tags|fiber|High-performance Fiber with Google Cloud services|google,firebase,firestore,gmail,gcp_storage"

# Build 4: Fiber AWS
builds[fiber_aws]="espyna-fiber-aws-postgres-s3-lambda-tags|fiber|AWS-optimized Fiber server for Lambda/ECS|aws,s3,postgres"

# Build 5: Gin Multi-cloud
builds[gin_hybrid]="espyna-gin-hybrid-multicloud-enterprise-tags|gin|Multi-cloud enterprise Gin server|google,microsoft,aws,postgres,firestore"

for build_key in "${!builds[@]}"; do
    IFS='|' read -r name framework description focus_tags <<< "${builds[$build_key]}"
    
    echo -e "${GREEN}🔨 Building: $description${NC}"
    echo -e "${GRAY}   Framework: $framework${NC}"
    echo -e "${GRAY}   Focus: $focus_tags${NC}"
    echo -e "${BLUE}   Output: $name${NC}"
    
    # Convert focus_tags to array
    IFS=',' read -ra FOCUS_TAGS_ARRAY <<< "$focus_tags"
    
    # Combine base working tags with focus-specific tags
    ALL_TAGS_ARRAY=(${BASE_WORKING_TAGS[@]} ${FOCUS_TAGS_ARRAY[@]})
    
    # Remove duplicates and framework from secondary tags
    UNIQUE_TAGS=($(remove_duplicates "${ALL_TAGS_ARRAY[@]}"))
    SECONDARY_TAGS=()
    
    for tag in "${UNIQUE_TAGS[@]}"; do
        if [[ "$tag" != "$framework" ]]; then
            SECONDARY_TAGS+=("$tag")
        fi
    done
    
    SECONDARY_TAGS_STRING=$(join_by "," "${SECONDARY_TAGS[@]}")
    
    echo -e "${GRAY}   Executing build...${NC}"
    
    # Execute build
    if ./scripts/build/build-with-tags.sh -f "$framework" -s "$SECONDARY_TAGS_STRING" -o "$name"; then
        echo -e "${GREEN}   ✅ Success!${NC}"
        
        # Show binary info
        BINARY_PATH="build/$name"
        if [[ -f "$BINARY_PATH" ]]; then
            BINARY_SIZE_BYTES=$(stat -f%z "$BINARY_PATH" 2>/dev/null || stat -c%s "$BINARY_PATH" 2>/dev/null || echo "0")
            BINARY_SIZE_MB=$((BINARY_SIZE_BYTES / 1024 / 1024))
            echo -e "${BLUE}   📏 Size: ${BINARY_SIZE_MB} MB${NC}"
        fi
    else
        echo -e "${RED}   ❌ Failed!${NC}"
    fi
    
    echo ""
done

echo -e "${GREEN}🎉 Specialized builds complete!${NC}"
echo ""
echo -e "${CYAN}📂 All binaries are in the build/ directory:${NC}"

# List all binaries with sizes
if [[ -d "build" ]]; then
    for binary in build/*; do
        if [[ -f "$binary" ]]; then
            BINARY_SIZE_BYTES=$(stat -f%z "$binary" 2>/dev/null || stat -c%s "$binary" 2>/dev/null || echo "0")
            BINARY_SIZE_MB=$((BINARY_SIZE_BYTES / 1024 / 1024))
            BINARY_NAME=$(basename "$binary")
            echo -e "${WHITE}   $BINARY_NAME - ${BINARY_SIZE_MB} MB${NC}"
        fi
    done
fi

echo ""
echo -e "${CYAN}💡 Usage Tips:${NC}"
echo -e "${GRAY}   • Filenames clearly indicate capabilities (framework + key providers)${NC}"
echo -e "${GRAY}   • Choose builds based on your deployment target and requirements${NC}"
echo -e "${GRAY}   • All builds include mock providers for development/testing${NC}"