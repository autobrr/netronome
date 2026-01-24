---
name: solve-issue
description: Solve the easiest GitHub issue from the current repo. Fetches open issues, filters out those with associated PRs, ranks remaining by difficulty (easiest first), solves one, commits to a new branch, and creates a PR referencing the issue. Stops after completing one issue.
user-invocable: true
allowed-tools: Bash(gh:*), Bash(git:*), Bash(go:*), Bash(make:*), Bash(pnpm:*), Read, Write, Edit, Glob, Grep, Task
argument-hint: [optional: filter by label or max difficulty]
---

# Solve One GitHub Issue

You are tasked with solving one GitHub issue from the current repository. Follow this workflow exactly and stop after creating a single PR.

## Phase 1: Fetch and Filter Issues

1. Fetch all open issues:

   ```
   gh issue list --state open --json number,title,body,labels,assignees --limit 50
   ```

2. Check which issues already have associated PRs by examining open PR titles/bodies for issue references:

   ```
   gh pr list --state open --json number,title,body --limit 100
   ```

   Also check closed/merged PRs that reference issues. Remove any issue that has an open PR associated with it.

3. Report the filtered list of issues that still need solving.

## Phase 2: Rank by Difficulty

Assess each remaining issue and assign a difficulty score (1-5):

**Score 1-2 (Easy):**

- Single file change
- Typo, config fix, small bug fix
- Clear requirements, obvious solution
- No database migrations
- No architectural changes

**Score 3-4 (Medium):**

- 2-3 files affected
- Feature enhancement or moderate refactor
- May need test updates
- Configuration or API changes

**Score 5 (Hard):**

- 4+ files affected
- New major feature
- Database migrations required
- Architectural changes
- Multi-service integration

Present the ranked list (easiest first) and confirm which issue you will solve.

## Phase 3: Solve the Easiest Issue

1. IMPORTANT: Ensure you are on the `develop` branch with a clean working tree before starting:

   ```
   git checkout develop
   git pull origin develop
   ```

2. Read the issue thoroughly. Understand what is being asked.

3. Read CLAUDE.md for project conventions and architecture.

4. Explore the relevant codebase areas to understand current implementation.

5. Create a feature branch from `develop`:
   - Format: `fix/issue-<number>-<short-slug>` or `feat/issue-<number>-<short-slug>`
   - Example: `fix/issue-102-agent-env-vars`

6. Implement the fix following project conventions:
   - Follow patterns in CLAUDE.md
   - Keep changes minimal and focused
   - Do not over-engineer

7. Verify the fix:
   - Run `go build ./...` to check compilation
   - Run `go test ./...` for backend changes
   - Run `cd web && pnpm lint && pnpm tsc --noEmit` for frontend changes
   - Run `./license.sh false` if new files were created

## Phase 4: Simplify and Review

IMPORTANT: Before committing, run the `code-simplifier` agent (via the Task tool with `subagent_type: "code-simplifier"`) on all files that were modified or created during this session. This reviews the changes for clarity, consistency, and maintainability. Apply any improvements it suggests, then re-run builds/tests to confirm nothing broke.

## Phase 4.5: Documentation Check

Evaluate whether the README needs updating based on the changes made:

1. Read the current README.md
2. Determine if the fix introduces user-facing changes that should be documented:
   - New features, commands, or configuration options
   - Changed behavior that users need to know about
   - New environment variables or CLI flags
   - New dependencies or setup requirements
3. If documentation updates are needed, make minimal, focused additions to README.md that match the existing style and structure
4. Skip this step if the change is purely internal (bug fix, refactor, test-only, or code cleanup with no user-facing impact)

## Phase 5: Commit and Create PR

1. Commit using Conventional Commits format:
   - Example: `fix(agent): load environment variables when no config file is provided`
   - Include `Closes #<number>` in the commit body
   - Do NOT add Co-Authored-By headers
   - Do NOT use emoji

2. Push the branch:

   ```
   git push -u origin <branch-name>
   ```

3. Create a PR:
   ```
   gh pr create --title "<conventional commit title>" --body "<body with Closes #number>"
   ```
   Include a Summary section and Test Plan section in the PR body.

## Phase 6: Stop

After successfully creating the PR:

- Report the issue number, PR URL, and what was fixed
- Explain why this was ranked as the easiest issue
- Stop. Do not attempt another issue.

## Constraints

- Solve exactly ONE issue per invocation
- Never skip the ranking phase
- Always verify builds/tests pass before committing
- Follow all CLAUDE.md guidelines strictly
- If no solvable issues exist, report that and stop
