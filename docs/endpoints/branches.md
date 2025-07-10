# Branches

## Summary
The Branches API provides endpoints to manage repository branches, including listing branches, getting branch details, and protecting branches. It allows you to control branch protection rules and view branch information.

## Common Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET    | /repos/{owner}/{repo}/branches | List branches in a repository |
| GET    | /repos/{owner}/{repo}/branches/{branch} | Get a branch |
| PUT    | /repos/{owner}/{repo}/branches/{branch}/protection | Update branch protection |
| DELETE | /repos/{owner}/{repo}/branches/{branch}/protection | Remove branch protection |

## Official Documentation
[GitHub REST API: Branches](https://docs.github.com/en/rest/branches)