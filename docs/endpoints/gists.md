# Gists

## Summary
The Gists API provides endpoints to create, read, update, and delete gists—GitHub's way to share code snippets, notes, or any text content. Gists can be public or secret and support versioning and comments.

## Common Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET    | /gists | List gists for the authenticated user |
| POST   | /gists | Create a gist |
| GET    | /gists/{gist_id} | Get a single gist |
| PATCH  | /gists/{gist_id} | Update a gist |
| DELETE | /gists/{gist_id} | Delete a gist |

## Official Documentation
[GitHub REST API: Gists](https://docs.github.com/en/rest/gists)