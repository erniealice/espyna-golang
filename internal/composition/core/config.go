package core

/*
═ ESPYNA SERVER CONFIGURATION ═

This configuration system uses environment variables to dynamically configure
different providers and services. The main entry point is NewContainerFromEnv()
in container.go.

═══════════════════════════════════════════════════════════════════════════
🔧 DATABASE PROVIDERS:
═══════════════════════════════════════════════════════════════════════════

CONFIG_DATABASE_PROVIDER=mock_db          (default) Mock database for development
CONFIG_DATABASE_PROVIDER=postgresql       PostgreSQL database
CONFIG_DATABASE_PROVIDER=firestore        Google Cloud Firestore

🧪 MOCK DATABASE (mock_db):
BUSINESS_TYPE=education                   Business type for mock data (default: education)

🐘 POSTGRESQL (postgresql):
DATABASE_POSTGRES_HOST=localhost          Database host (default: localhost)
DATABASE_POSTGRES_PORT=5432               Database port (default: 5432)
DATABASE_POSTGRES_DBNAME=espyna           Database name (default: espyna)
DATABASE_POSTGRES_USER=postgres           Database user (default: postgres)
DATABASE_POSTGRES_PASSWORD=               Database password (required)
DATABASE_POSTGRES_URL=                    Full connection URL (optional)
DATABASE_POSTGRES_SSLMODE=disable         SSL mode (default: disable)

🔥 FIRESTORE (firestore):
DATABASE_FIRESTORE_PROJECT_ID=your-project-id      Google Cloud project ID (required)
DATABASE_FIRESTORE_CREDENTIALS_FILE=/path/to/creds.json Service account credentials file path
DATABASE_FIRESTORE_DATABASE=              Firestore database name (optional)

═══════════════════════════════════════════════════════════════════════════
🔐 AUTHENTICATION PROVIDERS:
═══════════════════════════════════════════════════════════════════════════

CONFIG_AUTH_PROVIDER=mock                 (default) Mock authentication for development
CONFIG_AUTH_PROVIDER=password             Password + session auth (any DB backend via DatabaseOperation)
CONFIG_AUTH_PROVIDER=firebase             Firebase Authentication

🔥 FIREBASE AUTH (firebase):
FIREBASE_AUTH_PROJECT_ID=your-project-id      Google Cloud project ID (required)
FIREBASE_AUTH_CREDENTIALS_PATH=/path/to/creds.json Service account credentials file path
FIREBASE_AUTH_TENANT_ID=                     Firebase Auth tenant ID (optional)

═══════════════════════════════════════════════════════════════════════════
🆔 ID PROVIDERS:
═══════════════════════════════════════════════════════════════════════════

CONFIG_ID_PROVIDER=noop                   (default) NoOp ID service (timestamp-based)
CONFIG_ID_PROVIDER=google_uuidv7          Google UUID v7 (time-ordered, globally unique)

Note: google_uuidv7 requires build tag: -tags google_uuidv7

═══════════════════════════════════════════════════════════════════════════
📁 STORAGE PROVIDERS:
═══════════════════════════════════════════════════════════════════════════

CONFIG_STORAGE_PROVIDER=mock_storage      (default) Mock storage for development
CONFIG_STORAGE_PROVIDER=local             Local file system storage

🏠 LOCAL STORAGE (local):
STORAGE_LOCAL_BASE_PATH=./storage         Base path for local storage (default: ./storage)

═══════════════════════════════════════════════════════════════════════════
🖥️  SERVER CONFIGURATION:
═══════════════════════════════════════════════════════════════════════════

SERVER_HOST=localhost                     Server host (default: localhost)
SERVER_PORT=8080                          Server port (default: 8080)

═══════════════════════════════════════════════════════════════════════════
📊 TABLE/COLLECTION NAMES:
═══════════════════════════════════════════════════════════════════════════

You can customize table/collection names using environment variables:
For PostgreSQL: LEAPFOR_DATABASE_POSTGRES_TABLE_<ENTITY>=<table_name>
For Firestore: LEAPFOR_DATABASE_FIRESTORE_COLLECTION_<ENTITY>=<collection_name>

Example entities: CLIENT, MANAGER, SUBSCRIPTION, PAYMENT, PRODUCT, etc.

═══════════════════════════════════════════════════════════════════════════
🚀 EXAMPLE USAGE:
═══════════════════════════════════════════════════════════════════════════

# Development with mock providers:
export CONFIG_DATABASE_PROVIDER=mock_db
export CONFIG_AUTH_PROVIDER=mock
export BUSINESS_TYPE=education

# Production with PostgreSQL and Firebase Auth:
export CONFIG_DATABASE_PROVIDER=postgresql
export CONFIG_AUTH_PROVIDER=firebase_auth
export DATABASE_POSTGRES_HOST=your-db-host
export DATABASE_POSTGRES_PASSWORD=your-password
export FIREBASE_AUTH_PROJECT_ID=your-project

# Production with Firestore and Firebase Auth:
export CONFIG_DATABASE_PROVIDER=firestore
export CONFIG_AUTH_PROVIDER=firebase_auth
export DATABASE_FIRESTORE_PROJECT_ID=your-project
export DATABASE_FIRESTORE_CREDENTIALS_FILE=/path/to/creds.json

═══════════════════════════════════════════════════════════════════════════
*/
