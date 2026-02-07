# Espyna API

Simple Go API server that demonstrates accessing shared mock data from the `@leapfor/copya` package.

## Overview

Espyna is a lightweight HTTP API built with Go 1.24 that provides RESTful endpoints for accessing mock event data. It serves as a demonstration of cross-language data sharing using the centralized copya package.

## Features

- ✅ RESTful HTTP endpoints using Go 1.24 standard library
- ✅ Integration with shared `@leapfor/copya` package
- ✅ JSON response format with consistent API structure
- ✅ Basic CORS middleware for frontend integration
- ✅ Health check endpoint for monitoring
- ✅ Business type filtering support

## API Endpoints

```
GET /api/health              # Health check
GET /api/events              # List all events (default: education)
GET /api/events/{id}         # Get specific event by ID
GET /api/events?type={type}  # Filter events by business type
```

### Supported Business Types
- `education` (default)
- `fitness_center`
- `aesthetic_clinic` 
- `business`

## Usage

### Start the server
```bash
cd packages/espyna
go run main.go
```

### Example requests
```bash
# Health check
curl http://localhost:8080/api/health

# List all education events
curl http://localhost:8080/api/events

# List fitness center events
curl http://localhost:8080/api/events?type=fitness_center

# Get specific event
curl http://localhost:8080/api/events/event-001
```

### Example response
```json
{
  "success": true,
  "data": [
    {
      "id": "event-001",
      "name": "Math Olympiad Competition",
      "description": "Regional mathematics competition for advanced students",
      "startDateTimeUtc": "1739631600000",
      "endDateTimeUtc": "1739653200000",
      "timezone": "America/New_York",
      "active": true,
      "dateCreated": "1732752000000",
      "dateCreatedString": "2024-12-08",
      "dateModified": "1737586800000",
      "dateModifiedString": "2025-01-22"
    }
  ],
  "message": "Found 5 events for business type 'education'"
}
```

## Configuration

The server can be configured using environment variables:

- `PORT` - Server port (default: 8080)

## Dependencies

- Go 1.24 standard library only
- `@leapfor/copya` package for shared mock data
- `@leapfor/protobuf-models-go` package for type definitions (future use)

## Development

This is a simple demonstration API. Features that could be added:
- Database integration
- Authentication and authorization
- Request validation
- Logging middleware
- Rate limiting
- OpenAPI/Swagger documentation