# Projects (Classic)

## Summary
The Projects (Classic) API provides endpoints to create, manage, and delete classic project boards, columns, and cards in repositories and organizations. Note: Projects (Classic) is deprecated in favor of the new Projects experience. For new projects, use the GraphQL API.

## Common Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET    | /users/{username}/projects | List projects for a user |
| GET    | /orgs/{org}/projects | List organization projects |
| GET    | /repos/{owner}/{repo}/projects | List repository projects |
| POST   | /repos/{owner}/{repo}/projects | Create a repository project |
| GET    | /projects/{project_id} | Get a project |
| PATCH  | /projects/{project_id} | Update a project |
| DELETE | /projects/{project_id} | Delete a project |
| GET    | /projects/{project_id}/columns | List columns in a project |
| POST   | /projects/{project_id}/columns | Create a project column |
| GET    | /projects/columns/{column_id}/cards | List cards in a column |
| POST   | /projects/columns/{column_id}/cards | Create a project card |

## Official Documentation
[GitHub REST API: Projects (Classic)](https://docs.github.com/en/rest/projects/projects)