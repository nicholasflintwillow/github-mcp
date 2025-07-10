# Git Database

## Summary
The Git Database API provides endpoints to interact directly with Git objects such as blobs, trees, and commits. It allows you to read and write raw Git data, create new references, and manage the underlying Git structure of a repository.

## Common Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET    | /repos/{owner}/{repo}/git/commits/{commit_sha} | Get a commit object |
| GET    | /repos/{owner}/{repo}/git/trees/{tree_sha} | Get a tree object |
| GET    | /repos/{owner}/{repo}/git/blobs/{file_sha} | Get a blob object |
| POST   | /repos/{owner}/{repo}/git/refs | Create a reference |
| POST   | /repos/{owner}/{repo}/git/commits | Create a commit object |

## Official Documentation
[GitHub REST API: Git Database](https://docs.github.com/en/rest/git)