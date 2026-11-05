# Referência de comandos

Gerado a partir da árvore Cobra por `make docs`. Não edite à mão.
A saída de `bk <comando> --help` é a fonte para a versão instalada.

## Flags globais

- `--hostname string` - Bitbucket host to use for this command
- `-h, --help` - help for bk
- `-v, --version` - version for bk

## bk alias

Create and manage command aliases

**Uso:** `bk alias <subcomando> [flags]`

## bk alias delete

Delete an alias

**Uso:** `bk alias delete NAME [flags]`

## bk alias list

List aliases

**Uso:** `bk alias list [flags]`

## bk alias set

Create an alias

**Uso:** `bk alias set NAME EXPANSION... [flags]`

## bk api

Make a raw Bitbucket REST API request

Send an authenticated request to the Bitbucket REST API.

The path is relative to the configured API base (/2.0 on Cloud,
/rest/api/1.0 on Data Center). Use this command for endpoints bk does not wrap.

Examples:
  bk api /user
  bk api /repositories/myteam/myrepo/pullrequests -f state=OPEN
  bk api POST /repositories/myteam/myrepo/commit/abc123/statuses/build \
    -f key=CI -f state=SUCCESSFUL -f url=https://ci.example.com/1
  bk api /repositories/myteam/myrepo/pullrequests --paginate

**Uso:** `bk api [METHOD] PATH [flags]`

**Flags:**

- `-f, --field stringArray` - Add a string field (KEY=VALUE, repeatable)
- `-H, --header stringArray` - Add a request header (repeatable)
- `--include` - Print the response status and headers
- `-i, --input string` - Read the request body from a file (- for stdin)
- `-X, --method string` - HTTP method
- `--paginate` - Follow pagination and print all results as a JSON array
- `-F, --raw-field stringArray` - Add a typed JSON field (KEY=JSON, repeatable)
- `--silent` - Do not print the response body

## bk auth

Authenticate with a Bitbucket host

Log in, log out, and inspect the credentials bk uses for each Bitbucket host.

**Uso:** `bk auth <subcomando> [flags]`

## bk auth login

Log in to Bitbucket

Store a credential for a Bitbucket host.

Bitbucket Cloud removed app passwords in July 2026. Use an Atlassian API token
created with Bitbucket scopes: the username is your Atlassian account email and
the password is the token. Repository, project, and workspace access tokens use
Bearer authentication and need no username.

Data Center personal access tokens use Bearer authentication by default.

**Uso:** `bk auth login [flags]`

**Flags:**

- `--access-token` - Treat the token as a Bearer access token
- `--auth-method string` - Authentication method: basic or bearer
- `--hostname string` - Bitbucket host (default bitbucket.org)
- `--no-verify` - Do not validate the credential against the host
- `-p, --project string` - Default Data Center project key
- `-u, --username string` - Atlassian account email for API tokens
- `--with-token` - Read the token from standard input
- `-w, --workspace string` - Default Cloud workspace

## bk auth logout

Remove the stored credential for a host

**Uso:** `bk auth logout [flags]`

**Flags:**

- `--hostname string` - Bitbucket host

## bk auth status

Show authentication status for configured hosts

**Uso:** `bk auth status [flags]`

**Flags:**

- `--hostname string` - Check only this host
- `--json[=FIELDS]` - Output JSON; use --json=FIELD,... to select fields
- `--template string` - Format output with a Go template

## bk auth token

Print the stored token for a host

Print the token exactly as stored so scripts can reuse it. Treat the output as a secret.

**Uso:** `bk auth token [flags]`

**Flags:**

- `--hostname string` - Bitbucket host

## bk branch

Work with branches and branch restrictions

**Uso:** `bk branch <subcomando> [flags]`

## bk branch list

List branches

**Uso:** `bk branch list [flags]`

**Flags:**

- `--json[=FIELDS]` - Output JSON; use --json=FIELD,... to select fields
- `-L, --limit int (default 30)` - Maximum number of branches (0 for all)
- `-R, --repo string` - Repository as OWNER/REPO
- `--template string` - Format output with a Go template

## bk branch restriction

Manage branch restrictions

**Uso:** `bk branch restriction <subcomando> [flags]`

## bk branch restriction create

Create a branch restriction

Create a branch restriction. Common kinds: push, delete, force, restrict_merges, require_approvals_to_merge, require_passing_builds_to_merge.

**Uso:** `bk branch restriction create [flags]`

**Flags:**

- `--group stringArray` - Group slug allowed or required by the rule (repeatable)
- `--json[=FIELDS]` - Output JSON; use --json=FIELD,... to select fields
- `--kind string` - Restriction kind
- `--match string (default glob)` - Pattern match kind: glob, regex, branching_model
- `--pattern string` - Branch pattern
- `-R, --repo string` - Repository as OWNER/REPO
- `--template string` - Format output with a Go template
- `--user stringArray` - User allowed or required by the rule (repeatable)
- `--value int` - Numeric value for the restriction (e.g. approvals required)

## bk branch restriction delete

Delete a branch restriction

**Uso:** `bk branch restriction delete ID [flags]`

**Flags:**

- `-R, --repo string` - Repository as OWNER/REPO
- `-y, --yes` - Confirm deletion

## bk branch restriction list

List branch restrictions

**Uso:** `bk branch restriction list [flags]`

**Flags:**

- `--json[=FIELDS]` - Output JSON; use --json=FIELD,... to select fields
- `-L, --limit int (default 100)` - Maximum number of rules (0 for all)
- `-R, --repo string` - Repository as OWNER/REPO
- `--template string` - Format output with a Go template

## bk browse

Open a repository, file, pull request, or pipeline in a browser

**Uso:** `bk browse [PATH] [flags]`

**Flags:**

- `-b, --branch string` - Branch to browse (default current branch)
- `--commit string` - Commit hash to browse
- `--no-open` - Print the URL instead of opening a browser
- `--pipeline int` - Pipeline build number to browse
- `--pr int` - Pull request ID to browse
- `-R, --repo string` - Repository as OWNER/REPO

## bk completion

Generate shell completion scripts

**Uso:** `bk completion [bash|zsh|fish|powershell] [flags]`

**Flags:**

- `--no-descriptions` - Disable completion descriptions

## bk config

Manage bk configuration

Read and write the configuration file. Credentials are never stored here.

**Uso:** `bk config <subcomando> [flags]`

## bk config get

Print a configuration value

Valid keys: active_host, host, workspace, project, user, kind, api_url.

**Uso:** `bk config get KEY [flags]`

## bk config list

Print the configuration file

**Uso:** `bk config list [flags]`

## bk config set

Set a configuration value

Valid keys: active_host, workspace, project, user, api_url. workspace and project apply to the active host.

**Uso:** `bk config set KEY VALUE [flags]`

## bk download

Manage repository download files

List, upload, and delete files in the repository downloads directory.

**Uso:** `bk download <subcomando> [flags]`

## bk download delete

Delete a download file

**Uso:** `bk download delete NAME [flags]`

**Flags:**

- `-R, --repo string` - Repository as OWNER/REPO
- `-y, --yes` - Confirm deletion

## bk download list

List download files

**Uso:** `bk download list [flags]`

**Flags:**

- `--json[=FIELDS]` - Output JSON; use --json=FIELD,... to select fields
- `-L, --limit int (default 30)` - Maximum number of files (0 for all)
- `-R, --repo string` - Repository as OWNER/REPO
- `--template string` - Format output with a Go template

## bk download upload

Upload a file to the downloads directory

**Uso:** `bk download upload FILE [flags]`

**Flags:**

- `--json[=FIELDS]` - Output JSON; use --json=FIELD,... to select fields
- `--name string` - Remote file name (default: base name of FILE)
- `-R, --repo string` - Repository as OWNER/REPO
- `--template string` - Format output with a Go template

## bk gpg-key

Manage GPG keys on your account

**Uso:** `bk gpg-key <subcomando> [flags]`

## bk gpg-key add

Add a GPG key

**Uso:** `bk gpg-key add [flags]`

**Flags:**

- `--json[=FIELDS]` - Output JSON; use --json=FIELD,... to select fields
- `--key-file string` - ASCII-armored public key file
- `--template string` - Format output with a Go template

## bk gpg-key delete

Delete a GPG key

**Uso:** `bk gpg-key delete FINGERPRINT [flags]`

**Flags:**

- `-y, --yes` - Confirm deletion

## bk gpg-key list

List GPG keys

**Uso:** `bk gpg-key list [flags]`

**Flags:**

- `--json[=FIELDS]` - Output JSON; use --json=FIELD,... to select fields
- `--template string` - Format output with a Go template

## bk pipeline

Work with Bitbucket Pipelines

List, inspect, run, stop, and read logs from Bitbucket Pipelines, and manage pipeline variables.

**Uso:** `bk pipeline <subcomando> [flags]`

## bk pipeline list

List pipeline runs

**Uso:** `bk pipeline list [flags]`

**Flags:**

- `-b, --branch string` - Filter by branch
- `--json[=FIELDS]` - Output JSON; use --json=FIELD,... to select fields
- `-L, --limit int (default 30)` - Maximum number of pipelines (0 for all)
- `-R, --repo string` - Repository as OWNER/REPO
- `--status string` - Filter by status (SUCCESSFUL, FAILED, IN_PROGRESS, ...)
- `--template string` - Format output with a Go template

## bk pipeline logs

Print pipeline step logs

**Uso:** `bk pipeline logs UUID|BUILD [flags]`

**Flags:**

- `-R, --repo string` - Repository as OWNER/REPO
- `--step string` - Step to print: 1-based index, name, or UUID

## bk pipeline run

Run a pipeline

**Uso:** `bk pipeline run [flags]`

**Flags:**

- `-b, --branch string` - Branch to run (default current branch)
- `--custom string` - Custom pipeline pattern
- `--json[=FIELDS]` - Output JSON; use --json=FIELD,... to select fields
- `-R, --repo string` - Repository as OWNER/REPO
- `--template string` - Format output with a Go template
- `--var stringArray` - Pipeline variable KEY=VALUE (repeatable)
- `-w, --web` - Open the pipeline in a browser

## bk pipeline stop

Stop a running pipeline

**Uso:** `bk pipeline stop UUID|BUILD [flags]`

**Flags:**

- `-R, --repo string` - Repository as OWNER/REPO

## bk pipeline variable

Manage pipeline variables

**Uso:** `bk pipeline variable <subcomando> [flags]`

## bk pipeline variable delete

Delete a pipeline variable

**Uso:** `bk pipeline variable delete UUID [flags]`

**Flags:**

- `-R, --repo string` - Repository as OWNER/REPO

## bk pipeline variable list

List pipeline variables

**Uso:** `bk pipeline variable list [flags]`

**Flags:**

- `--json[=FIELDS]` - Output JSON; use --json=FIELD,... to select fields
- `-R, --repo string` - Repository as OWNER/REPO
- `--template string` - Format output with a Go template

## bk pipeline variable set

Create or update a pipeline variable

**Uso:** `bk pipeline variable set KEY VALUE [flags]`

**Flags:**

- `-R, --repo string` - Repository as OWNER/REPO
- `--secured` - Mark the variable as secured

## bk pipeline view

View a pipeline run

**Uso:** `bk pipeline view UUID|BUILD [flags]`

**Flags:**

- `--json[=FIELDS]` - Output JSON; use --json=FIELD,... to select fields
- `-R, --repo string` - Repository as OWNER/REPO
- `--template string` - Format output with a Go template
- `-w, --web` - Open the pipeline in a browser

## bk pr

Work with pull requests

List, view, create, review, merge, decline, and inspect pull requests.

**Uso:** `bk pr <subcomando> [flags]`

## bk pr approve

Approve a pull request

**Uso:** `bk pr approve [ID] [flags]`

**Flags:**

- `-R, --repo string` - Repository as OWNER/REPO

## bk pr checkout

Check out a pull request branch

**Uso:** `bk pr checkout [ID] [flags]`

**Flags:**

- `-R, --repo string` - Repository as OWNER/REPO

## bk pr checks

Show build statuses for a pull request

List the commit statuses reported for the pull request head and exit 1 when any status failed.

**Uso:** `bk pr checks [ID] [flags]`

**Flags:**

- `--json[=FIELDS]` - Output JSON; use --json=FIELD,... to select fields
- `-L, --limit int (default 100)` - Maximum number of statuses
- `-R, --repo string` - Repository as OWNER/REPO
- `--template string` - Format output with a Go template

## bk pr comment

Comment on a pull request

**Uso:** `bk pr comment [ID] [flags]`

**Flags:**

- `-b, --body string` - Comment body
- `-F, --body-file string` - Read the comment from a file (- for stdin)
- `-R, --repo string` - Repository as OWNER/REPO

## bk pr comments

List pull request comments

**Uso:** `bk pr comments [ID] [flags]`

**Flags:**

- `--json[=FIELDS]` - Output JSON; use --json=FIELD,... to select fields
- `-L, --limit int (default 100)` - Maximum number of comments
- `-R, --repo string` - Repository as OWNER/REPO
- `--template string` - Format output with a Go template

## bk pr commits

List commits in a pull request

**Uso:** `bk pr commits [ID] [flags]`

**Flags:**

- `--json[=FIELDS]` - Output JSON; use --json=FIELD,... to select fields
- `-L, --limit int (default 100)` - Maximum number of commits
- `-R, --repo string` - Repository as OWNER/REPO
- `--template string` - Format output with a Go template

## bk pr create

Create a pull request

Open a pull request from the current branch (or --head) into the repository default branch (or --base).

**Uso:** `bk pr create [flags]`

**Flags:**

- `-B, --base string` - Destination branch
- `-b, --body string` - Pull request description
- `-F, --body-file string` - Read the description from a file (- for stdin)
- `--close-source-branch` - Close the source branch after merging
- `--draft` - Create as a draft pull request
- `-f, --fill` - Use the last commit subject and body
- `-H, --head string` - Source branch
- `--json[=FIELDS]` - Output JSON; use --json=FIELD,... to select fields
- `-R, --repo string` - Repository as OWNER/REPO
- `-r, --reviewer stringArray` - Request a review from a username (repeatable)
- `--template string` - Format output with a Go template
- `-t, --title string` - Pull request title
- `-w, --web` - Open the pull request in a browser

## bk pr decline

Decline a pull request

**Uso:** `bk pr decline [ID] [flags]`

**Flags:**

- `--reason string` - Reason for declining (posted as a comment)
- `-R, --repo string` - Repository as OWNER/REPO

## bk pr diff

Show the diff of a pull request

**Uso:** `bk pr diff [ID] [flags]`

**Flags:**

- `-R, --repo string` - Repository as OWNER/REPO

## bk pr list

List pull requests

**Uso:** `bk pr list [flags]`

**Flags:**

- `-a, --author string` - Filter by author (@me or username)
- `-B, --base string` - Filter by destination branch
- `-H, --head string` - Filter by source branch
- `--json[=FIELDS]` - Output JSON; use --json=FIELD,... to select fields
- `-L, --limit int (default 30)` - Maximum number of pull requests (0 for all)
- `-R, --repo string` - Repository as OWNER/REPO
- `--reviewer string` - Filter by reviewer (@me or username)
- `--state string (default open)` - Filter by state: open, merged, declined, all
- `--template string` - Format output with a Go template
- `-w, --web` - Open the pull request list in a browser

## bk pr merge

Merge a pull request

**Uso:** `bk pr merge [ID] [flags]`

**Flags:**

- `--close-source-branch` - Close the source branch after merging
- `--json[=FIELDS]` - Output JSON; use --json=FIELD,... to select fields
- `-m, --message string` - Merge commit message
- `-R, --repo string` - Repository as OWNER/REPO
- `-s, --strategy string (default merge_commit)` - Merge strategy: merge_commit, squash, fast_forward
- `--template string` - Format output with a Go template

## bk pr reopen

Reopen a declined pull request

**Uso:** `bk pr reopen [ID] [flags]`

**Flags:**

- `-R, --repo string` - Repository as OWNER/REPO

## bk pr request-changes

Request changes on a pull request

**Uso:** `bk pr request-changes [ID] [flags]`

**Flags:**

- `--clear` - Clear your change request
- `-R, --repo string` - Repository as OWNER/REPO

## bk pr unapprove

Remove your approval from a pull request

**Uso:** `bk pr unapprove [ID] [flags]`

**Flags:**

- `-R, --repo string` - Repository as OWNER/REPO

## bk pr view

View a pull request

**Uso:** `bk pr view [ID] [flags]`

**Flags:**

- `-c, --comments` - Include comments
- `--json[=FIELDS]` - Output JSON; use --json=FIELD,... to select fields
- `-L, --limit int (default 100)` - Maximum number of comments
- `-R, --repo string` - Repository as OWNER/REPO
- `--template string` - Format output with a Go template
- `-w, --web` - Open the pull request in a browser

## bk repo

Work with Bitbucket repositories

List, view, create, clone, fork, edit, and delete repositories.

**Uso:** `bk repo <subcomando> [flags]`

## bk repo clone

Clone a repository

**Uso:** `bk repo clone [OWNER/REPO] [DIRECTORY] [flags]`

**Flags:**

- `-R, --repo string` - Repository as OWNER/REPO
- `--ssh` - Clone using SSH

## bk repo create

Create a repository

**Uso:** `bk repo create OWNER/REPO [flags]`

**Flags:**

- `--clone` - Clone the repository after creating it
- `-d, --description string` - Repository description
- `--json[=FIELDS]` - Output JSON; use --json=FIELD,... to select fields
- `--project string` - Cloud project key
- `--public` - Create a public repository
- `--scm string (default git)` - Repository SCM
- `--template string` - Format output with a Go template

## bk repo delete

Delete a repository

**Uso:** `bk repo delete [OWNER/REPO] [flags]`

**Flags:**

- `-R, --repo string` - Repository as OWNER/REPO
- `-y, --yes` - Confirm deletion

## bk repo edit

Edit repository settings

**Uso:** `bk repo edit [OWNER/REPO] [flags]`

**Flags:**

- `--default-branch string` - Default branch name
- `-d, --description string` - Repository description
- `--json[=FIELDS]` - Output JSON; use --json=FIELD,... to select fields
- `--private` - Make the repository private
- `--public` - Make the repository public
- `-R, --repo string` - Repository as OWNER/REPO
- `--template string` - Format output with a Go template

## bk repo fork

Fork a repository

**Uso:** `bk repo fork [OWNER/REPO] [flags]`

**Flags:**

- `--clone` - Clone the fork after creating it
- `--json[=FIELDS]` - Output JSON; use --json=FIELD,... to select fields
- `--name string` - Name for the fork
- `-R, --repo string` - Repository as OWNER/REPO
- `--template string` - Format output with a Go template
- `-w, --workspace string` - Destination workspace

## bk repo list

List repositories in a workspace or project

**Uso:** `bk repo list [WORKSPACE] [flags]`

**Flags:**

- `--json[=FIELDS]` - Output JSON; use --json=FIELD,... to select fields
- `-L, --limit int (default 30)` - Maximum number of repositories (0 for all)
- `--role string` - Filter by role: admin, contributor, member, owner
- `--sort string (default -updated_on)` - Sort field
- `--template string` - Format output with a Go template
- `-w, --workspace string` - Workspace or project key

## bk repo view

View a repository

**Uso:** `bk repo view [OWNER/REPO] [flags]`

**Flags:**

- `--json[=FIELDS]` - Output JSON; use --json=FIELD,... to select fields
- `-R, --repo string` - Repository as OWNER/REPO
- `--template string` - Format output with a Go template
- `-w, --web` - Open the repository in a browser

## bk search

Search Bitbucket

**Uso:** `bk search <subcomando> [flags]`

## bk search code

Search code in a workspace

Search code in a Cloud workspace.

Atlassian announced the deprecation of the workspace code search endpoint for
November 2026. Data Center has no equivalent public API.

**Uso:** `bk search code QUERY [flags]`

**Flags:**

- `--json[=FIELDS]` - Output JSON; use --json=FIELD,... to select fields
- `-L, --limit int (default 30)` - Maximum number of results (0 for all)
- `--template string` - Format output with a Go template
- `-w, --workspace string` - Workspace slug

## bk snippet

Work with Bitbucket snippets

List, view, create, clone, and delete snippets in a workspace.

**Uso:** `bk snippet <subcomando> [flags]`

## bk snippet clone

Clone a snippet as a git repository

**Uso:** `bk snippet clone ID [DIRECTORY] [flags]`

**Flags:**

- `-w, --workspace string` - Workspace slug

## bk snippet create

Create a snippet

Create a snippet from one or more local files. Use --file NAME=PATH to control the snippet file name.

**Uso:** `bk snippet create [flags]`

**Flags:**

- `-f, --file stringArray` - File to include (PATH or NAME=PATH, repeatable)
- `--json[=FIELDS]` - Output JSON; use --json=FIELD,... to select fields
- `--private` - Create a private snippet
- `--template string` - Format output with a Go template
- `-t, --title string` - Snippet title
- `-w, --workspace string` - Workspace slug

## bk snippet delete

Delete a snippet

**Uso:** `bk snippet delete ID [flags]`

**Flags:**

- `-w, --workspace string` - Workspace slug
- `-y, --yes` - Confirm deletion

## bk snippet list

List snippets

**Uso:** `bk snippet list [flags]`

**Flags:**

- `--json[=FIELDS]` - Output JSON; use --json=FIELD,... to select fields
- `-L, --limit int (default 30)` - Maximum number of snippets (0 for all)
- `--role string` - Filter by role: owner, contributor, member
- `--template string` - Format output with a Go template
- `-w, --workspace string` - Workspace slug

## bk snippet view

View a snippet

**Uso:** `bk snippet view ID [flags]`

**Flags:**

- `--json[=FIELDS]` - Output JSON; use --json=FIELD,... to select fields
- `--template string` - Format output with a Go template
- `-w, --workspace string` - Workspace slug

## bk ssh-key

Manage SSH keys on your account

**Uso:** `bk ssh-key <subcomando> [flags]`

## bk ssh-key add

Add an SSH key

**Uso:** `bk ssh-key add [flags]`

**Flags:**

- `--json[=FIELDS]` - Output JSON; use --json=FIELD,... to select fields
- `--key-file string` - Public key file
- `--template string` - Format output with a Go template
- `--title string (default bk)` - Key label

## bk ssh-key delete

Delete an SSH key

**Uso:** `bk ssh-key delete ID [flags]`

**Flags:**

- `-y, --yes` - Confirm deletion

## bk ssh-key list

List SSH keys

**Uso:** `bk ssh-key list [flags]`

**Flags:**

- `--json[=FIELDS]` - Output JSON; use --json=FIELD,... to select fields
- `--template string` - Format output with a Go template

## bk version

Print the version of bk

**Uso:** `bk version [flags]`

## bk webhook

Manage repository webhooks

**Uso:** `bk webhook <subcomando> [flags]`

## bk webhook create

Create a webhook

**Uso:** `bk webhook create [flags]`

**Flags:**

- `-d, --description string` - Webhook description
- `--event stringArray (default [repo:push])` - Event to subscribe to (repeatable)
- `--json[=FIELDS]` - Output JSON; use --json=FIELD,... to select fields
- `-R, --repo string` - Repository as OWNER/REPO
- `--secret string` - Webhook secret
- `--template string` - Format output with a Go template
- `--url string` - Webhook target URL

## bk webhook delete

Delete a webhook

**Uso:** `bk webhook delete UUID [flags]`

**Flags:**

- `-R, --repo string` - Repository as OWNER/REPO
- `-y, --yes` - Confirm deletion

## bk webhook list

List webhooks

**Uso:** `bk webhook list [flags]`

**Flags:**

- `--json[=FIELDS]` - Output JSON; use --json=FIELD,... to select fields
- `-L, --limit int (default 30)` - Maximum number of webhooks (0 for all)
- `-R, --repo string` - Repository as OWNER/REPO
- `--template string` - Format output with a Go template

## bk webhook view

View a webhook

**Uso:** `bk webhook view UUID [flags]`

**Flags:**

- `--json[=FIELDS]` - Output JSON; use --json=FIELD,... to select fields
- `-R, --repo string` - Repository as OWNER/REPO
- `--template string` - Format output with a Go template

## bk workspace

Work with Bitbucket workspaces

List workspaces, view one workspace, and inspect its members.

**Uso:** `bk workspace <subcomando> [flags]`

## bk workspace list

List workspaces

**Uso:** `bk workspace list [flags]`

**Flags:**

- `--json[=FIELDS]` - Output JSON; use --json=FIELD,... to select fields
- `-L, --limit int (default 30)` - Maximum number of workspaces (0 for all)
- `--template string` - Format output with a Go template

## bk workspace members

List workspace members

**Uso:** `bk workspace members [SLUG] [flags]`

**Flags:**

- `--json[=FIELDS]` - Output JSON; use --json=FIELD,... to select fields
- `-L, --limit int (default 100)` - Maximum number of members (0 for all)
- `--template string` - Format output with a Go template

## bk workspace view

View a workspace

**Uso:** `bk workspace view [SLUG] [flags]`

**Flags:**

- `--json[=FIELDS]` - Output JSON; use --json=FIELD,... to select fields
- `--template string` - Format output with a Go template
- `-w, --web` - Open the workspace in a browser
