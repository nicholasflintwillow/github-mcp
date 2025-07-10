# Metrics

## Summary
The Metrics API provides endpoints to access repository traffic, clones, views, and popular content statistics. This helps repository maintainers understand how their projects are being discovered and used.

## Common Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET    | /repos/{owner}/{repo}/traffic/views | Get the total number of views and breakdown per day or week |
| GET    | /repos/{owner}/{repo}/traffic/clones | Get the total number of clones and breakdown per day or week |
| GET    | /repos/{owner}/{repo}/traffic/popular/referrers | Get the top referrers to a repository |
| GET    | /repos/{owner}/{repo}/traffic/popular/paths | Get the top paths accessed in a repository |

## Official Documentation
[GitHub REST API: Metrics](https://docs.github.com/en/rest/metrics)