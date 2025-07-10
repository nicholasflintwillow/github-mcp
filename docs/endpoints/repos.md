# GitHub REST API: Repositories Endpoints

This page documents the main endpoints for working with **repositories** in the GitHub REST API.

## Overview

The Repositories API allows you to create, view, update, delete, and manage repositories and their settings.

## Common Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /repos/{owner}/{repo}` | Get a repository |
| `PATCH /repos/{owner}/{repo}` | Update a repository |
| `DELETE /repos/{owner}/{repo}` | Delete a repository |
| `GET /user/repos` | List repositories for the authenticated user |
| `GET /orgs/{org}/repos` | List repositories for an organization |
| `POST /user/repos` | Create a repository for the authenticated user |
| `POST /orgs/{org}/repos` | Create a repository for an organization |
| `GET /repos/{owner}/{repo}/branches` | List branches |
| `GET /repos/{owner}/{repo}/collaborators` | List collaborators |
| `PUT /repos/{owner}/{repo}/collaborators/{username}` | Add a collaborator |
| `GET /repos/{owner}/{repo}/languages` | List languages |
| `GET /repos/{owner}/{repo}/topics` | Get repository topics |
| `PUT /repos/{owner}/{repo}/topics` | Replace repository topics |
| `GET /repos/{owner}/{repo}/tags` | List tags |
| `GET /repos/{owner}/{repo}/forks` | List forks |

## Official Documentation

For a complete, up-to-date list of all Repositories API endpoints and their parameters, see the [GitHub REST API Repositories documentation](https://docs.github.com/en/rest/repos/repos).

---