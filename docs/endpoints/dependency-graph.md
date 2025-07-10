# Dependency Graph

## Summary
The Dependency Graph API provides endpoints to submit and review dependencies for a project. This enables you to add dependencies that are resolved during build or compilation, enhancing GitHub's dependency graph feature. The dependency graph helps track direct and transitive dependencies, supports security alerts, and integrates with Dependabot for automated updates.

## Common Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST   | /repos/{owner}/{repo}/dependency-graph/snapshots | Create a snapshot of dependencies for a repository |
| GET    | /repos/{owner}/{repo}/dependency-graph/compare/{basehead} | Compare dependency graphs between two commits |
| POST   | /repos/{owner}/{repo}/dependency-graph/dependency-submission | Submit dependencies for a project |

## Official Documentation
[GitHub REST API: Dependency Graph](https://docs.github.com/en/rest/dependency-graph)