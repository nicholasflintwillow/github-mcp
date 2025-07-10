# Packages

## Summary
The Packages API provides endpoints to manage GitHub Packages, a service for hosting and managing packages (such as npm, Docker, Maven, etc.) within GitHub. You can list, view, and delete packages for users, organizations, and repositories.

## Common Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET    | /users/{username}/packages | List packages for a user |
| GET    | /orgs/{org}/packages | List packages for an organization |
| GET    | /repos/{owner}/{repo}/packages | List packages for a repository |
| GET    | /users/{username}/packages/{package_type}/{package_name} | Get a specific package for a user |
| DELETE | /users/{username}/packages/{package_type}/{package_name} | Delete a package for a user |

## Official Documentation
[GitHub REST API: Packages](https://docs.github.com/en/rest/packages)