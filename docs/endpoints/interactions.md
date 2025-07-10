# Interactions

## Summary
The Interactions API provides endpoints to manage interaction limits for repositories and organizations. This allows you to restrict who can comment, open issues, or create pull requests, helping to reduce spam and manage community engagement.

## Common Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET    | /repos/{owner}/{repo}/interaction-limits | Get interaction limits for a repository |
| PUT    | /repos/{owner}/{repo}/interaction-limits | Set interaction limits for a repository |
| DELETE | /repos/{owner}/{repo}/interaction-limits | Remove interaction limits for a repository |
| GET    | /orgs/{org}/interaction-limits | Get interaction limits for an organization |
| PUT    | /orgs/{org}/interaction-limits | Set interaction limits for an organization |
| DELETE | /orgs/{org}/interaction-limits | Remove interaction limits for an organization |

## Official Documentation
[GitHub REST API: Interactions](https://docs.github.com/en/rest/interactions)