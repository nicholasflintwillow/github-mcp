# Licenses

## Summary
The Licenses API provides endpoints to retrieve information about open source licenses and the licenses applied to repositories. It allows you to get metadata for common licenses, detect the license for a repository, and access license content.

## Common Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET    | /licenses | List all commonly used licenses |
| GET    | /licenses/{license} | Get information about a specific license |
| GET    | /repos/{owner}/{repo}/license | Get the license for a repository |

## Official Documentation
[GitHub REST API: Licenses](https://docs.github.com/en/rest/licenses)