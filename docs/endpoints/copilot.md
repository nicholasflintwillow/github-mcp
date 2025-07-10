# Copilot

## Summary
The Copilot API provides endpoints to manage and interact with GitHub Copilot features, such as enabling or disabling Copilot for organizations or repositories, and retrieving Copilot usage information.

## Common Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET    | /orgs/{org}/copilot/billing | Get Copilot billing information for an organization |
| GET    | /repos/{owner}/{repo}/copilot | Get Copilot status for a repository |
| PUT    | /repos/{owner}/{repo}/copilot | Enable Copilot for a repository |
| DELETE | /repos/{owner}/{repo}/copilot | Disable Copilot for a repository |

## Official Documentation
[GitHub REST API: Copilot](https://docs.github.com/en/rest/copilot)