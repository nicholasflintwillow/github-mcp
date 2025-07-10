# Actions

## Summary
The GitHub Actions API allows you to manage and interact with GitHub Actions workflows, runs, artifacts, and secrets. You can retrieve workflow run information, download artifacts, manage self-hosted runners, and control repository and organization-level Actions settings.

## Common Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET    | /repos/{owner}/{repo}/actions/workflows | List repository workflows |
| GET    | /repos/{owner}/{repo}/actions/workflows/{workflow_id} | Get a workflow |
| GET    | /repos/{owner}/{repo}/actions/runs | List workflow runs for a repository |
| GET    | /repos/{owner}/{repo}/actions/runs/{run_id} | Get a workflow run |
| POST   | /repos/{owner}/{repo}/actions/runs/{run_id}/rerun | Rerun a workflow |
| POST   | /repos/{owner}/{repo}/actions/runs/{run_id}/cancel | Cancel a workflow run |
| GET    | /repos/{owner}/{repo}/actions/artifacts | List workflow run artifacts |
| GET    | /repos/{owner}/{repo}/actions/artifacts/{artifact_id} | Get an artifact |
| DELETE | /repos/{owner}/{repo}/actions/artifacts/{artifact_id} | Delete an artifact |
| GET    | /repos/{owner}/{repo}/actions/secrets | List repository secrets |
| GET    | /repos/{owner}/{repo}/actions/runners | List self-hosted runners for a repository |
| POST   | /repos/{owner}/{repo}/actions/workflows/{workflow_id}/dispatches | Create a workflow dispatch event |
| PUT    | /repos/{owner}/{repo}/actions/permissions | Set GitHub Actions permissions for a repository |

## Official Documentation
[GitHub REST API: Actions](https://docs.github.com/en/rest/actions)