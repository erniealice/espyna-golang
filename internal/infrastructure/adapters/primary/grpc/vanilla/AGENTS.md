# gRPC Vanilla Adapter - Agent Usage Guide

**Package:** `leapfor.xyz/espyna/internal/infrastructure/adapters/primary/grpc/vanilla`

This guide explains how to use the gRPC Vanilla adapter for AI agents and developers.

---

## Overview

The gRPC Vanilla adapter provides a **gRPC transport layer** for the Espyna business management API. It mirrors all HTTP routes as gRPC methods, enabling:

- High-performance protobuf serialization
- Strongly-typed service contracts
- Bi-directional streaming support
- Multi-language client generation

---

## Architecture

### HTTP to gRPC Method Mapping

```
HTTP Route:                           gRPC Full Method:
POST /api/entity/client/create     -> /espyna.entity.v1.ClientService/Create
POST /api/entity/client/list       -> /espyna.entity.v1.ClientService/List
POST /api/entity/client/read       -> /espyna.entity.v1.ClientService/Read
POST /api/entity/client/update     -> /espyna.entity.v1.ClientService/Update
POST /api/entity/client/delete     -> /espyna.entity.v1.ClientService/Delete
POST /api/subscription/plan/read   -> /espyna.subscription.v1.PlanService/Read
```

**Mapping Formula:**
```
/espyna.{domain}.v1.{Resource}Service/{Operation}
```

| Component | Source | Example |
|-----------|--------|---------|
| Domain | `route.Metadata.Domain` | `entity`, `subscription` |
| Resource | `route.Metadata.Resource` | `Client`, `Plan` |
| Operation | `route.Metadata.Operation` | `Create`, `List`, `Read` |

---

## Quick Start

### 1. Build with gRPC Support

```bash
# Build with grpc_vanilla build tag
go build -tags grpc_vanilla,firestore,mock_auth,mock_storage,google_uuidv7 ./cmd/server

# Or for development
go build -tags grpc_vanilla,mock_db,mock_auth,mock_storage ./cmd/server
```

### 2. Set Environment Variables

```bash
# Select gRPC provider
export CONFIG_SERVER_PROVIDER=grpc_vanilla

# Server port (default: 50051 for gRPC)
export SERVER_PORT=50051

# Optional: Enable/disable reflection
export GRPC_REFLECTION_ENABLED=true

# Other providers (same as HTTP)
export CONFIG_DATABASE_PROVIDER=firestore
export CONFIG_AUTH_PROVIDER=firebase_auth
```

### 3. Run the Server

```bash
./server

# Output:
#   Espyna Server
#   Framework: grpc_vanilla
#   Address: :50051
#   Database: firestore
#   Auth: firebase_auth
#   gRPC reflection enabled
# gRPC server starting on :50051
```

---

## Testing with grpcurl

### List All Services

```bash
grpcurl -plaintext localhost:50051 list

# Output:
# grpc.health.v1.Health
# espyna.entity.v1.ClientService
# espyna.subscription.v1.PlanService
# espyna.workflow.v1.WorkflowService
# ...
```

### Health Check

```bash
grpcurl -plaintext localhost:50051 grpc.health.v1.Health/Check

# Output:
# {
#   "status": "SERVING"
# }
```

### Call a Method (No Auth)

```bash
grpcurl -plaintext \
  -d '{"workspace_id": "test-workspace"}' \
  localhost:50051 \
  espyna.entity.v1.ClientService/List
```

### Call with Authentication

```bash
# Using Bearer token
grpcurl -plaintext \
  -H "authorization: Bearer <firebase-token>" \
  -d '{"workspace_id": "workspace123"}' \
  localhost:50051 \
  espyna.entity.v1.ClientService/List

# Using API Key
grpcurl -plaintext \
  -H "x-api-key: your-api-key" \
  -d '{"workspace_id": "workspace123"}' \
  localhost:50051 \
  espyna.entity.v1.ClientService/Create
```

### Describe a Service

```bash
grpcurl -plaintext describe espyna.entity.v1.ClientService localhost:50051
```

---

## Client Code Generation

### Go Client

```bash
# Generate Go client from protobuf
protoc --go_out=. --go-grpc_out=. espyna.proto

# Or use grpcurl to extract service descriptor
grpcurl -plaintext localhost:50051 describe espyna.entity.v1.ClientService > client_service.desc
```

### Python Client

```python
import grpc
from espyna_pb2 import ClientCreateRequest, ClientListRequest
from espyna_pb2_grpc import ClientServiceStub

# Create channel
channel = grpc.insecure_channel('localhost:50051')
stub = ClientServiceStub(channel)

# List clients
request = ClientListRequest(workspace_id="workspace123")
response = stub.List(request)

for client in response.clients:
    print(f"Client: {client.id} - {client.name}")
```

### TypeScript/JavaScript Client

```typescript
import * as grpc from '@grpc/grpc-js';
import * as protoLoader from '@grpc/proto-loader';

const packageDefinition = protoLoader.loadSync('espyna.proto', {
  keepCase: true,
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true
});

const espyna = grpc.loadPackageDefinition(packageDefinition).espyna.entity.v1;
const client = new espyna.ClientService(
  'localhost:50051',
  grpc.credentials.createInsecure()
);

// List clients
client.List({ workspace_id: 'workspace123' }, (err, response) => {
  if (err) console.error(err);
  console.log(response.clients);
});
```

---

## Authentication

### Supported Methods

| Method | Header | Value |
|--------|--------|-------|
| Bearer Token | `authorization` | `Bearer <firebase-token>` |
| API Key | `x-api-key` | `<your-api-key>` |
| Scheduler Key | `x-api-key-scheduler` | `<scheduler-key>` |

### Environment Variables

```bash
# API Key bypasses auth validation
export X_API_KEY=your-secret-api-key

# Scheduler key for workflow automation
export X_API_KEY_SCHEDULER=your-scheduler-key
```

### Public Methods (No Auth)

- `/grpc.health.v1.Health/Check`
- `/grpc.health.v1.Health/Watch`

---

## Interceptor Chain

The gRPC server uses an interceptor chain that processes requests in order:

```
Request → Recovery → Logging → Authentication → Handler → Response
```

### 1. Recovery Interceptor
- Catches panics and converts to gRPC errors
- Logs stack traces for debugging
- Returns `codes.Internal` on panic

### 2. Logging Interceptor
- Logs all incoming requests with method name
- Logs response status and duration
- Format: `GRPC_REQUEST: method=/espyna.entity.v1.ClientService/Create`

### 3. Authentication Interceptor
- Validates Bearer tokens via AuthService
- Checks API keys
- Enriches context with `uid`, `email`, `identity`
- Skips public methods

---

## Context Values

After authentication, the following values are available in the gRPC context:

| Key | Type | Description |
|-----|------|-------------|
| `uid` | `string` | User ID from auth token |
| `email` | `string` | User email |
| `identity` | `*authpb.Identity` | Full identity object |
| `workspace_id` | `string` | Workspace from header/metadata |
| `expires` | `int64` | Token expiration timestamp |

---

## Error Codes

| Condition | gRPC Code |
|-----------|-----------|
| Missing auth | `codes.Unauthenticated` |
| Invalid token | `codes.Unauthenticated` |
| No permission | `codes.PermissionDenied` |
| Method not found | `codes.Unimplemented` |
| Handler error | `codes.Internal` |
| Invalid request | `codes.InvalidArgument` |
| Not found | `codes.NotFound` |
| Already exists | `codes.AlreadyExists` |

---

## Environment Variables Reference

| Variable | Default | Description |
|----------|---------|-------------|
| `CONFIG_SERVER_PROVIDER` | `vanilla` | Set to `grpc_vanilla` or `grpc` |
| `SERVER_PORT` | `8080` | Use `50051` for gRPC (standard port) |
| `GRPC_REFLECTION_ENABLED` | `true` | Enable gRPC server reflection |
| `X_API_KEY` | - | API key for auth bypass |
| `X_API_KEY_SCHEDULER` | - | Scheduler API key |

---

## Build Tags

| Build Tag | Description |
|-----------|-------------|
| `grpc_vanilla` | Enables gRPC server adapter |

**Full build command:**
```bash
go build -tags "grpc_vanilla,firestore,firebase_auth,gcs,google_uuidv7,gmail" ./cmd/server
```

---

## Inter-Service Communication

### Service A calling Service B

```go
import (
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

// Create connection
conn, err := grpc.Dial("service-b:50051",
    grpc.WithTransportCredentials(insecure.NewCredentials()),
    grpc.WithBlock(),
)

// Create client
client := pb.NewEspynaServiceClient(conn)

// Call method
req := &pb.ClientListRequest{
    WorkspaceId: "workspace123",
}
resp, err := client.List(ctx, req)
```

---

## Production Checklist

- [ ] Enable TLS/SSL for production (use `grpc.WithTransportCredentials`)
- [ ] Set up proper authentication (Firebase Auth or JWT)
- [ ] Configure rate limiting
- [ ] Enable monitoring and observability
- [ ] Set up gRPC gateway for HTTP fallback (optional)
- [ ] Generate and distribute client stubs
- [ ] Document service versions and compatibility

---

## Troubleshooting

### Server not starting

```bash
# Check if port is in use
netstat -an | grep 50051

# Verify build tag
go list -f '{{.BuildTags}}' ./...

# Check provider registration
# (Look for "gRPC Vanilla adapter initialized successfully" in logs)
```

### "method not found" error

- Verify the method name format: `/espyna.{domain}.v1.{Resource}Service/{Operation}`
- Check if the route exists in HTTP (`GET /api/entity/client/list`)
- Ensure the route metadata is complete (domain, resource, operation)

### Connection refused

```bash
# Test connectivity
telnet localhost 50051

# Check firewall rules
sudo ufw allow 50051/tcp
```

---

## See Also

- **Plan Document:** `packages/espyna/docs/plan/20251223-server-adapter/plan-grpc-extend.md`
- **HTTP Adapters:** `../http/` for comparison
- **Interceptors:** `../interceptors/` for auth/recovery/logging
- **Registry:** `internal/infrastructure/registry/server.go`
