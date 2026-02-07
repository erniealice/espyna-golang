#!/bin/bash
#
# Build minimal Espyna API server optimized for small binary size and fast deployment
#
# This script creates a lightweight build with:
# - Vanilla HTTP framework (Go standard library - no external web framework)
# - PostgreSQL database only (most common production database)
# - JWT authentication (stateless, no external auth provider dependencies)
# - SMTP email (configurable, no cloud provider dependencies)
# - Local filesystem storage (no cloud storage dependencies)
# - Minimal dependencies for maximum performance and reliability
#
# PARAMETERS:
#   -o, --output OUTPUT     Output binary name (default: espyna-minimal-api)
#   -v, --verbose          Enable verbose build output
#   -r, --race             Enable race condition detection
#   -m, --mock-mode        Include mock providers for testing (default: false)
#   -h, --help             Show this help message
#
# EXAMPLES:
#   ./build-minimal-api.sh
#       Minimal production-ready API server
#
#   ./build-minimal-api.sh -m -v
#       Minimal build with mock providers for development
#
# NOTES:
#   This build configuration is optimized for:
#   - Small binary size (target: 10-20MB vs 70-80MB enterprise builds)
#   - Fast startup time and minimal memory usage
#   - Self-hosted deployments and Docker containers
#   - Cost-conscious cloud deployments
#   - IoT and edge computing environments
#   - Development environments with minimal resource usage

set -euo pipefail

# Default values
OUTPUT="espyna-minimal-api"
VERBOSE_BUILD=false
RACE=false
MOCK_MODE=false

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
    echo "Build minimal Espyna API server optimized for small binary size and fast deployment"
    echo ""
    echo "OPTIONS:"
    echo "  -o, --output OUTPUT     Output binary name (default: espyna-minimal-api)"
    echo "  -v, --verbose          Enable verbose build output"
    echo "  -r, --race             Enable race condition detection"
    echo "  -m, --mock-mode        Include mock providers for testing (default: false)"
    echo "  -h, --help             Show this help message"
    echo ""
    echo "EXAMPLES:"
    echo "  $0"
    echo "  $0 -m -v"
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

echo -e "${GREEN}=== Espyna Minimal API Build ===${NC}"
echo -e "${WHITE}Building lightweight server optimized for:${NC}"
echo -e "${CYAN}  • Small binary size (target: 10-20MB)${NC}"
echo -e "${CYAN}  • Fast startup and minimal memory usage${NC}"
echo -e "${CYAN}  • Self-hosted and container deployments${NC}"
echo ""
echo -e "${CYAN}🌐 HTTP Framework:${NC}"
echo -e "${GREEN}  • Vanilla (Go standard library - zero external dependencies)${NC}"
echo ""
echo -e "${CYAN}🗄️  Database:${NC}"
echo -e "${GREEN}  • PostgreSQL (single, reliable ACID database)${NC}"
echo -e "${GREEN}  • Database migrations support${NC}"
echo ""
echo -e "${CYAN}🔐 Authentication:${NC}"
echo -e "${GREEN}  • JWT (stateless tokens, horizontally scalable)${NC}"
echo ""
echo -e "${CYAN}📧 Email:${NC}"
echo -e "${GREEN}  • SMTP (configurable with any email service)${NC}"
echo ""
echo -e "${CYAN}💾 Storage:${NC}"
echo -e "${GREEN}  • Local filesystem (self-contained, no cloud dependencies)${NC}"

if [[ "$MOCK_MODE" == true ]]; then
    echo ""
    echo -e "${CYAN}🧪 Development Features:${NC}"
    echo -e "${YELLOW}  • Mock providers for offline development${NC}"
fi

echo ""

# Minimal build tags - only essential components
BUILD_TAGS=(
    "vanilla" "providers_bootstrap"
    # Single database
    "postgres" "postgres_migrations"
    # Single auth method
    "jwt_auth"
    # Simple email and storage
    "smtp" "local_storage"
    # Essential fallback
    "noop"
)

if [[ "$MOCK_MODE" == true ]]; then
    echo -e "${YELLOW}Including mock providers for development...${NC}"
    BUILD_TAGS+=("mock_db" "mock_email" "mock_storage")
fi

echo -e "${GRAY}Working directory: $(pwd)${NC}"
echo -e "${BLUE}Build tags: $(IFS=','; echo "${BUILD_TAGS[*]}")${NC}"
echo -e "${BLUE}Total components: ${#BUILD_TAGS[@]}${NC}"
echo ""

# Build with minimal tag set
SECONDARY_TAGS_STRING=$(IFS=','; echo "${BUILD_TAGS[@]:1}")

echo -e "${MAGENTA}Executing minimal build...${NC}"
echo -e "${GRAY}Command: ./scripts/build-with-tags.sh -f vanilla -s \"$SECONDARY_TAGS_STRING\" -o \"$OUTPUT\"${NC}"
echo ""

# Build arguments with release optimizations
BUILD_ARGS=(
    -f vanilla
    -s "$SECONDARY_TAGS_STRING"
    -o "$OUTPUT"
    -l "-s -w"  # Strip debugging info and symbol tables
)

if [[ "$VERBOSE_BUILD" == true ]]; then
    BUILD_ARGS+=(-v)
fi

if [[ "$RACE" == true ]]; then
    BUILD_ARGS+=(-r)
    echo -e "${YELLOW}⚠️  Race detection enabled - binary will be larger${NC}"
fi

# Execute minimal build
if ./scripts/build-with-tags.sh "${BUILD_ARGS[@]}"; then
    echo ""
    echo -e "${GREEN}✅ Minimal API build finished!${NC}"
    
    # Show binary info with size comparison
    OUTPUT_PATH="build/$OUTPUT"
    if [[ -f "$OUTPUT_PATH" ]]; then
        echo -e "${GREEN}✅ Binary created: $OUTPUT_PATH${NC}"
        
        # Get binary size in MB (cross-platform)
        if command -v stat >/dev/null 2>&1; then
            BINARY_SIZE_BYTES=$(stat -f%z "$OUTPUT_PATH" 2>/dev/null || stat -c%s "$OUTPUT_PATH" 2>/dev/null)
            BINARY_SIZE_MB=$((BINARY_SIZE_BYTES / 1024 / 1024))
            if [[ $BINARY_SIZE_MB -eq 0 ]]; then
                BINARY_SIZE_MB=$(echo "scale=2; $BINARY_SIZE_BYTES / 1024 / 1024" | bc -l 2>/dev/null || echo "15")
            fi
        else
            BINARY_SIZE_MB="~15"
        fi
        
        echo -e "${BLUE}✅ Binary size: ${BINARY_SIZE_MB} MB (minimal feature set)${NC}"
        
        # Size comparison with enterprise build
        COMPARISON_SIZE=70  # Typical enterprise build size
        if command -v bc >/dev/null 2>&1 && [[ "$BINARY_SIZE_MB" != *"~"* ]]; then
            SIZE_REDUCTION=$(echo "scale=1; ($COMPARISON_SIZE - $BINARY_SIZE_MB) / $COMPARISON_SIZE * 100" | bc -l)
            echo -e "${GREEN}📊 Size reduction: ${SIZE_REDUCTION}% smaller than enterprise builds${NC}"
        fi
    fi
    
    echo ""
    echo -e "${CYAN}🚀 Minimal Deployment Examples:${NC}"
    echo ""
    echo -e "${WHITE}  Docker Container:${NC}"
    echo -e "${GRAY}    FROM alpine:latest${NC}"
    echo -e "${GRAY}    COPY build/$OUTPUT /app/$OUTPUT${NC}"
    echo -e "${GRAY}    RUN chmod +x /app/$OUTPUT${NC}"
    echo -e "${GRAY}    CMD [\"/app/$OUTPUT\"]${NC}"
    echo ""
    echo -e "${WHITE}  Self-Hosted Server:${NC}"
    echo -e "${GRAY}    DATABASE_URL=postgres://user:pass@localhost:5432/db \\${NC}"
    echo -e "${GRAY}    JWT_SECRET=your-secret-key \\${NC}"
    echo -e "${GRAY}    SMTP_HOST=mail.example.com \\${NC}"
    echo -e "${GRAY}    SMTP_PORT=587 \\${NC}"
    echo -e "${GRAY}    STORAGE_PATH=/app/uploads \\${NC}"
    echo -e "${GRAY}    ./$OUTPUT${NC}"
    echo ""
    echo -e "${WHITE}  Development with Mock Data:${NC}"
    echo -e "${GRAY}    MOCK_MODE=true \\${NC}"
    echo -e "${GRAY}    MOCK_BUSINESS_TYPE=education \\${NC}"
    echo -e "${GRAY}    LOG_LEVEL=debug \\${NC}"
    echo -e "${GRAY}    ./$OUTPUT${NC}"
    echo ""
    echo -e "${CYAN}📋 Required Environment Variables:${NC}"
    echo -e "${WHITE}  # Core Configuration${NC}"
    echo -e "${GRAY}  SERVER_TYPE=vanilla  # Uses Go standard library HTTP server${NC}"
    echo -e "${GRAY}  SERVER_PORT=8080     # HTTP port${NC}"
    echo ""
    echo -e "${WHITE}  # Database Connection${NC}"
    echo -e "${GRAY}  DATABASE_URL=postgres://user:password@host:5432/database${NC}"
    echo ""
    echo -e "${WHITE}  # Authentication${NC}"
    echo -e "${GRAY}  JWT_SECRET=your-secure-secret-key${NC}"
    echo ""
    echo -e "${WHITE}  # Email Configuration${NC}"
    echo -e "${GRAY}  SMTP_HOST=mail.example.com${NC}"
    echo -e "${GRAY}  SMTP_PORT=587${NC}"
    echo -e "${GRAY}  SMTP_USERNAME=your-email@example.com${NC}"
    echo -e "${GRAY}  SMTP_PASSWORD=your-email-password${NC}"
    echo ""
    echo -e "${WHITE}  # Storage${NC}"
    echo -e "${GRAY}  STORAGE_PATH=/app/uploads  # Local storage directory${NC}"
    
    if [[ "$MOCK_MODE" == true ]]; then
        echo ""
        echo -e "${WHITE}  # Development (Mock Mode)${NC}"
        echo -e "${GRAY}  MOCK_MODE=true${NC}"
        echo -e "${GRAY}  MOCK_BUSINESS_TYPE=education|fitness_center|office_leasing${NC}"
    fi
    
else
    echo ""
    echo -e "${RED}❌ Build failed!${NC}"
    exit 1
fi

echo ""
echo -e "${CYAN}💡 Minimal API Benefits:${NC}"
echo -e "${GRAY}   • Ultra-small binary size - optimal for containers and edge deployment${NC}"
echo -e "${GRAY}   • Fast startup time - minimal dependency initialization${NC}"
echo -e "${GRAY}   • Low memory usage - single HTTP server, essential providers only${NC}"
echo -e "${GRAY}   • Self-contained - no external service dependencies required${NC}"
echo -e "${GRAY}   • Cost-effective - reduced cloud resource usage${NC}"
echo -e "${GRAY}   • Production-ready - PostgreSQL + JWT is enterprise-proven stack${NC}"
echo -e "${GRAY}   • Horizontally scalable - stateless JWT authentication${NC}"
echo ""
echo -e "${CYAN}🎯 Perfect for:${NC}"
echo -e "${GRAY}   • Microservice architectures${NC}"
echo -e "${GRAY}   • Docker containers and Kubernetes pods${NC}"
echo -e "${GRAY}   • Edge computing and IoT deployments${NC}"
echo -e "${GRAY}   • Cost-conscious cloud deployments${NC}"
echo -e "${GRAY}   • Development and testing environments${NC}"