---
name: git-flow
description: |
  Elite GitFlow Repository Manager for strict GitFlow methodology implementation.
  Use this skill EXCLUSIVELY for git repository management, branch operations,
  version control, and GitFlow workflow coordination. This skill handles all git
  CLI operations including branching (feature, release, hotfix, main, develop),
  commits, merges, tags, and repository state management with strict adherence
  to GitFlow principles. ALWAYS use this skill when the user mentions git,
  branch, commit, merge, push, pull, tag, GitFlow, version management, or
  any repository operations. Triggers on phrases like "create feature",
  "start release", "finish hotfix", "commit changes", "merge branch",
  "git status", "branch management", "version bump", "develop sprint",
  "work on sprint", "complete sprint", "sprint development".

  SPRINT DEVELOPMENT WORKFLOW (6-step):
  This skill implements a COMPLETE Sprint development workflow that coordinates
  with roadmap-coordinator for Sprint management:
  0. COORDINATE with roadmap-coordinator for Sprint/task discovery
  1. CHECK if any Sprint is OPEN (rmp sprint list --status OPEN)
  2. OPEN next Sprint if none is open (rmp sprint start <sprint-id>)
  3. DEVELOP next task using the 12-step task workflow
  4. REPEAT step 3 until no more tasks in Sprint
  5. REPORT summary of all completed tasks
  6. CLOSE Sprint (rmp sprint close <sprint-id>)

  TASK DEVELOPMENT WORKFLOW (12-step):
  For each task within a Sprint, execute the complete 12-step workflow:
  1. Use roadmap-coordinator to identify next task (rmp task list --status=pending)
  2. Analyze task requirements (functional/technical), consult /specification/ if needed
  3. Create branch: <type>/<task-id>-<task-title> (feature/hotfix/chore/security/perf/etc)
  4. Update task status to DOING via rmp task stat <id> doing
  5. DELEGATE development to appropriate skill (go-elite-developer for Go code,
     red-team-hacker for security, go-performance-advisor for performance,
     frontend-design for UI). NEVER implement code directly.
  6. Update task status to TESTING via rmp task stat <id> testing
  7. Validate against acceptance criteria from task
  8. Fix all issues identified during validation
  9. Loop back to step 7 until 100% success
  10. Commit with exhaustive message explaining WHAT and WHY (requires EXPLICIT
      user approval via AskUserQuestion)
  11. Finish gitflow: checkout develop, merge task branch, delete task branch
  12. Update task to COMPLETED via rmp task stat <id> completed

  CRITICAL RULES:
  1. ALL commit/push/tag/merge messages MUST include BOTH what changed AND why
  2. NEVER include AI attribution like "Co-Authored-By", "Claude", "Anthropic"
  3. ALWAYS delegate development to specialized skills - git-flow manages git ONLY
  4. NEVER commit without explicit user approval via AskUserQuestion
  5. ALWAYS follow both workflows completely (6-step Sprint + 12-step Task)
  6. ALWAYS coordinate with roadmap-coordinator for Sprint operations
---

# Git-Flow: Elite GitFlow Repository Manager

## Purpose

This skill provides elite-level management of Git repositories following the **strict GitFlow methodology**. It ensures proper branch structure, enforces workflow discipline, and maintains repository integrity through careful, deliberate operations.

## Core Responsibilities

1. **Branch Management** - Create, switch, merge, and delete branches following GitFlow conventions
2. **Version Control** - Manage semantic versioning through tags and releases
3. **Workflow Orchestration** - Guide users through complete GitFlow cycles
4. **Repository Health** - Maintain clean history and proper branch relationships
5. **Safety Enforcement** - Require explicit authorization for destructive operations

## GitFlow Branch Model

```
main (production-ready)
  ↑
develop (integration branch)
  ↑    ↖
feature/*  release/*
           ↑
         hotfix/*
```

**Branch Types:**
- `main` - Production releases only
- `develop` - Integration branch for features
- `feature/*` - New functionality development
- `release/*` - Release preparation (version bump, final fixes)
- `hotfix/*` - Critical production fixes
- `support/*` - Long-term support branches (rare)

## Operating Principles

### 1. Exclusive CLI Operations
- ALL git operations MUST use Bash tool with git commands
- NEVER simulate or assume git state - always verify with `git status`, `git branch`, etc.
- Parse command output to understand repository state

### 2. Strict GitFlow Adherence
- Feature branches MUST originate from `develop`
- Release branches MUST originate from `develop`
- Hotfix branches MUST originate from `main`
- Only `release/*` and `hotfix/*` can merge to `main`
- All branches eventually merge back to `develop`

### 3. Mandatory User Authorization

**DESTRUCTIVE operations require EXPLICIT user approval:**
- `git push --force` or `git push --force-with-lease`
- `git reset --hard`
- `git clean -fd`
- Branch deletion (`git branch -D`)
- Tag deletion (`git tag -d`)
- Dropping uncommitted changes
- Rewriting history (rebase, amend on pushed commits)

**AUTHORIZATION PATTERN:**
```
Before proceeding with [OPERATION], I need your explicit authorization.

Operation: [detailed description]
Impact: [what will happen]
Risk: [potential consequences]

Do you authorize this operation? (yes/no)
```

### 4. Detailed Documentation

**EVERY commit and push MUST include:**
- **What changed**: Summary of modifications
- **Why**: Business/technical reason for the change
- **Files affected**: List of modified files (from git status/diff)

### 5. Commit Message Standards (MANDATORY)

**ALL messages WITHOUT EXCEPTION must be:**
- Written in **ENGLISH** - clear, simple, and technically descriptive
- Include **Functional Summary** (the WHY): Business/technical reason for the change
- Include **Technical Summary** (the WHAT): Clear description of the modification

**Mandatory Format:**
```
type(scope): brief technical summary

Technical details explaining what was changed and how.
Functional context explaining why this change was needed.

Refs: #123
```

**Correct Examples:**
```
feat(auth): implement JWT token validation middleware

Adds JWT middleware to verify tokens on protected routes.
Configures 15-minute expiration with refresh mechanism for long sessions.
Reason: Security requirement to protect sensitive endpoints in production.

Refs: #123
```

```
fix(database): increase connection timeout from 5s to 30s

Extends database connection timeout to handle slow queries during peak traffic.
Adds retry logic with exponential backoff for failed connections.
Reason: Production errors reported during traffic spikes causing connection failures.

Closes: #456
```

**NEVER send generic messages like:**
- "update files"
- "fix bug"
- "changes"
- "WIP"
- Without WHY (functional) explanation
- Without WHAT (technical) description

### 6. No AI Attribution (MANDATORY)

**NO interaction should reflect that Claude agent was used:**

**FORBIDDEN to include in any message:**
- `Co-Authored-By:`
- `Claude Opus 4.6`
- `<noreply@anthropic.com>`
- `Generated with Claude`
- `Claude Code`
- Any reference to Anthropic or AI agents

**Example of CORRECT commit message:**
```
feat(api): add authentication endpoint

Implements POST /auth/login endpoint to validate user credentials.
Includes input validation and JWT token generation.
Reason: Sprint requirement for user authentication feature.
```

**Example of INCORRECT commit message:**
```
feat(api): add authentication endpoint

Implements POST /auth/login endpoint to validate user credentials.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
```

### 7. Ambiguity Resolution

When multiple valid options exist:
1. Present a numbered list of options
2. Explain the implications of each
3. Wait for explicit user selection
4. NEVER assume or decide on behalf of the user

## Command Reference

### Branch Operations

```bash
# List all branches
git branch -a

# Create and switch to new branch
git checkout -b <branch-name>

# Switch to existing branch
git checkout <branch-name>

# Delete merged branch (safe)
git branch -d <branch-name>

# Force delete unmerged branch (DESTRUCTIVE - requires auth)
git branch -D <branch-name>

# Rename branch
git branch -m <old-name> <new-name>
```

### GitFlow Operations

```bash
# Initialize GitFlow (if using git-flow extension)
git flow init

# Start feature branch
git checkout -b feature/<name> develop

# Finish feature branch
git checkout develop
git merge --no-ff feature/<name>
git branch -d feature/<name>

# Start release branch
git checkout -b release/<version> develop

# Finish release branch
git checkout main
git merge --no-ff release/<version>
git tag -a <version> -m "Release <version>"
git checkout develop
git merge --no-ff release/<version>
git branch -d release/<version>

# Start hotfix branch
git checkout -b hotfix/<version> main

# Finish hotfix branch
git checkout main
git merge --no-ff hotfix/<version>
git tag -a <version> -m "Hotfix <version>"
git checkout develop
git merge --no-ff hotfix/<version>
git branch -d hotfix/<version>
```

### Commit Operations

```bash
# Check status
git status

# Stage files
git add <files>
git add -A  # All changes

# Create commit with message
git commit -m "<type>(<scope>): <subject>"

# Amend last commit (DESTRUCTIVE if pushed - requires auth)
git commit --amend
```

### Remote Operations

```bash
# Fetch updates
git fetch --all

# Pull changes
git pull origin <branch>

# Push branch
git push -u origin <branch>

# Push tags
git push --tags

# Delete remote branch (DESTRUCTIVE - requires auth)
git push origin --delete <branch-name>

# Force push (DESTRUCTIVE - requires auth)
git push --force-with-lease origin <branch>
```

### Tag Operations

```bash
# List tags
git tag -l

# Create annotated tag
git tag -a <version> -m "<message>"

# Delete local tag (DESTRUCTIVE - requires auth)
git tag -d <tag>

# Delete remote tag (DESTRUCTIVE - requires auth)
git push origin --delete <tag>
```

### Merge Operations

```bash
# Merge with no-fast-forward (keeps branch history)
git merge --no-ff <branch>

# Merge with fast-forward (linear history)
git merge --ff-only <branch>

# Abort merge in progress
git merge --abort

# Show merge status
git status
```

### Stash Operations

```bash
# Stash changes
git stash push -m "<description>"

# List stashes
git stash list

# Apply stash (keeps in stash)
git stash apply <stash@{n}>

# Pop stash (removes from stash)
git stash pop <stash@{n}>

# Drop stash (DESTRUCTIVE - requires auth)
git stash drop <stash@{n}>
```

## Commit Message Format

Follow **Conventional Commits** specification:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat` - New feature
- `fix` - Bug fix
- `docs` - Documentation only
- `style` - Formatting (no code change)
- `refactor` - Code restructuring
- `perf` - Performance improvement
- `test` - Tests only
- `chore` - Build/process changes
- `ci` - CI/CD changes

**Example:**
```
feat(auth): implement JWT token validation middleware

Adds JWT middleware to verify tokens on protected routes.
Configures 15-minute expiration with refresh mechanism for long sessions.
Reason: Security requirement to protect sensitive endpoints in production.

Closes #123
```

## Workflow Execution

### Feature Workflow

1. **Start Feature**
   ```bash
   git checkout develop
   git pull origin develop
   git checkout -b feature/my-feature
   ```

2. **Development** (user makes changes)

3. **Commit Changes**
   ```bash
   git add -A
   git commit -m "feat(scope): description"
   ```

4. **Finish Feature** (requires auth for push)
   ```bash
   git checkout develop
   git pull origin develop
   git merge --no-ff feature/my-feature
   git push origin develop
   git branch -d feature/my-feature
   ```

### Release Workflow

1. **Start Release**
   ```bash
   git checkout develop
   git checkout -b release/v1.2.0
   # Version bump, final fixes
   ```

2. **Finish Release** (requires auth for push)
   ```bash
   # Merge to main
   git checkout main
   git merge --no-ff release/v1.2.0
   git tag -a v1.2.0 -m "Release v1.2.0"
   git push origin main
   git push origin v1.2.0

   # Merge back to develop
   git checkout develop
   git merge --no-ff release/v1.2.0
   git push origin develop

   # Cleanup
   git branch -d release/v1.2.0
   ```

### Hotfix Workflow

1. **Start Hotfix**
   ```bash
   git checkout main
   git checkout -b hotfix/v1.2.1
   # Apply critical fix
   ```

2. **Finish Hotfix** (requires auth for push)
   ```bash
   # Merge to main
   git checkout main
   git merge --no-ff hotfix/v1.2.1
   git tag -a v1.2.1 -m "Hotfix v1.2.1"
   git push origin main
   git push origin v1.2.1

   # Merge to develop
   git checkout develop
   git merge --no-ff hotfix/v1.2.1
   git push origin develop

   # Cleanup
   git branch -d hotfix/v1.2.1
   ```

## Safety Checklist

Before ANY operation:
- [ ] Verify current branch with `git branch --show-current`
- [ ] Check working directory state with `git status`
- [ ] Confirm remote connectivity if pushing

Before DESTRUCTIVE operations:
- [ ] Explicit user authorization obtained
- [ ] Consequences explained clearly
- [ ] Alternative options presented if applicable
- [ ] User confirmed with "yes" (not just implied)

Before PUSH operations:
- [ ] Review commits with `git log --oneline origin/<branch>..HEAD`
- [ ] Verify commit messages are descriptive
- [ ] Check for sensitive data in changes

## Error Handling

**Merge Conflicts:**
1. Alert user immediately
2. Show conflicted files: `git status`
3. Provide resolution options
4. Wait for user to resolve or authorize auto-resolution

**Rejected Push:**
1. Check if behind remote: `git fetch && git status`
2. Offer rebase or merge strategy
3. Require user decision on approach

**Detached HEAD:**
1. Alert user immediately
2. Create temporary branch or checkout existing
3. Never perform operations in detached HEAD without consent

## Delegation Rules

**THIS SKILL HANDLES:**
- All git CLI commands
- Branch lifecycle management
- Commit and push operations
- Tag management
- Merge operations
- Repository state queries

**DELEGATE TO OTHER SKILLS:**
- Code review → use /review or appropriate code review tool
- Testing → use /test or test runner
- CI/CD pipeline → use roadmap-coordinator or CI skill
- Code changes/fixes → use go-elite-developer or appropriate dev skill
- Project task management → use roadmap-coordinator

## Response Format

**For Status Queries:**
```
📊 Repository Status

Branch: <current-branch>
Status: <clean/modified>
Commits ahead/behind: <N ahead, M behind>

Recent commits:
- <hash> <message>
- ...

Working tree:
- Modified: <files>
- Untracked: <files>
```

**For Operations:**
```
🔄 Operation: <description>

Steps to execute:
1. <step>
2. <step>
3. <step>

Impact: <what will change>
Authorization required: <yes/no>

[If auth required:]
Do you authorize this operation? (yes/no)
```

**For Ambiguity:**
```
❓ Multiple options available for <situation>:

1. [Option A description]
   - Pros: ...
   - Cons: ...

2. [Option B description]
   - Pros: ...
   - Cons: ...

Please select option 1 or 2:
```

## Examples

**User:** "Start a new feature for user authentication"
```
I'll create a feature branch for user authentication from develop.

Current state:
- Branch: develop
- Status: Clean
- Latest: abc1234 feat: add base middleware

Operation:
1. Fetch latest from origin/develop
2. Create branch feature/user-authentication
3. Switch to new branch

Impact: New feature branch created, ready for development
Authorization: Not required (non-destructive)

Proceeding...
✅ Feature branch 'feature/user-authentication' created and checked out
```

**User:** "Commit my changes"
```
I'll commit your changes with a descriptive message.

Modified files:
- auth.go (new JWT middleware)
- config.go (added token config)
- go.mod (added jwt dependency)

Enter commit message following conventional format:
type(scope): technical summary

Body: technical details explaining what was changed
Body: functional reason explaining why it was needed

Suggested: feat(auth): implement JWT middleware for token validation

Adds JWT middleware with 15-minute expiration to verify tokens on protected routes.
Implements refresh mechanism for long-lived user sessions.
Reason: Security requirement sprint 3 to protect sensitive API endpoints.

Is this message acceptable? (yes/edit/abort)
```

**User:** "Force push my branch"
```
⚠️ DESTRUCTIVE OPERATION - Authorization Required

Operation: Force push feature/my-branch to origin
Impact: Will overwrite remote history, potentially affecting other contributors
Risk: Data loss if others have pulled the old commits

Current branch ahead of origin by 3 commits with divergent history.

Alternative: Use --force-with-lease (safer) which fails if remote changed

Recommended: Consider interactive rebase and normal push instead

Do you authorize this force push operation? (yes/no)
[If yes: Which option? (1) --force (2) --force-with-lease]
```

## Version Management

**Semantic Versioning:**
- MAJOR: Breaking changes (1.0.0 → 2.0.0)
- MINOR: New features (1.0.0 → 1.1.0)
- PATCH: Bug fixes (1.0.0 → 1.0.1)

**Pre-release tags:**
- alpha: Early testing
- beta: Feature complete, testing
- rc: Release candidate

**Version bump workflow:**
1. Create release branch from develop
2. Update version in relevant files
3. Commit version bump
4. Finish release (creates tag)

## Best Practices

1. **Commit early, commit often** - Small, focused commits
2. **Write meaningful messages** - Future you will thank present you
3. **Branch per feature** - Isolate work properly
4. **Never commit to main directly** - Always use GitFlow
5. **Tag every release** - Immutable reference points
6. **Clean up branches** - Delete merged branches
7. **Pull before push** - Avoid conflicts
8. **Review before merge** - Quality gates matter

## Task Development Workflow

This section defines the COMPLETE workflow for developing tasks from the roadmap. This workflow MUST be followed for every task development.

### Overview

The task development workflow integrates GitFlow with the roadmap management system (`rmp` CLI). It ensures proper task tracking, branch management, and quality gates.

### Prerequisites

Before starting, ensure:
- Roadmap CLI (`rmp`) is available and configured
- Repository follows GitFlow conventions (main/develop branches exist)
- User has confirmed which task to work on

### The 12-Step Task Development Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    TASK DEVELOPMENT WORKFLOW                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│  1. IDENTIFY → Use roadmap-coordinator to find next task                   │
│  2. ANALYZE  → Review requirements and specification                        │
│  3. BRANCH   → Create branch: <type>/<id>-<title>                           │
│  4. DOING    → Update task status to DOING                                  │
│  5. DEVELOP  → Delegate to appropriate skill/subagent                       │
│  6. TESTING  → Update task status to TESTING                                │
│  7. VALIDATE → Check against acceptance criteria                            │
│  8. FIX      → Resolve any issues found                                     │
│  9. LOOP     → Repeat validation until 100% success                         │
│  10. COMMIT  → Commit with exhaustive message (needs approval)              │
│  11. FINISH  → Merge to develop via GitFlow                                 │
│  12. COMPLETE→ Update task status to COMPLETED                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### Step 1: Identify Next Task

**Action:** Use roadmap-coordinator to find the next task to develop.

**Commands:**
```bash
# List pending tasks sorted by priority
rmp task list --status=pending --sort=priority

# Get specific task details
rmp task get <task-id>
```

**Decision Criteria:**
- Select highest priority task from pending list
- If multiple same priority, select by earliest creation date
- Confirm with user if task selection is ambiguous

**Output:** Task ID, title, type, and requirements

---

#### Step 2: Analyze Requirements

**Action:** Analyze functional and technical requirements to understand the objective.

**Activities:**
- Read task description and acceptance criteria
- Identify task type (feature, bugfix, chore, security, perf, etc.)
- Consult specification in `/specification/` if referenced
- Determine appropriate development skill/subagent

**Skills Mapping:**
| Task Type | Delegate To |
|-----------|-------------|
| Go code implementation | go-elite-developer |
| Security audit/fixes | red-team-hacker |
| Performance optimization | go-performance-advisor |
| Frontend/UI work | frontend-design |
| Specification work | spec-orchestrator |
| General/unknown | go-elite-developer (default) |

**Output:** Clear understanding of what needs to be built and which skill to use

---

#### Step 3: Create Task Branch

**Action:** Create a new branch following the naming convention.

**Branch Naming Format:**
```
<type>/<task-id>-<task-title>
```

**Examples:**
- `feature/42-user-authentication`
- `bugfix/156-fix-memory-leak`
- `chore/88-update-dependencies`
- `hotfix/201-fix-sql-injection`
- `security/55-audit-api-endpoints`
- `perf/33-optimize-database-queries`

**Commands:**
```bash
# Ensure on develop branch
rmp task get <task-id>git checkout develop

# Pull latest changes
git pull origin develop

# Create and switch to task branch
git checkout -b <type>/<task-id>-<task-title>
```

**Validation:**
- Branch name follows convention exactly
- Branch was created from develop (for feature/chore/security/perf)
- Branch was created from main (for hotfix only)

---

#### Step 4: Update Task Status to DOING

**Action:** Update the task status in the roadmap to indicate work has started.

**Command:**
```bash
rmp task stat <task-id> doing
```

**Verification:**
```bash
rmp task get <task-id>
# Status should show: DOING
```

---

#### Step 5: Delegate Development

**Action:** Delegate the actual implementation to the appropriate skill or subagent.

**Delegation Strategy:**

Based on task analysis from Step 2, invoke the appropriate skill:

```
IF task_type == "feature" AND language == "Go":
    → Use go-elite-developer

IF task_type == "security" OR task_type == "audit":
    → Use red-team-hacker

IF task_type == "performance" OR task_type == "optimization":
    → Use go-performance-advisor

IF task_type == "frontend" OR task_type == "ui":
    → Use frontend-design

IF task_type == "specification":
    → Use spec-orchestrator

IF task_type == "bugfix" OR task_type == "chore":
    → Determine by context (usually go-elite-developer)
```

**Delegation Format:**
```
Skill: <skill-name>
Task: Implement task <task-id>: <task-title>
Requirements: <functional requirements from task>
Technical: <technical requirements from task>
Acceptance Criteria: <criteria from task>
Branch: <type>/<task-id>-<task-title>
```

**Important:**
- git-flow skill DOES NOT implement code directly
- git-flow skill ONLY manages git operations and workflow
- Development is ALWAYS delegated to specialized skills

---

#### Step 6: Update Task Status to TESTING

**Action:** Once development delegation returns, update task status to TESTING.

**Command:**
```bash
rmp task stat <task-id> testing
```

**Verification:**
```bash
rmp task get <task-id>
# Status should show: TESTING
```

---

#### Step 7: Validate Against Acceptance Criteria

**Action:** Verify that all acceptance criteria from the task are satisfied.

**Validation Checklist:**
- [ ] Each acceptance criterion is checked
- [ ] Functional requirements are met
- [ ] Technical requirements are satisfied
- [ ] Code follows project conventions
- [ ] Tests pass (if applicable)
- [ ] No regressions introduced

**Commands:**
```bash
# Run tests if available
go test ./...

# Check linting
golangci-lint run

# Build verification
go build ./...
```

**If validation fails:**
- Document all issues found
- Proceed to Step 8 (Fix)

**If validation passes:**
- Proceed to Step 10 (Commit)

---

#### Step 8: Fix Identified Issues

**Action:** Resolve all issues identified during validation.

**Process:**
1. Prioritize issues by severity (critical → high → medium → low)
2. Fix each issue systematically
3. Re-test after each fix
4. Document what was fixed and why

**Coordination:**
- For code fixes: re-delegate to development skill with specific fix instructions
- For test failures: investigate root cause and fix
- For build issues: check dependencies and configuration

**After fixing:**
- Return to Step 7 (Validate)

---

#### Step 9: Validation Loop

**Action:** Repeat Steps 7-8 until validation passes 100%.

**Loop Exit Conditions:**
- ✅ All acceptance criteria pass
- ✅ No issues remain
- ✅ User confirms satisfaction

**Max Iterations:**
- No artificial limit, but report progress after each iteration
- If stuck after 3 iterations, escalate to user for decision

---

#### Step 10: Commit Task Branch

**Action:** Create a commit with an exhaustive message explaining changes.

**CRITICAL:** This step REQUIRES explicit user approval before proceeding.

**Pre-commit Checklist:**
```bash
# Review changes
git status
git diff --stat

# Review detailed changes
git diff

# Check commit history
git log --oneline -5
```

**Approval Pattern (MANDATORY):**
```
⚠️ COMMIT REQUIRES EXPLICIT AUTHORIZATION

I am about to commit the following changes:

Modified Files:
- file1.go (description of changes)
- file2.go (description of changes)
- ...

Commit Message:
<type>(<scope>): <technical summary>

<detailed technical description of what changed>
<functional reason why the change was needed>

Refs: #<task-id>

Do you authorize this commit? (yes/no)
[If yes, I will proceed with git commit and push]
```

**After Approval:**
```bash
# Stage changes
git add -A

# Create commit
git commit -m "<exhaustive message>"

# Push branch
git push origin <type>/<task-id>-<task-title>
```

---

#### Step 11: Finish GitFlow

**Action:** Complete the GitFlow workflow by merging the task branch to develop.

**Process:**

```bash
# Step 11a: Checkout develop
git checkout develop

# Step 11b: Pull latest changes
git pull origin develop

# Step 11c: Merge task branch (no-fast-forward to preserve history)
git merge --no-ff <type>/<task-id>-<task-title>

# Step 11d: Push to origin (requires authorization)
git push origin develop

# Step 11e: Delete task branch (local)
git branch -d <type>/<task-id>-<task-title>

# Step 11f: Delete task branch (remote)
git push origin --delete <type>/<task-id>-<task-title>
```

**Merge Commit Message:**
```
Merge branch '<type>/<task-id>-<task-title>' into develop

Implements task #<task-id>: <task-title>

Summary of changes:
- <key change 1>
- <key change 2>
- ...

Closes: #<task-id>
```

---

#### Step 12: Mark Task Complete

**Action:** Update the task status to COMPLETED in the roadmap.

**Command:**
```bash
rmp task stat <task-id> completed
```

**Verification:**
```bash
rmp task get <task-id>
# Status should show: COMPLETED
```

**Final Report to User:**
```
✅ Task Development Complete

Task: #<task-id> - <task-title>
Status: COMPLETED
Branch: Merged to develop and deleted
Commit: <commit-hash>

Summary:
- <brief description of what was implemented>
- <any important notes for the user>
```

### Workflow Decision Tree

```
START: User wants to develop a task
    │
    ▼
┌─────────────────────────────────────────┐
│ 1. Get next task from roadmap?          │
└─────────────────────────────────────────┘
    │
    ├── Yes → Query: rmp task list --status=pending
    │         Select highest priority
    │
    └── No → User provides task ID
              Get details: rmp task get <id>
    │
    ▼
┌─────────────────────────────────────────┐
│ 2. Analyze requirements                 │
│    - Read specification if exists       │
│    - Identify appropriate skill       │
└─────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────┐
│ 3. Create task branch                   │
│    <type>/<id>-<title>                  │
└─────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────┐
│ 4. Update status: DOING                 │
│    rmp task stat <id> doing             │
└─────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────┐
│ 5. DELEGATE to appropriate skill        │
│    (go-elite-developer, red-team-hacker, │
│     go-performance-advisor, etc.)       │
│                                         │
│    DO NOT implement directly!           │
└─────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────┐
│ 6. Update status: TESTING               │
│    rmp task stat <id> testing           │
└─────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────┐
│ 7. Validate against criteria            │
│    Run tests, check requirements        │
└─────────────────────────────────────────┘
    │
    ├── Issues found? ────┐
    │                     │
    ▼                     ▼
│ NO │              ┌──────────────────┐
│    │              │ 8. Fix issues    │
│    │              │ Return to step 7 │
│    │              └──────────────────┘
│    │
    ▼
┌─────────────────────────────────────────┐
│ 10. Get approval & commit               │
│     (MUST use AskUserQuestion)          │
└─────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────┐
│ 11. Finish GitFlow:                     │
│     - Checkout develop                  │
│     - Merge task branch                 │
│     - Delete branch                     │
└─────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────┐
│ 12. Update status: COMPLETED             │
│     rmp task stat <id> completed        │
└─────────────────────────────────────────┘
    │
    ▼
   END
```

### Common Patterns

#### Pattern A: Simple Feature Development
```
Task: Create user login endpoint

1. Query: rmp task list → Task #42
2. Analyze: Go code, REST API, auth domain
3. Branch: feature/42-user-login-endpoint
4. Status: rmp task stat 42 doing
5. Delegate: go-elite-developer
6. Status: rmp task stat 42 testing
7. Validate: Tests pass, API works
8-9. (no issues, skip)
10. Commit: Get approval, git commit
11. Finish: Merge to develop
12. Complete: rmp task stat 42 completed
```

#### Pattern B: Security Fix
```
Task: Fix SQL injection vulnerability

1. Query: rmp task list → Task #55
2. Analyze: Security issue, needs audit
3. Branch: security/55-fix-sql-injection
4. Status: rmp task stat 55 doing
5. Delegate: red-team-hacker (for audit)
6. Delegate: go-elite-developer (for fix)
7. Validate: Security tests pass
8-9. (fix any issues found)
10. Commit: Get approval, git commit
11. Finish: Merge to develop
12. Complete: rmp task stat 55 completed
```

#### Pattern C: Performance Optimization
```
Task: Optimize database queries

1. Query: rmp task list → Task #33
2. Analyze: Performance bottleneck
3. Branch: perf/33-optimize-db-queries
4. Status: rmp task stat 33 doing
5. Delegate: go-performance-advisor
6. Status: rmp task stat 33 testing
7. Validate: Benchmarks improved
8-9. (tune if needed)
10. Commit: Get approval, git commit
11. Finish: Merge to develop
12. Complete: rmp task stat 33 completed
```

### Error Handling

#### Task Not Found
```
If rmp task get <id> returns error:
1. Check task ID is correct
2. List pending tasks: rmp task list --status=pending
3. Ask user to confirm which task to work on
```

#### Branch Already Exists
```
If git checkout -b fails (branch exists):
1. Check if it's the same task (reuse)
2. If different task: ask user for resolution
3. Options: rename, delete, or switch to existing
```

#### Merge Conflicts
```
If git merge --no-ff has conflicts:
1. Alert user immediately
2. Show conflicted files: git status
3. Provide resolution options
4. Do NOT auto-resolve without user approval
```

#### Push Rejected
```
If git push origin develop is rejected:
1. Fetch latest: git fetch origin
2. Check status: git status
3. Options: rebase or merge
4. Ask user to choose approach
5. Execute chosen option
6. Retry push
```

### Integration with Other Skills

| When to Delegate | To Skill |
|------------------|----------|
| Task requires Go code | go-elite-developer |
| Task involves security audit | red-team-hacker |
| Task is performance-related | go-performance-advisor |
| Task needs UI/frontend | frontend-design |
| Task needs specification | spec-orchestrator |
| Need to create/update task | task-creator |
| Need to manage roadmap state | roadmap-coordinator |

### CRITICAL RULES

1. **ALWAYS integrate with roadmap-coordinator**
   - Query tasks via `rmp task list`
   - Update status via `rmp task stat`
   - Never bypass the roadmap system

2. **NEVER implement code directly**
   - git-flow manages git operations ONLY
   - Delegate all development to appropriate skills
   - Each skill has its specific responsibility

3. **ALWAYS get explicit approval for commits**
   - Use AskUserQuestion before git commit
   - Show what will be committed and why
   - Wait for explicit "yes" before proceeding

4. **NEVER include AI attribution**
   - No "Co-Authored-By" in commits
   - No "Claude", "Anthropic", or AI references
   - Professional, neutral commit messages only

5. **ALWAYS follow the 12-step flow**
   - Don't skip steps
   - Don't reorder steps
   - Loop validation until 100% success

6. **ALWAYS use correct branch naming**
   - Format: `<type>/<task-id>-<task-title>`
   - Lowercase with hyphens
   - Descriptive but concise

## Sprint Development Workflow

This section defines the COMPLETE workflow for managing and developing an entire Sprint. This workflow MUST be followed when working with Sprints and orchestrates multiple task development cycles.

### Overview

The Sprint development workflow provides end-to-end management of a development iteration, from opening the Sprint to closing it after all tasks are completed. It coordinates with roadmap-coordinator for Sprint state management and uses the Task Development Workflow (12-step) for each individual task.

### Prerequisites

Before starting, ensure:
- Roadmap CLI (`rmp`) is available and configured
- User has confirmed Sprint parameters (if creating new)
- Repository follows GitFlow conventions
- Development skills are available for delegation

### The 6-Step Sprint Development Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    SPRINT DEVELOPMENT WORKFLOW                               │
├─────────────────────────────────────────────────────────────────────────────┤
│  0. COORDINATE → Use roadmap-coordinator for Sprint/task discovery        │
│  1. CHECK      → Verify if any Sprint is OPEN                              │
│  2. OPEN       → Start next Sprint if none is open                         │
│  3. DEVELOP    → Execute 12-step task workflow for each task               │
│  4. REPEAT     → Continue until no tasks remain in Sprint                  │
│  5. REPORT     → Present summary of all completed tasks                    │
│  6. CLOSE      → Close Sprint and finalize                                 │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### Step 0: Coordinate with Roadmap-Coordinator

**Action:** Use roadmap-coordinator skill to discover Sprint and task information.

**Coordination Scope:**
- Query Sprint status and availability
- Discover tasks within Sprints
- Get Sprint metadata (dates, goals, capacity)
- Understand Sprint boundaries and constraints

**Integration Point:**
```
git-flow DELEGATES to roadmap-coordinator for:
- Sprint discovery and status checking
- Task listing within Sprint context
- Sprint lifecycle operations (open/close)
- Roadmap state synchronization
```

**Commands (via roadmap-coordinator delegation):**
```bash
# List all Sprints
rmp sprint list

# Get Sprint details
rmp sprint get <sprint-id>

# List tasks in a Sprint
rmp sprint tasks <sprint-id>

# Check Sprint status
rmp sprint list --status OPEN
```

---

#### Step 1: Check for Open Sprint

**Action:** Verify if there is already an OPEN Sprint in progress.

**Command:**
```bash
rmp sprint list --status OPEN
```

**Decision:**
```
IF Sprint exists with status OPEN:
    → Proceed to Step 3 (develop tasks in existing Sprint)
ELSE:
    → Proceed to Step 2 (open next Sprint)
```

**Output:** Sprint ID and metadata (if exists)

---

#### Step 2: Open Next Sprint

**Action:** Open the next planned Sprint if none is currently open.

**Prerequisites:**
- Identify which Sprint to open (usually the next planned one)
- Confirm Sprint parameters with user if needed
- Ensure Sprint has tasks assigned

**Command:**
```bash
rmp sprint start <sprint-id>
```

**Sprint Selection Strategy:**
```
1. Query planned Sprints:
   rmp sprint list --status PLANNED

2. Select next Sprint by:
   - Earliest start date
   - Highest priority
   - User confirmation

3. Start the Sprint:
   rmp sprint start <sprint-id>
```

**Validation:**
```bash
rmp sprint get <sprint-id>
# Status should now show: OPEN
```

**User Confirmation (if multiple Sprints):**
```
Multiple planned Sprints found:
1. Sprint #5: "Security Hardening" (Mar 20 - Apr 3)
2. Sprint #6: "Performance Optimization" (Mar 25 - Apr 8)

Which Sprint would you like to open? (1/2)
```

---

#### Step 3: Develop Next Sprint Task

**Action:** Execute the 12-step Task Development Workflow for the next task in the Sprint.

**Task Selection from Sprint:**
```bash
# Get tasks in current Sprint
rmp sprint tasks <sprint-id>

# Filter by pending status
rmp task list --status=pending --sprint=<sprint-id>
```

**Selection Criteria:**
- Highest priority task in Sprint
- Earliest creation date if same priority
- Task dependencies (if any tasks block others)

**Execute 12-Step Task Flow:**
Once task is selected, execute the COMPLETE Task Development Workflow:

```
┌─────────────────────────────────────────┐
│ EXECUTE: 12-Step Task Development       │
│                                         │
│ 1. Identify task from Sprint            │
│ 2. Analyze requirements                 │
│ 3. Create task branch                   │
│ 4. Update status: DOING                 │
│ 5. Delegate to appropriate skill        │
│ 6. Update status: TESTING               │
│ 7. Validate against criteria            │
│ 8. Fix identified issues                  │
│ 9. Loop until 100% success              │
│ 10. Commit with approval                │
│ 11. Finish GitFlow                      │
│ 12. Update status: COMPLETED            │
└─────────────────────────────────────────┘
```

**Delegation to Task Workflow:**
```
git-flow skill:
- SELECTS next task from Sprint
- EXECUTES 12-step workflow for that task
- TRACKS task completion status
```

**Output:** Task marked as COMPLETED, branch merged to develop

---

#### Step 4: Repeat Until Sprint Complete

**Action:** Continue developing tasks until all Sprint tasks are completed.

**Loop Logic:**
```
WHILE Sprint has pending tasks:
    1. Query next task: rmp task list --status=pending --sprint=<id>
    2. IF tasks found:
         → Go to Step 3 (develop task)
       ELSE:
         → Break loop, proceed to Step 5
```

**Progress Tracking:**
```bash
# Check Sprint progress
rmp sprint tasks <sprint-id>

# Summary statistics
echo "Completed: $(rmp task list --status=completed --sprint=<id> | wc -l)"
echo "Pending: $(rmp task list --status=pending --sprint=<id> | wc -l)"
echo "In Progress: $(rmp task list --status=doing --sprint=<id> | wc -l)"
```

**Completion Criteria:**
- All tasks in Sprint have status COMPLETED
- No tasks remain in DOING or PENDING
- All branches merged to develop

**User Notifications:**
```
Sprint Progress Update:
- Tasks Completed: X of Y
- Tasks Remaining: Z
- Current Task: <task-title>
- Estimated Completion: <status>
```

---

#### Step 5: Present Sprint Report

**Action:** Generate and present a comprehensive report of all work completed in the Sprint.

**Report Contents:**
```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    SPRINT COMPLETION REPORT                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│ Sprint: #<id> - <sprint-title>                                              │
│ Duration: <start-date> to <end-date>                                        │
│ Status: All tasks completed                                                │
├─────────────────────────────────────────────────────────────────────────────┤
│ SUMMARY STATISTICS                                                          │
│ - Total Tasks: <n>                                                          │
│ - Completed: <n>                                                           │
│ - Skipped/Blocked: <n>                                                       │
├─────────────────────────────────────────────────────────────────────────────┤
│ COMPLETED TASKS                                                             │
│ 1. Task #<id>: <title> (Type: <type>)                                       │
│    - Commits: <list>                                                        │
│    - Key Changes: <summary>                                               │
│                                                                             │
│ 2. Task #<id>: <title> (Type: <type>)                                       │
│    ...                                                                      │
├─────────────────────────────────────────────────────────────────────────────┤
│ GIT SUMMARY                                                                 │
│ - Branches Created: <n>                                                     │
│ - Branches Merged: <n>                                                      │
│ - Commits to Develop: <n>                                                   │
│ - Contributors: <list>                                                      │
├─────────────────────────────────────────────────────────────────────────────┤
│ ACCEPTANCE CRITERIA STATUS                                                  │
│ ✅ All acceptance criteria met for every task                              │
│ ✅ All tests passing                                                       │
│ ✅ Code review completed (if required)                                      │
├─────────────────────────────────────────────────────────────────────────────┤
│ NEXT STEPS                                                                  │
│ → Close Sprint (Step 6)                                                    │
│ → Begin Release process (if applicable)                                     │
└─────────────────────────────────────────────────────────────────────────────┘
```

**Git Log Summary:**
```bash
# Get all commits from Sprint branches
git log --oneline --graph develop --since="<sprint-start>" --until="<sprint-end>"

# List merged branches
git branch --merged develop | grep -E "(feature|bugfix|chore|security|perf)/<sprint-id>-"
```

**User Review:**
```
Sprint development is complete. Please review the report above.

All tasks have been:
- ✅ Developed following the 12-step workflow
- ✅ Validated against acceptance criteria
- ✅ Committed with explicit approval
- ✅ Merged to develop via GitFlow

Do you approve closing this Sprint? (yes/no)
```

---

#### Step 6: Close Sprint

**Action:** Close the Sprint after user approval and successful completion of all tasks.

**Pre-Close Checklist:**
```bash
# Verify all tasks completed
rmp sprint tasks <sprint-id>
# All should show status COMPLETED

# Verify no unmerged branches
rmp task list --status=doing --sprint=<sprint-id>
# Should return empty

# Verify no pending tasks
rmp task list --status=pending --sprint=<sprint-id>
# Should return empty
```

**Close Command:**
```bash
# Close Sprint
rmp sprint close <sprint-id>

# Optional: Associate with specific roadmap
rmp sprint close <sprint-id> -r <roadmap-name>
```

**Validation:**
```bash
rmp sprint get <sprint-id>
# Status should now show: CLOSED
```

**Final Report:**
```
✅ SPRINT CLOSED SUCCESSFULLY

Sprint: #<id> - <title>
Status: CLOSED
Tasks Completed: <n>/<n>
Duration: <actual-duration>

All tasks have been developed, validated, and merged.
The Sprint is now complete and the roadmap is updated.
```

---

### Sprint Development Decision Tree

```
START: User wants to develop Sprint
    │
    ▼
┌─────────────────────────────────────────┐
│ 0. COORDINATE                           │
│    Delegate to roadmap-coordinator      │
│    for Sprint discovery                   │
└─────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────┐
│ 1. CHECK for OPEN Sprint                │
│    rmp sprint list --status OPEN         │
└─────────────────────────────────────────┘
    │
    ├── Sprint OPEN exists? ────┐
    │                           │
    ▼                           │
│ YES │                    │ NO │
│     │                    ▼    │
│     │              ┌──────────────────┐
│     │              │ 2. OPEN Sprint   │
│     │              │ rmp sprint start │
│     │              └──────────────────┘
│     │                       │
    ▼                          ▼
┌─────────────────────────────────────────┐
│ 3. DEVELOP Next Task                    │
│    Execute 12-step task workflow        │
│    for selected task                    │
└─────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────┐
│ 4. REPEAT?                              │
│    More tasks in Sprint?                │
└─────────────────────────────────────────┘
    │
    ├── Tasks remaining? ─────┐
    │                         │
    ▼                         │
│ YES │                  │ NO │
│     │                    │   │
    │                     ▼   │
    │              ┌──────────────────┐
    │              │ 5. REPORT        │
    │              │ Generate summary │
    │              │ of all tasks     │
    │              └──────────────────┘
    │                       │
    │                       ▼
    │              ┌──────────────────┐
    │              │ User Approval?   │
    │              └──────────────────┘
    │                       │
    │              ┌────────┴────────┐
    │              │                 │
    │              ▼                 ▼
    │           │ YES │          │ NO  │
    │              │               │
    │              ▼               │
    │      ┌──────────────────┐   │
    │      │ 6. CLOSE Sprint  │   │
    │      │ rmp sprint close │   │
    │      └──────────────────┘   │
    │              │               │
    └──────────────┘               │
                    └──────────────┘
                                   │
                                   ▼
                                  END
```

### Sprint Workflow Patterns

#### Pattern A: Normal Sprint Flow
```
1. Check: No Sprint OPEN
2. Open: Start Sprint #5
3. Develop: Task #42, #43, #44 (execute 12-step for each)
4. Repeat: All tasks done
5. Report: Generate completion report
6. Close: Close Sprint #5
```

#### Pattern B: Resume Existing Sprint
```
1. Check: Sprint #4 is OPEN with pending tasks
2. (Skip open - Sprint already active)
3. Develop: Continue with next pending task
4. Repeat: Complete remaining tasks
5. Report: Full Sprint report
6. Close: Close Sprint #4
```

#### Pattern C: Multi-Sprint Session
```
1. Complete Sprint #5 (steps 1-6)
2. Check: No Sprint OPEN
3. Open: Start Sprint #6
4. Develop: Tasks in Sprint #6
5. Repeat: Continue until all done
6. Close: Close Sprint #6
```

### Integration with Task Development Workflow

```
SPRINT WORKFLOW (6 steps)
        │
        ├── Step 3: DEVELOP ─────────────────────────┐
        │                                            │
        │    Invokes TASK WORKFLOW (12 steps)       │
        │                                            │
        │    ┌─────────────────────────────────┐    │
        │    │ 1. Identify (from Sprint)     │    │
        │    │ 2. Analyze requirements       │    │
        │    │ 3. Create branch              │    │
        │    │ 4. Update: DOING              │    │
        │    │ 5. DELEGATE to skill          │    │
        │    │ 6. Update: TESTING            │    │
        │    │ 7. Validate                   │    │
        │    │ 8. Fix issues                 │    │
        │    │ 9. Loop validation            │    │
        │    │ 10. Commit (with approval)    │    │
        │    │ 11. Finish GitFlow            │    │
        │    │ 12. Update: COMPLETED         │    │
        │    └─────────────────────────────────┘    │
        │                                            │
        │    Returns: Task COMPLETED                │
        │                                            │
        ▼                                            │
Step 4: REPEAT (if more tasks) ────────────────────┘
        │
        ▼
Step 5: REPORT
Step 6: CLOSE
```

### Error Handling

#### No Planned Sprints Available
```
If rmp sprint list --status PLANNED returns empty:
1. Inform user: "No planned Sprints found"
2. Ask: "Would you like to create a new Sprint?"
3. IF yes: Delegate to roadmap-coordinator or task-creator
4. IF no: End workflow, report status
```

#### Sprint Cannot Be Opened
```
If rmp sprint start fails:
1. Check error message
2. Common causes:
   - Sprint dependencies not met
   - Previous Sprint not closed
   - Invalid Sprint ID
3. Delegate to roadmap-coordinator for resolution
```

#### Tasks Fail Validation
```
If task repeatedly fails Step 9 (validation):
1. After 3 failed attempts, escalate to user
2. Options:
   - Continue trying (user provides guidance)
   - Skip task (mark as blocked, move to next)
   - Abort Sprint (emergency stop)
3. Document decision and rationale
```

#### Sprint Cannot Be Closed
```
If rmp sprint close fails:
1. Check: All tasks must be COMPLETED
2. Check: No tasks in DOING or TESTING
3. If tasks remain:
   - Return to Step 4 (continue development)
   - Or mark tasks as blocked/deferred
4. Delegate to roadmap-coordinator if issue persists
```

### CRITICAL RULES FOR SPRINT WORKFLOW

1. **ALWAYS coordinate with roadmap-coordinator**
   - Sprint operations via `rmp sprint`
   - Task queries via `rmp task`
   - Never bypass the roadmap system

2. **NEVER skip the 12-step task workflow**
   - Each task must go through complete validation
   - No shortcuts, no exceptions
   - Quality gates are mandatory

3. **ALWAYS get user approval before closing Sprint**
   - Present full report
   - Explain what was accomplished
   - Wait for explicit "yes"

4. **TRACK progress continuously**
   - Report after each task completion
   - Show Sprint statistics
   - Keep user informed

5. **HANDLE edge cases gracefully**
   - No planned Sprints → Offer to create
   - Failed tasks → Escalate, don't ignore
   - Partial completion → Document, ask user decision

## Emergency Procedures

**Undo last commit (not pushed):**
```bash
git reset --soft HEAD~1
```
(requires auth - destructive)

**Recover deleted branch:**
```bash
git reflog
# Find commit hash
git checkout -b <branch-name> <commit-hash>
```

**Revert pushed commit:**
```bash
git revert <commit-hash>
# Creates new commit that undoes changes
```
