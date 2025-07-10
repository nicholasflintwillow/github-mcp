# MCP Streamable HTTP Transport Test Clients

This directory contains client-side components for testing the Streamable HTTP Transport implementation. The clients can test both the SSE streaming endpoint (`/mcp/stream`) and the HTTP POST request endpoint (`/mcp/request`).

## Available Clients

### 1. Go Client (`go_client.go`)

A command-line Go program that tests both endpoints with sample MCP messages.

#### Features:
- ✅ HTTP POST requests to `/mcp/request` endpoint
- ✅ SSE connection to `/mcp/stream` endpoint  
- ✅ Sample MCP messages (initialize, list tools, list resources, ping, initialized)
- ✅ Real-time message parsing and display
- ✅ Graceful shutdown with Ctrl+C

#### Usage:

```bash
# Run with default server (http://localhost:8080)
cd test/client
go run go_client.go

# Run with custom server URL
go run go_client.go http://localhost:9090
```

#### Sample Output:
```
🚀 MCP Streamable HTTP Transport Test Client
🌐 Server URL: http://localhost:8080
==================================================

🧪 Testing HTTP POST requests...

--- Test 1 ---
📤 Sending HTTP POST request to: http://localhost:8080/mcp/request
   📋 Message: {"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05",...}}
📥 Received HTTP response (status 200): {"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05",...}}
   ✅ Parsed MCP response - ID: 1, Error: false
✅ Success: Received response

🧪 Testing SSE connection...
Press Ctrl+C to stop...
📡 Connecting to SSE stream at: http://localhost:8080/mcp/stream
✅ SSE connection established successfully
📡 Listening for streamed messages...
📨 Received SSE message: {"jsonrpc":"2.0","method":"notification","params":{...}}
   📋 Parsed MCP message - Method: notification, ID: <nil>
---
```

### 2. Web Client (`web_client.html`)

A browser-based HTML/JavaScript client with a user-friendly interface.

#### Features:
- ✅ Interactive web interface
- ✅ Real-time SSE connection with status indicators
- ✅ HTTP POST request testing with custom messages
- ✅ Pre-built sample MCP messages
- ✅ JSON formatting and validation
- ✅ Separate logs for SSE and HTTP requests
- ✅ Configurable server URL

#### Usage:

1. **Start the MCP server** (ensure it's running on the configured port)

2. **Open the web client** in a browser:
   ```bash
   # Option 1: Open directly in browser
   open test/client/web_client.html
   
   # Option 2: Serve via HTTP server (recommended for CORS)
   cd test/client
   python3 -m http.server 8000
   # Then open http://localhost:8000/web_client.html
   ```

3. **Configure server URL** (default: http://localhost:8080)

4. **Test SSE connection:**
   - Click "Connect to Stream"
   - Monitor the connection status and incoming messages
   - Click "Disconnect" to close the connection

5. **Test HTTP requests:**
   - Use sample messages or create custom JSON-RPC messages
   - Click "Send Request" to test the `/mcp/request` endpoint
   - View responses in the request log

#### Interface Sections:

- **Server Configuration**: Set the MCP server URL
- **SSE Stream Connection**: Connect/disconnect from the stream endpoint
- **HTTP POST Requests**: Send custom or sample MCP messages

## Sample MCP Messages

Both clients include these sample messages for testing:

### 1. Initialize Request
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

### 2. List Tools Request
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/list",
  "params": null
}
```

### 3. List Resources Request
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "resources/list",
  "params": null
}
```

### 4. Ping Request
```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "ping",
  "params": null
}
```

### 5. Initialized Notification
```json
{
  "jsonrpc": "2.0",
  "method": "initialized",
  "params": null
}
```

## Testing Workflow

### Prerequisites
1. Ensure the MCP server is running with the Streamable HTTP Transport enabled
2. Verify the server exposes both `/mcp/request` and `/mcp/stream` endpoints
3. Check that CORS is properly configured for web client testing

### Recommended Testing Steps

1. **Start the MCP server**:
   ```bash
   cd /path/to/github-mcp
   go run cmd/github-mcp/main.go
   ```

2. **Test with Go client** (command-line testing):
   ```bash
   cd test/client
   go run go_client.go
   ```

3. **Test with Web client** (browser-based testing):
   - Open `web_client.html` in a browser
   - Test both SSE and HTTP POST functionality
   - Try different sample messages

4. **Verify functionality**:
   - ✅ HTTP POST requests receive appropriate responses
   - ✅ SSE connection establishes successfully
   - ✅ Streamed messages are received and parsed correctly
   - ✅ Error handling works for invalid requests
   - ✅ Connection management (connect/disconnect) works properly

## Troubleshooting

### Common Issues

1. **Connection Refused**:
   - Ensure the MCP server is running
   - Check the server URL and port
   - Verify firewall settings

2. **CORS Errors** (Web client):
   - Serve the HTML file via HTTP server instead of opening directly
   - Ensure server has proper CORS headers configured

3. **SSE Connection Fails**:
   - Check server logs for SSE endpoint errors
   - Verify the `/mcp/stream` endpoint is properly implemented
   - Test with browser developer tools network tab

4. **Invalid JSON Errors**:
   - Use the "Format JSON" button in web client
   - Validate JSON syntax before sending
   - Check sample message formats

### Server Logs
Monitor the MCP server logs to see:
- Incoming HTTP POST requests
- SSE connection establishments
- Message processing results
- Any errors or warnings

## Architecture

These test clients implement the client-side of the Streamable HTTP Transport architecture:

```
Client                    Server
------                    ------
HTTP POST  ────────────►  /mcp/request  (Standard MCP requests)
           ◄────────────  HTTP Response (MCP responses)

SSE Client ────────────►  /mcp/stream   (SSE connection)
           ◄────────────  SSE Events    (Streamed MCP messages)
```

The clients demonstrate:
- **Bidirectional communication**: HTTP POST for client→server, SSE for server→client
- **Standard MCP protocol**: JSON-RPC 2.0 message format
- **Real-time streaming**: Server-sent events for live updates
- **Error handling**: Proper handling of connection failures and invalid messages