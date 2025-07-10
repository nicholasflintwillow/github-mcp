# github-mcp
A github mcp with full api access.


## Configuration

This server requires a GitHub Personal Access Token (PAT) to authenticate with the GitHub API. The token must be provided via an environment variable.

### Setting the GitHub Token

You must set the `GITHUB_PERSONAL_ACCESS_TOKEN` environment variable before running the server.

```bash
export GITHUB_PERSONAL_ACCESS_TOKEN="your_github_token_here"
```

The server will fail to start if this environment variable is not set or is empty. It is recommended to add this line to your shell's startup file (e.g., `.zshrc`, `.bash_profile`) to avoid having to set it manually in every session.
