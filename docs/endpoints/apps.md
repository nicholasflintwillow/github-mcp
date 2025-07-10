# Apps

## Summary
The Apps API allows you to interact with GitHub Apps, including retrieving information about apps, managing app installations, and performing actions as a GitHub App. This includes endpoints for authenticating as an app, listing installations, and accessing app-specific resources.

## Common Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET    | /app | Get the authenticated app |
| GET    | /app/installations | List installations for the authenticated app |
| GET    | /users/{username}/installations | List app installations accessible to the user |
| GET    | /repos/{owner}/{repo}/installation | Get a repository installation for the authenticated app |

## Official Documentation
[GitHub REST API: Apps](https://docs.github.com/en/rest/apps)