# Secret Scanning

## Summary
The Secret Scanning API provides endpoints to manage secret scanning alerts and push protection for repositories and organizations. It helps detect and prevent the exposure of secrets such as API keys and credentials in your codebase.

## Common Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET    | /repos/{owner}/{repo}/secret-scanning/alerts | List secret scanning alerts for a repository |
| GET    | /orgs/{org}/secret-scanning/alerts | List secret scanning alerts for an organization |
| GET    | /repos/{owner}/{repo}/secret-scanning/alerts/{alert_number} | Get a secret scanning alert |
| PATCH  | /repos/{owner}/{repo}/secret-scanning/alerts/{alert_number} | Update a secret scanning alert |
| GET    | /repos/{owner}/{repo}/secret-scanning/push-protection/alerts | List push protection alerts for a repository |

## Official Documentation
[GitHub REST API: Secret Scanning](https://docs.github.com/en/rest/secret-scanning)