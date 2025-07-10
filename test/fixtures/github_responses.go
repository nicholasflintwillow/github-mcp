package fixtures

// GitHub API response fixtures for testing

// UserResponse represents a sample GitHub user response
const UserResponse = `{
  "login": "testuser",
  "id": 12345,
  "node_id": "MDQ6VXNlcjEyMzQ1",
  "avatar_url": "https://github.com/images/error/testuser_happy.gif",
  "gravatar_id": "",
  "url": "https://api.github.com/users/testuser",
  "html_url": "https://github.com/testuser",
  "followers_url": "https://api.github.com/users/testuser/followers",
  "following_url": "https://api.github.com/users/testuser/following{/other_user}",
  "gists_url": "https://api.github.com/users/testuser/gists{/gist_id}",
  "starred_url": "https://api.github.com/users/testuser/starred{/owner}{/repo}",
  "subscriptions_url": "https://api.github.com/users/testuser/subscriptions",
  "organizations_url": "https://api.github.com/users/testuser/orgs",
  "repos_url": "https://api.github.com/users/testuser/repos",
  "events_url": "https://api.github.com/users/testuser/events{/privacy}",
  "received_events_url": "https://api.github.com/users/testuser/received_events",
  "type": "User",
  "site_admin": false,
  "name": "Test User",
  "company": "Test Company",
  "blog": "https://testuser.com",
  "location": "Test City",
  "email": "test@example.com",
  "hireable": true,
  "bio": "Test bio",
  "twitter_username": "testuser",
  "public_repos": 10,
  "public_gists": 5,
  "followers": 100,
  "following": 50,
  "created_at": "2020-01-01T00:00:00Z",
  "updated_at": "2023-01-01T00:00:00Z"
}`

// UsersListResponse represents a sample GitHub users list response
const UsersListResponse = `[
  {
    "login": "testuser1",
    "id": 12345,
    "node_id": "MDQ6VXNlcjEyMzQ1",
    "avatar_url": "https://github.com/images/error/testuser1_happy.gif",
    "gravatar_id": "",
    "url": "https://api.github.com/users/testuser1",
    "html_url": "https://github.com/testuser1",
    "type": "User",
    "site_admin": false
  },
  {
    "login": "testuser2",
    "id": 12346,
    "node_id": "MDQ6VXNlcjEyMzQ2",
    "avatar_url": "https://github.com/images/error/testuser2_happy.gif",
    "gravatar_id": "",
    "url": "https://api.github.com/users/testuser2",
    "html_url": "https://github.com/testuser2",
    "type": "User",
    "site_admin": false
  }
]`

// OrganizationResponse represents a sample GitHub organization response
const OrganizationResponse = `{
  "login": "testorg",
  "id": 54321,
  "node_id": "MDEyOk9yZ2FuaXphdGlvbjU0MzIx",
  "url": "https://api.github.com/orgs/testorg",
  "repos_url": "https://api.github.com/orgs/testorg/repos",
  "events_url": "https://api.github.com/orgs/testorg/events",
  "hooks_url": "https://api.github.com/orgs/testorg/hooks",
  "issues_url": "https://api.github.com/orgs/testorg/issues",
  "members_url": "https://api.github.com/orgs/testorg/members{/member}",
  "public_members_url": "https://api.github.com/orgs/testorg/public_members{/member}",
  "avatar_url": "https://github.com/images/error/testorg_happy.gif",
  "description": "Test Organization",
  "name": "Test Org",
  "company": null,
  "blog": "https://testorg.com",
  "location": "Test City",
  "email": "contact@testorg.com",
  "twitter_username": "testorg",
  "is_verified": true,
  "has_organization_projects": true,
  "has_repository_projects": true,
  "public_repos": 25,
  "public_gists": 0,
  "followers": 500,
  "following": 0,
  "html_url": "https://github.com/testorg",
  "created_at": "2019-01-01T00:00:00Z",
  "updated_at": "2023-01-01T00:00:00Z",
  "type": "Organization"
}`

// TeamResponse represents a sample GitHub team response
const TeamResponse = `{
  "id": 98765,
  "node_id": "MDQ6VGVhbTk4NzY1",
  "url": "https://api.github.com/teams/98765",
  "html_url": "https://github.com/orgs/testorg/teams/testteam",
  "name": "Test Team",
  "slug": "testteam",
  "description": "Test team description",
  "privacy": "closed",
  "permission": "pull",
  "members_url": "https://api.github.com/teams/98765/members{/member}",
  "repositories_url": "https://api.github.com/teams/98765/repos",
  "parent": null,
  "members_count": 5,
  "repos_count": 10,
  "created_at": "2020-01-01T00:00:00Z",
  "updated_at": "2023-01-01T00:00:00Z",
  "organization": {
    "login": "testorg",
    "id": 54321,
    "node_id": "MDEyOk9yZ2FuaXphdGlvbjU0MzIx",
    "url": "https://api.github.com/orgs/testorg",
    "repos_url": "https://api.github.com/orgs/testorg/repos",
    "events_url": "https://api.github.com/orgs/testorg/events",
    "hooks_url": "https://api.github.com/orgs/testorg/hooks",
    "issues_url": "https://api.github.com/orgs/testorg/issues",
    "members_url": "https://api.github.com/orgs/testorg/members{/member}",
    "public_members_url": "https://api.github.com/orgs/testorg/public_members{/member}",
    "avatar_url": "https://github.com/images/error/testorg_happy.gif",
    "description": "Test Organization"
  }
}`

// ErrorResponse represents a sample GitHub API error response
const ErrorResponse = `{
  "message": "Not Found",
  "documentation_url": "https://docs.github.com/rest/reference/users#get-a-user"
}`
