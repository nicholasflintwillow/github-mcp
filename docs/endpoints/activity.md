# Activity

## Summary
The Activity API provides access to notifications, starring, watching, and other activity-related features on GitHub. It allows you to manage repository subscriptions, view who is watching or starring a repository, and access user notification feeds.

## Common Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET    | /users/{username}/received_events | List events that a user has received |
| GET    | /repos/{owner}/{repo}/stargazers | List users who have starred a repository |
| GET    | /repos/{owner}/{repo}/subscribers | List watchers of a repository |
| GET    | /notifications | List notifications for the authenticated user |

## Official Documentation
[GitHub REST API: Activity](https://docs.github.com/en/rest/activity)