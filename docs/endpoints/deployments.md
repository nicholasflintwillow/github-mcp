# Deployments

## Summary
The Deployments API provides endpoints to manage deployments and deployment statuses for repositories. It allows you to create, list, and update deployments, as well as track the status of each deployment, supporting continuous delivery workflows.

## Common Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST   | /repos/{owner}/{repo}/deployments | Create a deployment |
| GET    | /repos/{owner}/{repo}/deployments | List deployments for a repository |
| GET    | /repos/{owner}/{repo}/deployments/{deployment_id}/statuses | List deployment statuses |
| POST   | /repos/{owner}/{repo}/deployments/{deployment_id}/statuses | Create a deployment status |

## Official Documentation
[GitHub REST API: Deployments](https://docs.github.com/en/rest/deployments)