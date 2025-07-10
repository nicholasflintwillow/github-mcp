# Codespaces

## Summary
The Codespaces API provides endpoints to manage GitHub Codespaces, which are cloud-based development environments. You can create, list, update, and delete codespaces, as well as manage their settings and resources.

## Common Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET    | /user/codespaces | List codespaces for the authenticated user |
| POST   | /user/codespaces | Create a codespace for the authenticated user |
| GET    | /user/codespaces/{codespace_name} | Get information about a specific codespace |
| DELETE | /user/codespaces/{codespace_name} | Delete a codespace |

## Official Documentation
[GitHub REST API: Codespaces](https://docs.github.com/en/rest/codespaces)