# Reactions

## Summary
The Reactions API provides endpoints to manage emoji reactions to issues, comments, pull requests, and other resources. It allows you to add, list, and delete reactions, supporting community engagement and feedback.

## Common Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET    | /repos/{owner}/{repo}/issues/{issue_number}/reactions | List reactions for an issue |
| POST   | /repos/{owner}/{repo}/issues/{issue_number}/reactions | Create a reaction for an issue |
| DELETE | /reactions/{reaction_id} | Delete a reaction |

## Official Documentation
[GitHub REST API: Reactions](https://docs.github.com/en/rest/reactions)