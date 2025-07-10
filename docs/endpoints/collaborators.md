# Collaborators

## Summary
The Collaborators API provides endpoints to manage repository collaborators. You can add, remove, and list collaborators, as well as check a user's permission level for a repository.

## Common Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET    | /repos/{owner}/{repo}/collaborators | List collaborators for a repository |
| PUT    | /repos/{owner}/{repo}/collaborators/{username} | Add a collaborator to a repository |
| DELETE | /repos/{owner}/{repo}/collaborators/{username} | Remove a collaborator from a repository |
| GET    | /repos/{owner}/{repo}/collaborators/{username}/permission | Get a collaborator's permission level |

## Official Documentation
[GitHub REST API: Collaborators](https://docs.github.com/en/rest/collaborators)