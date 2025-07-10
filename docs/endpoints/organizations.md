# Organizations

## Summary
The Organizations API provides endpoints to manage organizations, their settings, members, teams, and roles. It allows you to create, update, and retrieve organization information, manage memberships, and configure organization-level settings and permissions.

## Common Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET    | /orgs | List all organizations |
| GET    | /orgs/{org} | Get an organization |
| PATCH  | /orgs/{org} | Update an organization |
| GET    | /orgs/{org}/members | List organization members |
| GET    | /orgs/{org}/teams | List teams in an organization |

## Official Documentation
[GitHub REST API: Organizations](https://docs.github.com/en/rest/orgs/orgs)