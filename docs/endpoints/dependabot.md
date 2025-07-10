# Dependabot

## Summary
The Dependabot API provides endpoints to manage Dependabot alerts and secrets for repositories and organizations. It allows you to view, create, update, and delete Dependabot alerts and secrets, helping to automate dependency management and security.

## Common Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET    | /repos/{owner}/{repo}/dependabot/alerts | List Dependabot alerts for a repository |
| GET    | /orgs/{org}/dependabot/secrets | List Dependabot secrets for an organization |
| GET    | /repos/{owner}/{repo}/dependabot/secrets | List Dependabot secrets for a repository |
| PUT    | /repos/{owner}/{repo}/dependabot/secrets/{secret_name} | Create or update a Dependabot secret for a repository |

## Official Documentation
[GitHub REST API: Dependabot](https://docs.github.com/en/rest/dependabot)