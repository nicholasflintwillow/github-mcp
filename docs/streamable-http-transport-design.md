# Architectural Design for Streamable HTTP Transport (Server-Side)

The Streamable HTTP Transport will leverage Server-Sent Events (SSE) to provide a continuous, unidirectional data stream from the server to the client, and standard HTTP POST requests for client-to-server communication. This approach maintains compatibility with standard HTTP protocols while enabling real-time server-to-client updates.

## 1. Core Components:

*   **`StreamHandler` (New):** A new HTTP handler responsible for establishing and managing SSE connections.
    *   It will set appropriate HTTP headers (`Content-Type: text/event-stream`, `Cache-Control: no-cache`, `Connection: keep-alive`) to enable SSE.
    *   It will keep the HTTP connection open and continuously send data to the client.
    *   It will manage a list of active client connections (e.g., a map of client IDs to `http.ResponseWriter` and `chan []byte`).
*   **`MCPStreamer` (New):** A component responsible for pushing MCP messages to connected clients.
    *   It will receive MCP `JSONRPCMessage`s (e.g., notifications, tool results) from the `mcp.Handler`.
    *   It will format these messages as SSE events and send them to all subscribed clients via their respective `http.ResponseWriter`s.
    *   It will handle serialization of `JSONRPCMessage`s into a format suitable for SSE (e.g., JSON string within an SSE `data` field).
*   **Existing `mcp.Handler`:** This component will remain largely unchanged in its core logic for processing MCP requests and generating responses/notifications.
    *   When the `mcp.Handler` generates a notification or a response that needs to be streamed back to the client (e.g., a long-running tool execution result), it will pass this message to the `MCPStreamer`.
*   **`HTTP POST Handler` (Existing/Modified):** The existing HTTP handler for incoming MCP requests (e.g., `tools/call`, `resources/read`) will continue to use standard HTTP POST requests.
    *   Upon receiving a request, it will pass the raw message to the `mcp.Handler` for processing.
    *   For immediate responses, it will return the `JSONRPCMessage` directly as an HTTP response.
    *   For operations that might result in streamed updates (e.g., `tools/call` for a long-running task), the `mcp.Handler` will signal the `MCPStreamer` to send updates.

## 2. Communication Flow:

*   **Client to Server (Requests):**
    1.  Client sends an MCP `JSONRPCMessage` as the body of an HTTP POST request to a designated endpoint (e.g., `/mcp/request`).
    2.  The HTTP POST handler receives the request and passes the message to `mcp.Handler.HandleMessage`.
    3.  `mcp.Handler` processes the message and returns an immediate `JSONRPCMessage` response, which the HTTP POST handler sends back to the client.
*   **Server to Client (Streamed Updates/Notifications):**
    1.  Client establishes an SSE connection to a designated endpoint (e.g., `/mcp/stream`).
    2.  The `StreamHandler` registers the client's connection.
    3.  When `mcp.Handler` generates a notification or a streamed result, it sends this `JSONRPCMessage` to the `MCPStreamer`.
    4.  The `MCPStreamer` formats the `JSONRPCMessage` as an SSE event (e.g., `event: mcp_message\ndata: {json_message}\n\n`) and pushes it to all active SSE connections.
    5.  Clients receive these SSE events in real-time.

## 3. Data Format:

*   **Client-to-Server:** Standard JSON-RPC messages within HTTP POST request bodies.
*   **Server-to-Client:** Server-Sent Events (SSE). Each event will have:
    *   `event`: A type identifier (e.g., `mcp_message`).
    *   `data`: The JSON-serialized `JSONRPCMessage` (or a part of it, depending on the specific streaming needs).

## 4. Error Handling and Connection Management:

*   The `StreamHandler` will need robust error handling for broken connections and client disconnections.
*   A mechanism for clients to re-establish connections (e.g., using `Last-Event-ID` for SSE) should be considered for resilience.
*   Heartbeat messages (empty SSE events) can be sent periodically to keep connections alive and detect disconnections.

## 5. Architectural Diagram:

```mermaid
graph TD
    subgraph Client
        C[MCP Client]
    end

    subgraph Server
        direction LR
        HTTP_SERVER[HTTP Server] --> HTTP_POST_HANDLER[HTTP POST Handler]
        HTTP_SERVER --> STREAM_HANDLER[StreamHandler (New)]

        HTTP_POST_HANDLER --> MCP_HANDLER[mcp.Handler (Existing)]
        MCP_HANDLER --> MCP_STREAMER[MCPStreamer (New)]
        MCP_STREAMER --> STREAM_HANDLER

        MCP_HANDLER --> HTTP_POST_HANDLER
    end

    C -- HTTP POST Request (MCP JSON-RPC) --> HTTP_POST_HANDLER
    HTTP_POST_HANDLER -- HTTP Response (MCP JSON-RPC) --> C

    C -- SSE Connection --> STREAM_HANDLER
    STREAM_HANDLER -- SSE Events (MCP JSON-RPC) --> C