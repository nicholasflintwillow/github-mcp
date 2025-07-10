# Migrations

## Summary
The Migrations API provides endpoints to help you migrate repositories and organizations to or from GitHub. It allows you to start, monitor, and manage migrations, as well as download migration archives and handle repository imports.

## Common Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST   | /orgs/{org}/migrations | Start an organization migration |
| GET    | /orgs/{org}/migrations | List organization migrations |
| GET    | /orgs/{org}/migrations/{migration_id} | Get an organization migration status |
| GET    | /orgs/{org}/migrations/{migration_id}/archive | Download a migration archive |
| DELETE | /orgs/{org}/migrations/{migration_id}/archive | Delete a migration archive |

## Official Documentation
[GitHub REST API: Migrations](https://docs.github.com/en/rest/migrations)