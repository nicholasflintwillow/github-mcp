# Releases

## Summary
The Releases API provides endpoints to create, manage, and delete releases and release assets in a repository. Releases are used to package and distribute software, and can include release notes and downloadable assets.

## Common Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET    | /repos/{owner}/{repo}/releases | List releases for a repository |
| POST   | /repos/{owner}/{repo}/releases | Create a release |
| GET    | /repos/{owner}/{repo}/releases/{release_id} | Get a single release |
| PATCH  | /repos/{owner}/{repo}/releases/{release_id} | Update a release |
| DELETE | /repos/{owner}/{repo}/releases/{release_id} | Delete a release |
| GET    | /repos/{owner}/{repo}/releases/{release_id}/assets | List release assets |
| POST   | /repos/{owner}/{repo}/releases/{release_id}/assets | Upload a release asset |

## Official Documentation
[GitHub REST API: Releases](https://docs.github.com/en/rest/releases/releases)