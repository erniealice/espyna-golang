#!/bin/bash

# Test all list endpoints on a running server
# Usage: ./scripts/test-list-endpoints.sh <port> <server_name>

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Check arguments
if [ $# -ne 2 ]; then
    echo -e "${RED}Usage: $0 <port> <server_name>${NC}"
    echo -e "${YELLOW}Example: $0 8080 Vanilla${NC}"
    exit 1
fi

PORT=$1
SERVER_NAME=$2

# Function to test a single endpoint
test_endpoint() {
    local endpoint=$1

    # Use GET for subscription plan list, POST for others
    if [[ "$endpoint" == "/api/subscription/plan/list" ]]; then
        method="GET"
        echo -e "${CYAN}Testing ${SERVER_NAME}: GET localhost:${PORT}${endpoint}${NC}"
        response=$(curl -s -o /dev/null -w "%{http_code}" \
            -X GET \
            -H "Content-Type: application/json" \
            "http://localhost:${PORT}${endpoint}")
    else
        method="POST"
        echo -e "${CYAN}Testing ${SERVER_NAME}: POST localhost:${PORT}${endpoint}${NC}"
        response=$(curl -s -o /dev/null -w "%{http_code}" \
            -X POST \
            -H "Content-Type: application/json" \
            -d '{}' \
            "http://localhost:${PORT}${endpoint}")
    fi

    if [[ $response -eq 200 ]]; then
        echo -e "${GREEN}✓ ${endpoint} - OK (${response})${NC}"
        return 0
    else
        echo -e "${RED}✗ ${endpoint} - FAILED (${response})${NC}"
        return 1
    fi
}

echo -e "${YELLOW}=== Testing all list endpoints on ${SERVER_NAME} server (port ${PORT}) ===${NC}"

# Check if server is running
if ! curl -s "http://localhost:${PORT}/health" > /dev/null 2>&1; then
    echo -e "${RED}Server is not running on port ${PORT}${NC}"
    exit 1
fi

echo -e "${GREEN}Server is responding on port ${PORT}${NC}"

# Test all endpoints
passed=0
failed=0
endpoints=(
    "/api/entity/admin/list"
    "/api/entity/client-attribute/list"
    "/api/entity/client/list"
    "/api/entity/delegate-client/list"
    "/api/entity/delegate/list"
    "/api/entity/group/list"
    "/api/entity/location-attribute/list"
    "/api/entity/location/list"
    "/api/entity/manager/list"
    "/api/entity/permission/list"
    "/api/entity/role-permission/list"
    "/api/entity/role/list"
    "/api/entity/staff/list"
    "/api/entity/user/list"
    "/api/entity/workspace-user-role/list"
    "/api/entity/workspace-user/list"
    "/api/entity/workspace/list"
    "/api/event/event/list"
    "/api/framework/framework/list"
    "/api/framework/objective/list"
    "/api/framework/task/list"
    "/api/payment/payment-method/list"
    "/api/payment/payment-profile/list"
    "/api/payment/payment/list"
    "/api/product/collection-plan/list"
    "/api/product/collection/list"
    "/api/product/price-product/list"
    "/api/product/product-attribute/list"
    "/api/product/product-collection/list"
    "/api/product/product-plan/list"
    "/api/product/product/list"
    "/api/product/resource/list"
        "/api/subscription/balance/list"
    "/api/subscription/invoice/list"
    "/api/subscription/plan-settings/list"
    "/api/subscription/plan/list"
    "/api/subscription/price-plan/list"
    "/api/subscription/subscription/list"
)

for endpoint in "${endpoints[@]}"; do
    if test_endpoint "$endpoint"; then
        ((passed++))
    else
        ((failed++))
    fi
done

echo -e "\n${YELLOW}=== ${SERVER_NAME} Summary ===${NC}"
echo -e "${GREEN}Passed: ${passed}${NC}"
echo -e "${RED}Failed: ${failed}${NC}"
echo -e "${CYAN}Total: $((passed + failed))${NC}"

if [[ $failed -eq 0 ]]; then
    echo -e "\n${GREEN}🎉 ALL TESTS PASSED for ${SERVER_NAME} server!${NC}"
    exit 0
else
    echo -e "\n${RED}❌ ${failed} tests failed for ${SERVER_NAME} server${NC}"
    exit 1
fi