build: 
    go build -o bin/github-mcp ./cmd/github-mcp

run: 
   just build 
   ./bin/github-mcp

test: 
   go test ./...

clean: 
   rm -f bin/github-mcp
