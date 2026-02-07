#!/bin/bash
#
# Build Espyna server with Fiber HTTP framework and AWS cloud services ecosystem
#
# DESCRIPTION:
#   This script creates a specialized build optimized for AWS deployment with:
#   - Fiber HTTP framework for ultra-high performance
#   - AWS S3 for scalable object storage
#   - PostgreSQL on AWS RDS for managed database
#   - JWT authentication for stateless scaling
#   - SMTP integration for Amazon SES email services
#
# PARAMETERS:
#   -o, --output OUTPUT       Output binary name (default: espyna-fiber-aws)
#   -v, --verbose            Enable verbose build output
#   -r, --race               Enable race condition detection
#   -m, --mock-mode BOOL     Include mock providers for testing (default: true)
#   -h, --help               Show this help message
#
# EXAMPLES:
#   ./build-fiber-aws.sh
#       Basic build with Fiber + AWS stack
#
#   ./build-fiber-aws.sh -v -r --mock-mode=false
#       Production build with verbose output and race detection
#
# NOTES:
#   This build configuration is optimized for:
#   - AWS cloud-native deployments (ECS, EKS, Lambda)
#   - High-performance applications with Fiber framework
#   - Scalable object storage with S3
#   - Managed PostgreSQL with RDS
#   - Cost-effective email with Amazon SES

set -euo pipefail

# Default values
OUTPUT="espyna-fiber-aws"
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
    echo "Build Espyna server with Fiber HTTP framework and AWS cloud services ecosystem"
    echo ""
    echo "OPTIONS:"
    echo "  -o, --output OUTPUT       Output binary name (default: espyna-fiber-aws)"
    echo "  -v, --verbose            Enable verbose build output"
    echo "  -r, --race               Enable race condition detection"
    echo "  -m, --mock-mode BOOL     Include mock providers for testing (default: true)"
    echo "  -h, --help               Show this help message"
    echo ""
    echo "EXAMPLES:"
    echo "  $0"
    echo "  $0 -v -r --mock-mode=false"
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
            if [[ "$2" == "true" ]]; then
                MOCK_MODE=true
            elif [[ "$2" == "false" ]]; then
                MOCK_MODE=false
            else
                echo -e "${RED}Error: Invalid mock-mode value '$2'. Use 'true' or 'false'${NC}"
                exit 1
            fi
            shift 2
            ;;
        --mock-mode=*)
            VALUE="${1#*=}"
            if [[ "$VALUE" == "true" ]]; then
                MOCK_MODE=true
            elif [[ "$VALUE" == "false" ]]; then
                MOCK_MODE=false
            else
                echo -e "${RED}Error: Invalid mock-mode value '$VALUE'. Use 'true' or 'false'${NC}"
                exit 1
            fi
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

echo -e "${CYAN}=== Espyna Fiber + AWS Build ===${NC}"
echo -e "${WHITE}Building AWS-optimized server with:${NC}"
echo -e "${GREEN}  • HTTP Framework: Fiber (ultra-high performance)${NC}"
echo -e "${GREEN}  • Database: PostgreSQL on AWS RDS (managed)${NC}"
echo -e "${GREEN}  • Authentication: JWT (stateless for auto-scaling)${NC}"
echo -e "${GREEN}  • Storage: AWS S3 (infinite scalability)${NC}"
echo -e "${GREEN}  • Email: SMTP (Amazon SES integration)${NC}"
echo -e "${GREEN}  • Deployment: ECS/EKS/Lambda ready${NC}"
if [[ "$MOCK_MODE" == true ]]; then
    echo -e "${YELLOW}  • Mock providers included for testing${NC}"
fi
echo ""

# AWS-optimized build tags
BUILD_TAGS=("fiber" "providers_bootstrap" "postgres" "jwt_auth" "aws" "s3" "postgres_migrations")
if [[ "$MOCK_MODE" == true ]]; then
    BUILD_TAGS+=("mock_db" "mock_email" "mock_storage")
fi
# Include essential fallback providers
BUILD_TAGS+=("local_storage" "noop" "google" "firebase" "firestore" "microsoft")

echo -e "${GRAY}Working directory: $(pwd)${NC}"
echo -e "${BLUE}Build tags: $(IFS=','; echo "${BUILD_TAGS[*]}")${NC}"
echo ""

# Use the quick-build approach with tested tag combinations
SECONDARY_TAGS=()
for tag in "${BUILD_TAGS[@]}"; do
    if [[ "$tag" != "fiber" ]]; then
        SECONDARY_TAGS+=("$tag")
    fi
done
SECONDARY_TAGS_STRING=$(IFS=','; echo "${SECONDARY_TAGS[*]}")

echo -e "${MAGENTA}Executing: ./scripts/build-with-tags.sh -f fiber -s '$SECONDARY_TAGS_STRING' -o $OUTPUT${NC}"
echo ""

# Build arguments for the main build script
BUILD_ARGS=(-f "fiber" -s "$SECONDARY_TAGS_STRING" -o "$OUTPUT")
if [[ "$VERBOSE_BUILD" == true ]]; then
    BUILD_ARGS+=(-v)
fi
if [[ "$RACE" == true ]]; then
    BUILD_ARGS+=(-r)
fi

# Execute build using the working build script
if ./scripts/build-with-tags.sh "${BUILD_ARGS[@]}"; then
    echo ""
    echo -e "${GREEN}✅ AWS-optimized build completed!${NC}"
    
    # Show binary info
    OUTPUT_PATH="build/$OUTPUT"
    if [[ -f "$OUTPUT_PATH" ]]; then
        echo -e "${GREEN}✅ Binary created: $OUTPUT_PATH${NC}"
        BINARY_SIZE_BYTES=$(stat -f%z "$OUTPUT_PATH" 2>/dev/null || stat -c%s "$OUTPUT_PATH" 2>/dev/null || echo "0")
        BINARY_SIZE_MB=$((BINARY_SIZE_BYTES / 1024 / 1024))
        echo -e "${BLUE}✅ Binary size: ${BINARY_SIZE_MB} MB${NC}"
    fi
    
    echo ""
    echo -e "${CYAN}🚀 AWS Deployment Examples:${NC}"
    echo ""
    echo -e "${WHITE}  Local Development:${NC}"
    echo -e "${GRAY}    MOCK_MODE=true MOCK_BUSINESS_TYPE=education ./$OUTPUT_PATH${NC}"
    echo ""
    echo -e "${WHITE}  AWS ECS Deployment:${NC}"
    echo -e "${GRAY}    SERVER_TYPE=fiber SERVER_PORT=8080 \\${NC}"
    echo -e "${GRAY}    DATABASE_URL=\$RDS_DATABASE_URL \\${NC}"
    echo -e "${GRAY}    AWS_REGION=us-east-1 \\${NC}"
    echo -e "${GRAY}    S3_BUCKET_NAME=my-app-storage \\${NC}"
    echo -e "${GRAY}    JWT_SECRET=\$JWT_SECRET_FROM_SECRETS_MANAGER \\${NC}"
    echo -e "${GRAY}    ./$OUTPUT_PATH${NC}"
    echo ""
    echo -e "${WHITE}  AWS Lambda (with adapter):${NC}"
    echo -e "${GRAY}    AWS_LAMBDA_RUNTIME_API=\$AWS_LAMBDA_RUNTIME_API \\${NC}"
    echo -e "${GRAY}    DATABASE_URL=\$RDS_PROXY_ENDPOINT \\${NC}"
    echo -e "${GRAY}    ./$OUTPUT_PATH${NC}"
    echo ""
    echo -e "${CYAN}📋 Environment Variables:${NC}"
    echo -e "${GRAY}  SERVER_TYPE=fiber                    # HTTP framework${NC}"
    echo -e "${GRAY}  SERVER_PORT=8080                     # Server port${NC}"
    echo -e "${GRAY}  DATABASE_URL=postgres://...          # RDS connection string${NC}"
    echo -e "${GRAY}  JWT_SECRET=secret-from-ssm           # JWT signing key (use AWS SSM)${NC}"
    echo -e "${GRAY}  AWS_REGION=us-east-1                 # AWS region${NC}"
    echo -e "${GRAY}  AWS_ACCESS_KEY_ID=AKIAXXXXX          # AWS credentials (or use IAM roles)${NC}"
    echo -e "${GRAY}  AWS_SECRET_ACCESS_KEY=secret         # AWS secret (or use IAM roles)${NC}"
    echo -e "${GRAY}  S3_BUCKET_NAME=my-storage-bucket     # S3 bucket for file storage${NC}"
    echo -e "${GRAY}  SES_FROM_EMAIL=noreply@example.com   # Amazon SES sender email${NC}"
    echo -e "${GRAY}  MOCK_MODE=true                       # Enable mock providers (dev only)${NC}"
    
else
    echo ""
    echo -e "${RED}❌ Build failed!${NC}"
    exit 1
fi

echo ""
echo -e "${CYAN}☁️ AWS + Fiber Stack Benefits:${NC}"
echo -e "${GRAY}   • Ultra-fast HTTP performance with Fiber framework${NC}"
echo -e "${GRAY}   • Infinite scalability with AWS S3 object storage${NC}"
echo -e "${GRAY}   • Managed PostgreSQL with AWS RDS (automated backups, scaling)${NC}"
echo -e "${GRAY}   • Cost-effective email delivery with Amazon SES${NC}"
echo -e "${GRAY}   • Stateless JWT authentication for auto-scaling groups${NC}"
echo -e "${GRAY}   • Container-ready for ECS, EKS, and Fargate deployments${NC}"
echo -e "${GRAY}   • Lambda-compatible for serverless architectures${NC}"