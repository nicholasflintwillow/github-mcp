# MCP Server Transports

This document describes the available transport mechanisms for the MCP (Model Context Protocol) server, focusing on the Streamable HTTP Transport implementation.

## Streamable HTTP Transport

The Streamable HTTP Transport is a mechanism for efficient and continuous communication between an MCP client and server over HTTP. Unlike traditional request-response models, this transport allows for long-lived connections where data can be streamed in real-time using Server-Sent Events (SSE).

### Key Features

- **Real-time Data Flow:** Enables immediate delivery of data as it becomes available, reducing latency for interactive applications.
- **Bidirectional Communication:** Supports both client-to-server (HTTP POST) and server-to-client (SSE) streaming within standard HTTP protocols.
- **Efficient Resource Utilization:** Minimizes overhead by maintaining persistent connections for server-to-client communication, avoiding the need to establish new connections for each data exchange.
- **Compatibility:** Leverages standard HTTP protocols, making it compatible with existing network infrastructure and proxies.
- **MCP Protocol Compliance:** Maintains full compatibility with the MCP JSON-RPC 2.0 message format.

### Architecture Overview

The Streamable HTTP Transport uses a hybrid approach:

1. **Client-to-Server Communication:** Standard HTTP POST requests to `/mcp/request`
2. **Server-to-Client Communication:** Server-Sent Events (SSE) via `/mcp/stream`

```
Client                    Server
------                    ------
HTTP POST  ────────────►  /mcp/request  (MCP JSON-RPC requests)
           ◄────────────  HTTP Response (MCP JSON-RPC responses)

SSE Client ────────────►  /mcp/stream   (SSE connection)
           ◄────────────  SSE Events    (Streamed MCP messages)
```

### Core Components

#### StreamHandler
- Manages SSE connections to `/mcp/stream` endpoint
- Sets appropriate HTTP headers (`Content-Type: text/event-stream`, `Cache-Control: no-cache`, `Connection: keep-alive`)
- Maintains active client connections and handles connection lifecycle
- Implements heartbeat mechanism to detect disconnections

#### MCPStreamer
- Receives MCP [`JSONRPCMessage`](internal/mcp/protocol.go:1)s from the MCP handler
- Formats messages as SSE events with `event: mcp_message` and JSON data
- Broadcasts messages to all connected SSE clients
- Handles message serialization and error recovery

#### Integration Points
- [`internal/server/handlers.go`](internal/server/handlers.go:1) - HTTP POST handler for `/mcp/request`
- [`internal/mcp/handler.go`](internal/mcp/handler.go:1) - Core MCP message processing
- [`internal/server/server.go`](internal/server/server.go:1) - Server setup and routing

### Communication Flow

#### Client-to-Server (HTTP POST)
1. Client sends MCP JSON-RPC message as HTTP POST to `/mcp/request`
2. Server processes the request through the MCP handler
3. Server returns immediate JSON-RPC response via HTTP response

#### Server-to-Client (SSE)
1. Client establishes SSE connection to `/mcp/stream`
2. Server registers the client connection
3. When the server generates notifications or streamed results, it sends them via [`MCPStreamer`](internal/mcp/mcp_streamer.go:1)
4. Messages are formatted as SSE events: `event: mcp_message\ndata: {json_message}\n\n`
5. Clients receive real-time updates through the SSE connection

### Data Format

#### HTTP POST Requests
Standard MCP JSON-RPC 2.0 messages in request body:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/list",
  "params": null
}
```

#### SSE Events
Server-Sent Events with MCP JSON-RPC messages:
```
event: mcp_message
data: {"jsonrpc":"2.0","method":"notification","params":{...}}

```

### Use Cases

- **Live Updates:** Delivering real-time notifications, progress updates, or streaming responses
- **Long-running Operations:** Monitoring the progress of asynchronous tasks or large data transfers
- **Interactive Sessions:** Powering collaborative tools or real-time MCP interactions
- **Tool Execution Streaming:** Providing real-time feedback during tool execution

This transport is particularly beneficial for scenarios requiring low-latency, high-throughput data exchange, and for applications that need to maintain an open communication channel for extended periods.

## Getting Started

### Server Setup

The Streamable HTTP Transport is automatically enabled when running the MCP server. The server exposes two endpoints:

- `POST /mcp/request` - For client-to-server MCP requests
- `GET /mcp/stream` - For server-to-client SSE connections

Start the server:
```bash
go run cmd/github-mcp/main.go
```

The server will be available at `http://localhost:8080` by default.

### Client Implementation

#### Basic HTTP POST Client
```go
import (
    "bytes"
    "encoding/json"
    "net/http"
)

// Send MCP request
message := map[string]interface{}{
    "jsonrpc": "2.0",
    "id":      1,
    "method":  "tools/list",
    "params":  nil,
}

jsonData, _ := json.Marshal(message)
resp, err := http.Post("http://localhost:8080/mcp/request", 
    "application/json", bytes.NewBuffer(jsonData))
```

#### SSE Client
```go
import (
    "bufio"
    "net/http"
    "strings"
)

// Connect to SSE stream
resp, err := http.Get("http://localhost:8080/mcp/stream")
if err != nil {
    log.Fatal(err)
}
defer resp.Body.Close()

scanner := bufio.NewScanner(resp.Body)
for scanner.Scan() {
    line := scanner.Text()
    if strings.HasPrefix(line, "data: ") {
        data := strings.TrimPrefix(line, "data: ")
        // Parse JSON data as MCP message
        var message map[string]interface{}
        json.Unmarshal([]byte(data), &message)
        // Process MCP message
    }
}
```

#### JavaScript/Browser Client
```javascript
// HTTP POST request
const response = await fetch('http://localhost:8080/mcp/request', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
        jsonrpc: '2.0',
        id: 1,
        method: 'tools/list',
        params: null
    })
});
const result = await response.json();

// SSE connection
const eventSource = new EventSource('http://localhost:8080/mcp/stream');
eventSource.addEventListener('mcp_message', (event) => {
    const message = JSON.parse(event.data);
    console.log('Received MCP message:', message);
});
```

## Testing and Examples

### Test Clients

The project includes comprehensive test clients for validating the Streamable HTTP Transport:

#### Go Client ([`test/client/go_client.go`](test/client/go_client.go:1))
A command-line client that tests both HTTP POST and SSE functionality:

```bash
# Run with default server (http://localhost:8080)
cd test/client
go run go_client.go

# Run with custom server URL
go run go_client.go http://localhost:9090

# Or use the convenience script
./run_go_client.sh
```

**Features:**
- Tests HTTP POST requests to `/mcp/request`
- Establishes SSE connection to `/mcp/stream`
- Sends sample MCP messages (initialize, list tools, list resources, ping)
- Real-time message parsing and display
- Graceful shutdown with Ctrl+C

#### Web Client ([`test/client/web_client.html`](test/client/web_client.html:1))
A browser-based client with interactive interface:

```bash
# Serve via HTTP server (recommended for CORS)
cd test/client
python3 -m http.server 8000
# Open http://localhost:8000/web_client.html
```

**Features:**
- Interactive web interface with real-time status indicators
- SSE connection management (connect/disconnect)
- HTTP POST request testing with custom messages
- Pre-built sample MCP messages
- JSON formatting and validation
- Separate logs for SSE and HTTP requests

### Sample MCP Messages

Both test clients include these sample messages:

#### Initialize Request
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2024-11-05",
    "capabilities": {
      "experimental": {}
    },
    "clientInfo": {
      "name": "test-client",
      "version": "1.0.0"
    }
  }
}
```

#### List Tools Request
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/list",
  "params": null
}
```

#### List Resources Request
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "resources/list",
  "params": null
}
```

### Testing Workflow

1. **Start the MCP server:**
   ```bash
   go run cmd/github-mcp/main.go
   ```

2. **Test with Go client:**
   ```bash
   cd test/client
   ./run_go_client.sh
   ```

3. **Test with Web client:**
   - Open [`test/client/web_client.html`](test/client/web_client.html:1) in a browser
   - Test both SSE and HTTP POST functionality
   - Try different sample messages

4. **Verify functionality:**
   - ✅ HTTP POST requests receive appropriate responses
   - ✅ SSE connection establishes successfully
   - ✅ Streamed messages are received and parsed correctly
   - ✅ Error handling works for invalid requests
   - ✅ Connection management works properly

## Error Handling and Connection Management

### Connection Resilience
- Automatic detection of broken SSE connections
- Heartbeat messages to maintain connection health
- Graceful handling of client disconnections
- Support for connection re-establishment

### Error Scenarios
- **Connection Refused:** Server not running or incorrect URL
- **CORS Errors:** Improper CORS configuration for web clients
- **SSE Connection Fails:** Server-side SSE endpoint issues
- **Invalid JSON:** Malformed MCP messages

### Troubleshooting

#### Common Issues

1. **Connection Refused:**
   - Ensure the MCP server is running
   - Check the server URL and port
   - Verify firewall settings

2. **CORS Errors (Web client):**
   - Serve HTML file via HTTP server instead of opening directly
   - Ensure server has proper CORS headers configured

3. **SSE Connection Fails:**
   - Check server logs for SSE endpoint errors
   - Verify `/mcp/stream` endpoint implementation
   - Test with browser developer tools network tab

4. **Invalid JSON Errors:**
   - Validate JSON syntax before sending
   - Use provided sample message formats
   - Check MCP protocol compliance

#### Roo Client Integration Issues

When integrating with Roo (an AI assistant that uses MCP), several specific issues may occur due to how Roo's internal `streamable-http` client communicates with the server:

##### 1. Initial 404 Error: Client Posting to Root Path

**Problem:** Roo attempts to POST to the root path (`/`) instead of the designated MCP request endpoint (`/mcp/request`).

**Symptoms:**
```
404 Not Found - POST /
```

**Root Cause:** Incorrect `url` configuration in [`.roo/mcp.json`](.roo/mcp.json:1).

**Solution:** Ensure the `url` in your `.roo/mcp.json` configuration points to the base path of the MCP server with a trailing slash:

```json
{
  "mcpServers": {
    "your-server-name": {
      "type": "streamable-http",
      "url": "http://localhost:8080/mcp/",
      "enabled": true
    }
  }
}
```

**Important:** The trailing slash is critical. Roo appends endpoint paths (`request`, `stream`) to this base URL.

##### 2. "Only POST method is allowed for MCP requests" Error

**Problem:** Roo's internal client sends GET requests for initial communication (e.g., for `tools/list` or `resources/list`), while the server's `/mcp/request` endpoint strictly expects POST requests.

**Symptoms:**
```
HTTP 400 - "only POST method is allowed for MCP requests"
```

**Root Cause:** Roo's `streamable-http` client behavior during initial handshake/discovery phase.

**Solution:** This issue has been addressed with a workaround in [`internal/server/handlers.go`](internal/server/handlers.go:138). The server now accepts GET requests to `/mcp/request` and interprets them as implicit `tools/list` requests.

**Workaround Details:**
- GET requests to `/mcp/request` are automatically converted to `tools/list` requests
- This maintains compatibility with Roo's current client behavior
- The workaround is logged with a warning message for monitoring

##### 3. "Failed to parse JSON-RPC message: unexpected end of JSON input" Error

**Problem:** The server attempts to read a request body from a GET request, which typically has no body, resulting in JSON parsing failures.

**Symptoms:**
```
Failed to parse JSON-RPC message: unexpected end of JSON input
```

**Root Cause:** GET requests don't have request bodies, but the server was trying to parse JSON from an empty body.

**Solution:** The workaround in [`handleMCPRequest`](internal/server/handlers.go:138) addresses this by:
1. Detecting GET requests
2. Creating a synthetic [`JSONRPCMessage`](internal/mcp/protocol.go:1) for `tools/list`
3. Processing this synthetic message normally

##### 4. 301 Redirect Issue: Missing Trailing Slash

**Problem:** Missing trailing slash in the `url` configuration causes unnecessary 301 redirects that can interfere with client communication.

**Symptoms:**
```
HTTP 301 Moved Permanently
Location: http://localhost:8080/mcp/
```

**Root Cause:** Web servers typically redirect URLs without trailing slashes to include them when the path represents a directory.

**Solution:** Always include the trailing slash in your `.roo/mcp.json` configuration:

```json
// ❌ Incorrect - causes 301 redirect
"url": "http://localhost:8080/mcp"

// ✅ Correct - no redirect needed
"url": "http://localhost:8080/mcp/"
```

##### 5. Configuration Best Practices for Roo Integration

**Complete `.roo/mcp.json` Example:**
```json
{
  "mcpServers": {
    "github-mcp": {
      "type": "streamable-http",
      "url": "http://localhost:8080/mcp/",
      "enabled": true
    }
  }
}
```

**Key Points:**
- Use `"type": "streamable-http"` for this transport
- Include trailing slash in the `url`
- Ensure the server is running before enabling the client
- Monitor server logs for connection and request patterns

##### 6. Debugging Roo Integration

**Server-side Monitoring:**
Monitor your server logs for these patterns:
```
INFO  MCP request received method=GET path=/mcp/request
WARN  Received GET request for MCP endpoint; assuming tools/list or resources/list
INFO  MCP request received method=POST path=/mcp/request
```

**Client-side Verification:**
1. Verify Roo can connect: Check for successful `initialize` requests
2. Test tool listing: Ensure `tools/list` requests work
3. Test actual tool calls: Verify POST requests for tool execution

**Common Resolution Steps:**
1. Check `.roo/mcp.json` configuration syntax and URL format
2. Verify server is running and accessible at the configured URL
3. Review server logs for specific error messages
4. Test with manual HTTP requests to isolate client vs. server issues

##### 7. Known Limitations and Workarounds

**Current Workaround Status:**
- ✅ GET request handling: Implemented in [`handleMCPRequest`](internal/server/handlers.go:138)
- ✅ Trailing slash handling: Documented configuration requirement
- ✅ JSON parsing: Handled via synthetic message creation

**Future Considerations:**
- The GET request workaround is specific to Roo's current client behavior
- Monitor for updates to Roo's `streamable-http` client implementation
- Consider removing workaround if Roo's client behavior changes

### Monitoring

Monitor server logs to observe:
- Incoming HTTP POST requests
- SSE connection establishments and disconnections
- Message processing results
- Error conditions and warnings

## Implementation Details

For detailed architectural information, see [`docs/streamable-http-transport-design.md`](docs/streamable-http-transport-design.md:1).

Key implementation files:
- [`internal/mcp/stream_handler.go`](internal/mcp/stream_handler.go:1) - SSE connection management
- [`internal/mcp/mcp_streamer.go`](internal/mcp/mcp_streamer.go:1) - Message streaming logic
- [`internal/server/handlers.go`](internal/server/handlers.go:1) - HTTP request handling
- [`test/streamable_transport_integration_test.go`](test/streamable_transport_integration_test.go:1) - Integration tests

Unit tests are available in:
- [`internal/mcp/stream_handler_test.go`](internal/mcp/stream_handler_test.go:1)
- [`internal/mcp/mcp_streamer_test.go`](internal/mcp/mcp_streamer_test.go:1)