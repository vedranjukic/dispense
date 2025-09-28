# Dispense REST API Documentation

This document describes the REST API endpoints provided by the gRPC-Gateway for the Dispense service.

## Base URL

```
http://localhost:8081/v1
```

## Authentication

All endpoints (except config endpoints) require authentication via API key:

### Methods:
1. **Header**: `X-API-Key: your-api-key`
2. **Bearer Token**: `Authorization: Bearer your-api-key`
3. **Query Parameter**: `?api_key=your-api-key`

## Endpoints

### Sandbox Management

#### Create Sandbox
**POST** `/v1/sandboxes`

Create a new sandbox (local or remote).

```bash
curl -X POST http://localhost:8081/v1/sandboxes \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{
    "name": "my-sandbox",
    "is_remote": false,
    "force": true
  }'
```

**Request Body:**
```json
{
  "name": "string",
  "branch_name": "string",
  "is_remote": false,
  "force": false,
  "skip_copy": false,
  "skip_daemon": false,
  "group": "string",
  "model": "string",
  "task": "string",
  "resources": {
    "snapshot": "string",
    "target": "string",
    "cpu": 0,
    "memory": 0,
    "disk": 0,
    "auto_stop": 0
  },
  "source_directory": "string",
  "task_data": {
    "description": "string",
    "github_issue": {
      "url": "string",
      "number": 0,
      "owner": "string",
      "repo": "string",
      "title": "string",
      "body": "string"
    }
  }
}
```

#### List Sandboxes
**GET** `/v1/sandboxes`

List all sandboxes with optional filtering.

```bash
curl -X GET "http://localhost:8081/v1/sandboxes?show_local=true&show_remote=true" \
  -H "X-API-Key: your-api-key"
```

**Query Parameters:**
- `show_local` (boolean): Show local sandboxes
- `show_remote` (boolean): Show remote sandboxes
- `verbose` (boolean): Verbose output
- `group` (string): Filter by group

#### Get Sandbox
**GET** `/v1/sandboxes/{identifier}`

Get details of a specific sandbox.

```bash
curl -X GET http://localhost:8081/v1/sandboxes/my-sandbox \
  -H "X-API-Key: your-api-key"
```

#### Delete Sandbox
**DELETE** `/v1/sandboxes/{identifier}`

Delete a specific sandbox.

```bash
curl -X DELETE http://localhost:8081/v1/sandboxes/my-sandbox \
  -H "X-API-Key: your-api-key"
```

**Query Parameters:**
- `delete_all` (boolean): Delete all sandboxes
- `force` (boolean): Force deletion

#### Wait for Sandbox
**POST** `/v1/sandboxes/{identifier}/wait`

Wait for sandbox to be ready.

```bash
curl -X POST http://localhost:8081/v1/sandboxes/my-sandbox/wait \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{
    "timeout_seconds": 300
  }'
```

### Claude Operations

#### Run Claude Task
**POST** `/v1/claude/tasks`

Execute a task using Claude in a sandbox. **Note**: This endpoint uses Server-Sent Events (SSE) for streaming responses.

```bash
curl -X POST http://localhost:8081/v1/claude/tasks \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -H "Accept: text/event-stream" \
  -d '{
    "sandbox_identifier": "my-sandbox",
    "task_description": "List all files in the current directory",
    "model": "claude-3-sonnet"
  }'
```

**Request Body:**
```json
{
  "sandbox_identifier": "string",
  "task_description": "string",
  "model": "string"
}
```

**Response** (Server-Sent Events):
```
data: {"type": "STDOUT", "content": "output text", "timestamp": 1234567890}
data: {"type": "STATUS", "exit_code": 0, "is_finished": true}
```

#### Get Claude Status
**GET** `/v1/claude/{sandbox_identifier}/status`

Get the status of Claude daemon in a sandbox.

```bash
curl -X GET http://localhost:8081/v1/claude/my-sandbox/status \
  -H "X-API-Key: your-api-key"
```

#### Get Claude Logs
**GET** `/v1/claude/{sandbox_identifier}/logs`

Get Claude logs from a sandbox.

```bash
curl -X GET "http://localhost:8081/v1/claude/my-sandbox/logs?task_id=task123" \
  -H "X-API-Key: your-api-key"
```

**Query Parameters:**
- `task_id` (string, optional): Specific task ID to get logs for

### Configuration Management

#### Get API Key
**GET** `/v1/config/api-key`

Get the current API key (no authentication required).

```bash
curl -X GET "http://localhost:8081/v1/config/api-key?interactive=false"
```

**Query Parameters:**
- `interactive` (boolean): Whether to prompt if not found

#### Set API Key
**POST** `/v1/config/api-key`

Set a new API key (no authentication required).

```bash
curl -X POST http://localhost:8081/v1/config/api-key \
  -H "Content-Type: application/json" \
  -d '{
    "api_key": "your-new-api-key"
  }'
```

#### Validate API Key
**POST** `/v1/config/api-key/validate`

Validate an API key (no authentication required).

```bash
curl -X POST http://localhost:8081/v1/config/api-key/validate \
  -H "Content-Type: application/json" \
  -d '{
    "api_key": "key-to-validate"
  }'
```

## Health Checks

#### Health Check
**GET** `/health` or `/healthz`

Check if the service is healthy.

```bash
curl -X GET http://localhost:8081/health
```

**Response:**
```json
{
  "status": "healthy",
  "service": "dispense-gateway"
}
```

#### Readiness Check
**GET** `/ready`

Check if the service is ready to serve requests.

```bash
curl -X GET http://localhost:8081/ready
```

## Error Handling

The API returns standard HTTP status codes and JSON error responses:

```json
{
  "error": {
    "code": "VALIDATION_FAILED",
    "message": "name or branch_name must be provided",
    "details": {}
  }
}
```

### Common Status Codes:
- `200` - Success
- `400` - Bad Request (validation failed)
- `401` - Unauthorized (invalid API key)
- `404` - Not Found (resource not found)
- `429` - Too Many Requests (rate limited)
- `500` - Internal Server Error

## CORS

The API includes CORS headers for cross-origin requests:
- `Access-Control-Allow-Origin: *`
- `Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS`
- `Access-Control-Allow-Headers: Content-Type, Authorization, X-API-Key`

## Rate Limiting

The API includes rate limiting:
- **Limit**: 100 requests per minute per IP
- **Response**: HTTP 429 with retry information

## Examples

### Complete Workflow Example

```bash
#!/bin/bash

API_KEY="your-api-key"
BASE_URL="http://localhost:8081/v1"

# 1. Create a sandbox
SANDBOX_RESPONSE=$(curl -s -X POST "$BASE_URL/sandboxes" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d '{
    "name": "demo-sandbox",
    "is_remote": false
  }')

echo "Created sandbox: $SANDBOX_RESPONSE"

# 2. List sandboxes
curl -s -X GET "$BASE_URL/sandboxes" \
  -H "X-API-Key: $API_KEY" | jq

# 3. Check Claude status
curl -s -X GET "$BASE_URL/claude/demo-sandbox/status" \
  -H "X-API-Key: $API_KEY" | jq

# 4. Run a Claude task
curl -X POST "$BASE_URL/claude/tasks" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -H "Accept: text/event-stream" \
  -d '{
    "sandbox_identifier": "demo-sandbox",
    "task_description": "Create a simple hello world Python script"
  }'

# 5. Clean up - delete sandbox
curl -s -X DELETE "$BASE_URL/sandboxes/demo-sandbox" \
  -H "X-API-Key: $API_KEY" | jq
```

### JavaScript/Fetch Example

```javascript
const BASE_URL = 'http://localhost:8081/v1';
const API_KEY = 'your-api-key';

// Create sandbox
async function createSandbox() {
  const response = await fetch(`${BASE_URL}/sandboxes`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-API-Key': API_KEY
    },
    body: JSON.stringify({
      name: 'js-sandbox',
      is_remote: false
    })
  });

  return response.json();
}

// Run Claude task with streaming
async function runClaudeTask(sandboxId, task) {
  const response = await fetch(`${BASE_URL}/claude/tasks`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-API-Key': API_KEY,
      'Accept': 'text/event-stream'
    },
    body: JSON.stringify({
      sandbox_identifier: sandboxId,
      task_description: task
    })
  });

  const reader = response.body.getReader();
  const decoder = new TextDecoder();

  while (true) {
    const { value, done } = await reader.read();
    if (done) break;

    const chunk = decoder.decode(value);
    const lines = chunk.split('\n');

    for (const line of lines) {
      if (line.startsWith('data: ')) {
        const data = JSON.parse(line.slice(6));
        console.log('Task output:', data);

        if (data.is_finished) {
          return;
        }
      }
    }
  }
}
```

## OpenAPI/Swagger

For interactive API documentation, the service can be extended to serve an OpenAPI/Swagger UI. The proto files can be converted to OpenAPI specification for better documentation and testing.