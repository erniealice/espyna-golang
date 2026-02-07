#!/bin/bash
#
# Interactive menu for building Espyna server variations with different technology stacks
#
# This script provides a user-friendly menu to choose from pre-configured build variations:
# 1. Fiber + Firebase (High-performance + Cloud-native)
# 2. Gin + Microsoft (Enterprise + Office 365)
# 3. Vanilla + PostgreSQL (Minimal + Self-hosted)
# 4. Multi-Hybrid (All frameworks + All providers)
# 5. Custom build with manual tag selection
#
# PARAMETERS:
#   -a, --auto-build NUM     Skip menu and build specific variation (1-10)
#   -m, --mock-mode BOOL     Include mock providers for testing (default: true)
#   -h, --help               Show this help message
#
# EXAMPLES:
#   ./build-menu.sh
#       Interactive menu for build selection
#
#   ./build-menu.sh -a 1 -m false
#       Auto-build Fiber + Firebase without mock providers
#
# NOTES:
#   Each build variation is optimized for specific use cases:
#   - Choose based on deployment target and technology preferences
#   - Mock providers are included by default for development/testing
#

set -euo pipefail

# Default values
AUTO_BUILD=""
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
    echo "Interactive menu for building Espyna server variations with different technology stacks."
    echo ""
    echo "OPTIONS:"
    echo "  -a, --auto-build NUM     Skip menu and build specific variation (1-10)"
    echo "  -m, --mock-mode BOOL     Include mock providers for testing (default: true)"
    echo "  -h, --help               Show this help message"
    echo ""
    echo "EXAMPLES:"
    echo "  $0"
    echo "  $0 -a 1 -m false"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -a|--auto-build)
            AUTO_BUILD="$2"
            shift 2
            ;;
        -m|--mock-mode)
            case "$2" in
                true|yes|1) MOCK_MODE=true ;;
                false|no|0) MOCK_MODE=false ;;
                *) echo "Error: Invalid mock-mode value '$2'. Use true/false" && exit 1 ;;
            esac
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
ESPYNA_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"
cd "$ESPYNA_DIR"

echo ""
echo -e "${CYAN}╔══════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║                    Espyna Server Build Menu                      ║${NC}"
echo -e "${CYAN}╚══════════════════════════════════════════════════════════════════╝${NC}"
echo ""

if [[ -z "$AUTO_BUILD" ]]; then
    echo -e "${WHITE}Choose your technology stack:${NC}"
    echo ""
    echo -e "${CYAN}=== CLOUD-NATIVE STACKS ===${NC}"
    echo ""
    echo -e "${GREEN}1. 🚀 Fiber + Firebase Stack${NC}"
    echo -e "${GRAY}   ├─ HTTP: Fiber (ultra-fast, Express.js-like)${NC}"
    echo -e "${GRAY}   ├─ Database: Firestore (NoSQL, real-time)${NC}"
    echo -e "${GRAY}   ├─ Auth: Firebase Auth (Google identity)${NC}"
    echo -e "${GRAY}   └─ Services: Google Cloud (Gmail, Storage)${NC}"
    echo -e "${YELLOW}   💡 Best for: Modern web apps, real-time features, Google ecosystem${NC}"
    echo ""
    
    echo -e "${GREEN}2. 🏢 Gin + Microsoft Stack${NC}"
    echo -e "${GRAY}   ├─ HTTP: Gin (flexible, middleware-rich)${NC}"
    echo -e "${GRAY}   ├─ Database: PostgreSQL (enterprise grade)${NC}"
    echo -e "${GRAY}   ├─ Auth: Azure Active Directory${NC}"
    echo -e "${GRAY}   └─ Services: Microsoft Graph (Office 365, Teams)${NC}"
    echo -e "${YELLOW}   💡 Best for: Enterprise apps, Office integration, Microsoft ecosystem${NC}"
    echo ""
    
    echo -e "${GREEN}3. ☁️  Fiber + AWS Stack${NC}"
    echo -e "${GRAY}   ├─ HTTP: Fiber (ultra-high performance)${NC}"
    echo -e "${GRAY}   ├─ Database: PostgreSQL on RDS (managed)${NC}"
    echo -e "${GRAY}   ├─ Auth: JWT (stateless for auto-scaling)${NC}"
    echo -e "${GRAY}   └─ Services: AWS S3, Amazon SES${NC}"
    echo -e "${YELLOW}   💡 Best for: AWS cloud deployments, ECS/EKS, Lambda${NC}"
    echo ""
    
    echo -e "${CYAN}=== DEPLOYMENT-OPTIMIZED STACKS ===${NC}"
    echo ""
    
    echo -e "${GREEN}4. ⚡ Minimal Edge Stack${NC}"
    echo -e "${GRAY}   ├─ HTTP: Vanilla (Go standard library)${NC}"
    echo -e "${GRAY}   ├─ Database: Mock in-memory (ultra-fast)${NC}"
    echo -e "${GRAY}   ├─ Auth: JWT (no external dependencies)${NC}"
    echo -e "${GRAY}   └─ Services: Local storage, Mock email${NC}"
    echo -e "${YELLOW}   💡 Best for: IoT, edge computing, Raspberry Pi, minimal containers${NC}"
    echo ""
    
    echo -e "${GREEN}5. 🐳 Container + K8s Stack${NC}"
    echo -e "${GRAY}   ├─ HTTP: Fiber (container-optimized performance)${NC}"
    echo -e "${GRAY}   ├─ Database: Multi-cloud (PostgreSQL, Firestore)${NC}"
    echo -e "${GRAY}   ├─ Auth: JWT (horizontal scaling ready)${NC}"
    echo -e "${GRAY}   └─ Services: Multi-cloud storage, Health checks${NC}"
    echo -e "${YELLOW}   💡 Best for: Docker containers, Kubernetes, service mesh${NC}"
    echo ""
    
    echo -e "${CYAN}=== DEVELOPMENT & ENTERPRISE ===${NC}"
    echo ""
    
    echo -e "${GREEN}6. 🛠️  Development Debug Stack${NC}"
    echo -e "${GRAY}   ├─ HTTP: Gin (hot-reload friendly)${NC}"
    echo -e "${GRAY}   ├─ Database: Mock providers (offline development)${NC}"
    echo -e "${GRAY}   ├─ Auth: All providers (integration testing)${NC}"
    echo -e "${GRAY}   └─ Services: Mock + real providers, race detection${NC}"
    echo -e "${YELLOW}   💡 Best for: Local development, API testing, debugging${NC}"
    echo ""
    
    echo -e "${GREEN}7. 🏆 Enterprise Complete Stack${NC}"
    echo -e "${GRAY}   ├─ HTTP: Gin (enterprise middleware)${NC}"
    echo -e "${GRAY}   ├─ Database: Multi-provider (PostgreSQL + Firestore backup)${NC}"
    echo -e "${GRAY}   ├─ Auth: All providers (Azure AD, Firebase, JWT)${NC}"
    echo -e "${GRAY}   └─ Services: Full multi-cloud support${NC}"
    echo -e "${YELLOW}   💡 Best for: Large enterprises, maximum flexibility, hybrid cloud${NC}"
    echo ""
    
    echo -e "${CYAN}=== LEGACY & CUSTOM ===${NC}"
    echo ""
    
    echo -e "${GREEN}8. ⚡ Vanilla + PostgreSQL Stack${NC}"
    echo -e "${GRAY}   ├─ HTTP: Vanilla (standard library, minimal)${NC}"
    echo -e "${GRAY}   ├─ Database: PostgreSQL (reliable, ACID)${NC}"
    echo -e "${GRAY}   ├─ Auth: JWT (stateless, self-hosted)${NC}"
    echo -e "${GRAY}   └─ Services: SMTP, Local storage${NC}"
    echo -e "${YELLOW}   💡 Best for: Self-hosted, minimal dependencies, traditional deployment${NC}"
    echo ""
    
    echo -e "${GREEN}9. 🌐 Multi-Hybrid Stack${NC}"
    echo -e "${GRAY}   ├─ HTTP: All frameworks (runtime switchable)${NC}"
    echo -e "${GRAY}   ├─ Database: Multiple providers (runtime switchable)${NC}"
    echo -e "${GRAY}   ├─ Auth: All providers (runtime switchable)${NC}"
    echo -e "${GRAY}   └─ Services: Everything included${NC}"
    echo -e "${YELLOW}   💡 Best for: Maximum flexibility, A/B testing, migration scenarios${NC}"
    echo ""
    
    echo -e "${GREEN}10. 🔧 Custom Build${NC}"
    echo -e "${GRAY}    └─ Manual tag selection with build-with-tags.sh${NC}"
    echo -e "${YELLOW}    💡 Best for: Specific combinations not covered above${NC}"
    echo ""
    
    if [[ "$MOCK_MODE" == true ]]; then
        echo -e "${CYAN}🧪 Mock providers will be included for testing/development${NC}"
    else
        echo -e "${MAGENTA}🔧 Production build - mock providers excluded${NC}"
    fi
    echo ""
    
    echo -n "Enter your choice (1-10): "
    read CHOICE
else
    CHOICE="$AUTO_BUILD"
    echo -e "${CYAN}Auto-building option $CHOICE${NC}"
fi

SCRIPT_PATH=""
DESCRIPTION=""

case "$CHOICE" in
    1)
        SCRIPT_PATH="./scripts/build/cloud-specific/build-fiber-firebase.sh"
        DESCRIPTION="Fiber + Firebase Stack"
        ;;
    2)
        SCRIPT_PATH="./scripts/build/cloud-specific/build-gin-microsoft.sh"
        DESCRIPTION="Gin + Microsoft Stack"
        ;;
    3)
        SCRIPT_PATH="./scripts/build/cloud-specific/build-fiber-aws.sh"
        DESCRIPTION="Fiber + AWS Stack"
        ;;
    4)
        SCRIPT_PATH="./scripts/build/legacy/build-minimal-edge.sh"
        DESCRIPTION="Minimal Edge Stack"
        ;;
    5)
        SCRIPT_PATH="./scripts/build/legacy/build-container-k8s.sh"
        DESCRIPTION="Container + K8s Stack"
        ;;
    6)
        SCRIPT_PATH="./scripts/build/development/build-development-debug.sh"
        DESCRIPTION="Development Debug Stack"
        ;;
    7)
        SCRIPT_PATH="./scripts/build/enterprise/build-enterprise-complete.sh"
        DESCRIPTION="Enterprise Complete Stack"
        ;;
    8)
        SCRIPT_PATH="./scripts/build/legacy/build-vanilla-postgres.sh"
        DESCRIPTION="Vanilla + PostgreSQL Stack"
        ;;
    9)
        SCRIPT_PATH="./scripts/build/enterprise/build-multi-hybrid.sh"
        DESCRIPTION="Multi-Hybrid Stack"
        ;;
    10)
        echo ""
        echo -e "${CYAN}Launching custom build script...${NC}"
        echo -e "${WHITE}Use: ./scripts/build/build-with-tags.sh -f <framework> -s <tags>${NC}"
        echo ""
        echo -e "${GRAY}Available frameworks: vanilla, gin, fiber${NC}"
        echo -e "${GRAY}Available secondary tags: firestore, firebase, google, microsoft, azure, postgres, jwt, etc.${NC}"
        echo ""
        echo -e "${YELLOW}Example: ./scripts/build/build-with-tags.sh -f fiber -s 'firestore,firebase'${NC}"
        exit 0
        ;;
    *)
        echo -e "${RED}Invalid choice. Please select 1-10.${NC}"
        exit 1
        ;;
esac

echo ""
echo -e "${CYAN}🔨 Building: $DESCRIPTION${NC}"
echo -e "${GRAY}📁 Script: $SCRIPT_PATH${NC}"

# Build the arguments for the script
BUILD_ARGS=()
if [[ "$MOCK_MODE" == true ]]; then
    BUILD_ARGS+=("-m" "true")
else
    BUILD_ARGS+=("-m" "false")
fi

echo -e "${BLUE}⚙️  Executing build script...${NC}"
echo ""

# Check if the script exists and is executable
if [[ ! -f "$SCRIPT_PATH" ]]; then
    echo -e "${RED}❌ Build script not found: $SCRIPT_PATH${NC}"
    echo -e "${YELLOW}💡 This script may not have a bash equivalent yet. Try the PowerShell version:${NC}"
    POWERSHELL_PATH="${SCRIPT_PATH%.sh}.ps1"
    echo -e "${WHITE}   $POWERSHELL_PATH${NC}"
    exit 1
fi

if [[ ! -x "$SCRIPT_PATH" ]]; then
    echo -e "${YELLOW}⚠️  Making script executable: $SCRIPT_PATH${NC}"
    chmod +x "$SCRIPT_PATH"
fi

# Execute the selected build script
if "$SCRIPT_PATH" "${BUILD_ARGS[@]}"; then
    echo ""
    echo -e "${GREEN}✅ Build completed successfully!${NC}"
    echo ""
    echo -e "${CYAN}Next steps:${NC}"
    echo -e "${WHITE}1. Test your build with the provided usage examples${NC}"
    echo -e "${WHITE}2. Configure environment variables for your target deployment${NC}"
    echo -e "${WHITE}3. Review the build-specific documentation above${NC}"
    echo ""
    echo -e "${BLUE}📂 Built binaries are located in: packages/espyna/build/${NC}"
else
    echo ""
    echo -e "${RED}❌ Build failed. Check the error messages above.${NC}"
    exit 1
fi

echo ""
echo -e "${CYAN}🎯 Build Menu Benefits:${NC}"
echo -e "${GRAY}   • Simplified build process with pre-configured stacks${NC}"
echo -e "${GRAY}   • Technology-specific optimizations for each use case${NC}"
echo -e "${GRAY}   • Easy switching between development and production builds${NC}"
echo -e "${GRAY}   • Clear documentation for each stack configuration${NC}"