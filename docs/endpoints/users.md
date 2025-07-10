# GitHub REST API: Users Endpoints

This page documents the main endpoints for working with **users** in the GitHub REST API.

## Overview

The Users API allows you to get public and private information about authenticated users, manage user profiles, and list users on GitHub.

## Common Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /users/{username}` | Get a user by username |
| `GET /user` | Get the authenticated user |
| `PATCH /user` | Update the authenticated user |
| `GET /users` | List all users |
| `GET /users/{username}/followers` | List followers of a user |
| `GET /users/{username}/following` | List users followed by a user |
| `GET /users/{username}/gists` | List gists for a user |
| `GET /users/{username}/repos` | List public repositories for a user |
| `GET /users/{username}/events` | List events performed by a user |
| `GET /users/{username}/received_events` | List events received by a user |

## Official Documentation

For a complete, up-to-date list of all Users API endpoints and their parameters, see the [GitHub REST API Users documentation](https://docs.github.com/en/rest/users/users).

---