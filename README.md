# GitHub API MCP Server

A Model Context Protocol (MCP) server that provides access to GitHub's REST API.

## Project Structure

```
├── cmd/github-mcp/          # Application entry point
├── internal/
│   ├── config/              # Configuration management
│   ├── server/              # HTTP server implementation
│   ├── logger/              # Structured logging
│   └── errors/              # Error handling
├── pkg/                     # Public packages (future)
├── test/                    # Test files
└── docs/                    # Documentation
```

## Quick Start

### Prerequisites

- Go 1.20 or later
- GitHub Personal Access Token

### Building

```bash
go build -o bin/github-mcp ./cmd/github-mcp
```

### Running

```bash
export GITHUB_PERSONAL_ACCESS_TOKEN=your_token_here
./bin/github-mcp
```

### Configuration

The server can be configured using environment variables:

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `GITHUB_PERSONAL_ACCESS_TOKEN` | GitHub Personal Access Token | - | Yes |
| `PORT` | Server port | 8080 | No |
| `HOST` | Server host | 0.0.0.0 | No |
| `LOG_LEVEL` | Log level (DEBUG, INFO, WARN, ERROR) | INFO | No |
| `LOG_FORMAT` | Log format (json, text) | json | No |
| `CACHE_TTL` | Cache TTL in seconds | 60 | No |
| `MAX_CONCURRENT_REQUESTS` | Maximum concurrent requests | 100 | No |

### Health Checks

- Health: `GET /health`
- Readiness: `GET /ready`

## Development Status

This is the initial infrastructure setup. The following components are implemented:

- ✅ Go project structure
- ✅ Configuration loading from environment variables
- ✅ Structured logging with configurable levels and formats
- ✅ Basic HTTP server with middleware
- ✅ Error handling with typed errors
- ✅ Health and readiness endpoints
- ✅ MCP protocol implementation
- ✅ GitHub API client

## Next Steps

1. Implement GitHub API authentication and client
2. Implement MCP protocol support
3. Add GitHub API endpoint mappings (Users, Organizations, Teams)
4. Add comprehensive testing (Users, Organizations, Teams APIs)
5. Add Docker containerization
