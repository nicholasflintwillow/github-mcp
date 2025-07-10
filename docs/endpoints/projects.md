# Projects

## Summary
The Projects API provides endpoints to manage the new GitHub Projects (beta), which offer flexible project planning and tracking features. You can create, update, and manage project boards, fields, views, and items for organizations and repositories.

## Common Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET    | /orgs/{org}/projects | List projects for an organization |
| GET    | /repos/{owner}/{repo}/projects | List projects for a repository |
| POST   | /orgs/{org}/projects | Create a project for an organization |
| POST   | /repos/{owner}/{repo}/projects | Create a project for a repository |
| GET    | /projects/{project_id} | Get a project |
| PATCH  | /projects/{project_id} | Update a project |
| DELETE | /projects/{project_id} | Delete a project |

## Official Documentation
[GitHub REST API: Projects](https://docs.github.com/en/rest/projects/projects)