# Meta

## Summary
The Meta API provides endpoints to retrieve meta information about GitHub, such as the IP addresses of GitHub services and the current API version. This is useful for configuring firewalls, monitoring, and understanding GitHub's infrastructure.

## Common Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET    | /meta | Get meta information about GitHub, including IP addresses |
| GET    | /rate_limit | Get the current rate limit status for the authenticated user |

## Official Documentation
[GitHub REST API: Meta](https://docs.github.com/en/rest/meta)