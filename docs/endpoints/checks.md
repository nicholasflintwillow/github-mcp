# Checks

## Summary
The Checks API provides endpoints to manage check runs and check suites, which are used by GitHub Apps to report status, progress, and results of integrations and CI/CD workflows on commits and pull requests.

## Common Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET    | /repos/{owner}/{repo}/check-suites/{check_suite_id} | Get a check suite |
| GET    | /repos/{owner}/{repo}/commits/{ref}/check-suites | List check suites for a commit |
| POST   | /repos/{owner}/{repo}/check-runs | Create a check run |
| GET    | /repos/{owner}/{repo}/check-runs/{check_run_id} | Get a check run |

## Official Documentation
[GitHub REST API: Checks](https://docs.github.com/en/rest/checks)