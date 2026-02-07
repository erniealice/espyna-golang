#!/bin/bash

# Run Espyna server with Fiber HTTP framework (with mock data)
# Usage: ./scripts/run-fiber-mock.sh

# Set environment variables
export SERVER_TYPE="fiber"
export SERVER_PORT="8082"
export BUSINESS_TYPE="education"

echo -e "\033[32mStarting Espyna server with Fiber HTTP framework and mock data...\033[0m"
echo -e "\033[36mPort: $SERVER_PORT\033[0m"
echo -e "\033[36mFramework: $SERVER_TYPE\033[0m"
echo -e "\033[36mBusiness Type: $BUSINESS_TYPE\033[0m"
echo ""

go run -tags "fiber,providers_bootstrap,mock_db,mock_email,mock_storage,mock_auth,local_storage,noop,google,firebase,firestore,microsoft,postgres,jwt_auth,postgres_migrations,gcp_storage,s3,gmail,microsoftgraph,aws" ./cmd/server