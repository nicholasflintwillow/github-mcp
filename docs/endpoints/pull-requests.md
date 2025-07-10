# Pull Requests

## Summary
The Pull Requests API provides endpoints to create, manage, and review pull requests in repositories. It allows you to open, update, merge, and close pull requests, as well as manage reviews, comments, and requested reviewers.

## Common Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET    | /repos/{owner}/{repo}/pulls | List pull requests for a repository |
| POST   | /repos/{owner}/{repo}/pulls | Create a pull request |
| GET    | /repos/{owner}/{repo}/pulls/{pull_number} | Get a pull request |
| PATCH  | /repos/{owner}/{repo}/pulls/{pull_number} | Update a pull request |
| PUT    | /repos/{owner}/{repo}/pulls/{pull_number}/merge | Merge a pull request |
| GET    | /repos/{owner}/{repo}/pulls/{pull_number}/reviews | List reviews on a pull request |
| POST   | /repos/{owner}/{repo}/pulls/{pull_number}/reviews | Create a review for a pull request |

## Official Documentation
[GitHub REST API: Pull Requests](https://docs.github.com/en/rest/pulls/pulls)