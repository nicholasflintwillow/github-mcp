# GitHub REST API: Issues Endpoints

This page documents the main endpoints for working with **issues** in the GitHub REST API.

## Overview

The Issues API allows you to create, view, update, and manage issues within repositories. Issues are used for bug tracking, feature requests, and general discussion.

## Common Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /repos/{owner}/{repo}/issues` | List issues for a repository |
| `POST /repos/{owner}/{repo}/issues` | Create an issue |
| `GET /repos/{owner}/{repo}/issues/{issue_number}` | Get a single issue |
| `PATCH /repos/{owner}/{repo}/issues/{issue_number}` | Update an issue |
| `GET /repos/{owner}/{repo}/issues/{issue_number}/comments` | List comments on an issue |
| `POST /repos/{owner}/{repo}/issues/{issue_number}/comments` | Create a comment on an issue |
| `GET /repos/{owner}/{repo}/issues/comments/{comment_id}` | Get a single comment |
| `PATCH /repos/{owner}/{repo}/issues/comments/{comment_id}` | Update a comment |
| `DELETE /repos/{owner}/{repo}/issues/comments/{comment_id}` | Delete a comment |
| `GET /repos/{owner}/{repo}/labels` | List labels for a repository |
| `POST /repos/{owner}/{repo}/labels` | Create a label |
| `GET /repos/{owner}/{repo}/milestones` | List milestones for a repository |
| `POST /repos/{owner}/{repo}/milestones` | Create a milestone |

## Official Documentation

For a complete, up-to-date list of all Issues API endpoints and their parameters, see the [GitHub REST API Issues documentation](https://docs.github.com/en/rest/issues/issues).

---