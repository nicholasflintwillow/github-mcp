# Commits

## Summary
The Commits API provides endpoints to interact with commits in a repository. You can list commits, get details for a specific commit, compare commits, and manage commit statuses. This API is essential for tracking changes, reviewing history, and integrating with CI/CD systems.

## Common Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET    | /repos/{owner}/{repo}/commits | List commits in a repository |
| GET    | /repos/{owner}/{repo}/commits/{ref} | Get a specific commit |
| GET    | /repos/{owner}/{repo}/compare/{base}...{head} | Compare two commits |
| GET    | /repos/{owner}/{repo}/commits/{ref}/status | Get the combined status for a specific reference |

## Official Documentation
[GitHub REST API: Commits](https://docs.github.com/en/rest/commits)