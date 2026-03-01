package main

import (
	"fmt"
	"log"
	"os"

	"github.com/erniealice/espyna-golang/consumer"
)

/*
 ESPYNA SERVER - Unified Entry Point

This is the unified entry point for the Espyna HTTP server.
The server framework is selected at compile time via build tags.

Build Tags (one required):
  - gin: Use Gin framework
  - fiber: Use Fiber v2 framework
  - fiber_v3: Use Fiber v3 framework
  - vanilla: Use vanilla net/http

Example Build Commands:
  go run -tags gin,firestore,mock_auth,mock_storage main.go
  go run -tags fiber,postgres,mock_auth,mock_storage main.go
  go run -tags vanilla,mock_db,mock_auth,mock_storage main.go
  go run -tags fiber_v3,firestore,mock_auth,mock_storage main.go

Environment Variables:
  - SERVER_HOST: Server host (default: localhost)
  - SERVER_PORT: Server port (default: 8080)
  - CONFIG_DATABASE_PROVIDER: Database provider (mock_db, postgres, firestore)
  - CONFIG_AUTH_PROVIDER: Auth provider (mock_auth, firebase_auth)
  - CONFIG_ID_PROVIDER: ID provider (noop, google_uuidv7)
  - CONFIG_STORAGE_PROVIDER: Storage provider (mock_storage, local)
  - CONFIG_SERVER_PROVIDER: Server hint (gin, fiber, fiber_v3, vanilla) - for logging only

The actual server implementation is determined by build tags at compile time.
CONFIG_SERVER_PROVIDER is only used for logging/configuration validation.
*/

func main() {
	// Create container from environment variables
	container, err := consumer.NewContainerFromEnv()
	if err != nil {
		log.Fatalf("Failed to create container from environment: %v", err)
	}
	defer container.Close()

	log.Println("SUCCESS: Container initialized")

	// Create server adapter (implementation selected by build tags)
	adapter := consumer.NewServerAdapterFromContainer(container)
	if adapter == nil {
		log.Fatal("Failed to create server adapter. Ensure you compiled with a server build tag (gin, fiber, fiber_v3, or vanilla)")
	}

	// Validate CONFIG_SERVER_PROVIDER matches build tag (if set)
	if configProvider := os.Getenv("CONFIG_SERVER_PROVIDER"); configProvider != "" {
		if configProvider != adapter.Name() {
			log.Printf("WARNING: CONFIG_SERVER_PROVIDER=%s but compiled with %s build tag", configProvider, adapter.Name())
			log.Printf("   Build tags take precedence - using %s server", adapter.Name())
		}
	}

	// Build server address
	host := getEnv("SERVER_HOST", "localhost")
	port := getEnv("SERVER_PORT", "8080")
	addr := fmt.Sprintf("%s:%s", host, port)

	// Start server
	log.Printf("Starting Espyna server (%s) on http://%s", adapter.Name(), addr)
	if err := adapter.Start(addr); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// getEnv returns environment variable value or default if not set
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
