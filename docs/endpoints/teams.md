# GitHub REST API: Teams Endpoints

This page documents the main endpoints for working with **teams** in the GitHub REST API.

## Overview

The Teams API allows you to create, manage, and configure teams within your GitHub organization, including team membership, repositories, discussions, and synchronization with external identity providers.

## Common Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /orgs/{org}/teams` | List teams in an organization |
| `POST /orgs/{org}/teams` | Create a team |
| `GET /orgs/{org}/teams/{team_slug}` | Get a team by slug |
| `PATCH /orgs/{org}/teams/{team_slug}` | Update a team |
| `DELETE /orgs/{org}/teams/{team_slug}` | Delete a team |
| `GET /orgs/{org}/teams/{team_slug}/members` | List team members |
| `PUT /orgs/{org}/teams/{team_slug}/memberships/{username}` | Add or update team membership |
| `DELETE /orgs/{org}/teams/{team_slug}/memberships/{username}` | Remove a team member |
| `GET /orgs/{org}/teams/{team_slug}/repos` | List team repositories |
| `PUT /orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}` | Add or update team repository |
| `DELETE /orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}` | Remove a repository from a team |
| `GET /orgs/{org}/teams/{team_slug}/discussions` | List team discussions |
| `POST /orgs/{org}/teams/{team_slug}/discussions` | Create a team discussion |
| `GET /orgs/{org}/teams/{team_slug}/team-sync/group-mappings` | Get team synchronization group mappings |

## Official Documentation

- [Teams API Overview](https://docs.github.com/en/rest/teams/teams)[4]
- [Team Members Endpoints](https://docs.github.com/en/rest/teams/members)[1][3]
- [Team Synchronization Endpoints](https://docs.github.com/en/rest/teams/team-sync)[2]
- [Team Discussions Endpoints](https://docs.github.com/en/rest/teams/discussions)[5]

---