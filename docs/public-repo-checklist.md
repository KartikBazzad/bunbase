# Public Repo Protection Checklist

Use this checklist to protect the BunBase codebase now that the repo is public.

## In the repo (already added)

- **`.github/SECURITY.md`** – How to report vulnerabilities privately (no public issues).
- **`.github/dependabot.yml`** – Weekly dependency and Docker base-image updates; security alerts via GitHub.
- **`LICENSE`** – MIT; clarifies usage and liability.

## GitHub Settings (do manually)

### 1. Branch protection for `main`

**Settings → Branches → Add branch protection rule (or edit rule for `main`):**

- [ ] **Require a pull request before merging**
  - Require at least 1 approval (or 0 if you’re solo; you can tighten later).
  - Dismiss stale reviews when new commits are pushed (optional).
- [ ] **Require status checks to pass before merging**
  - Select **CI** (or the job name from `.github/workflows/ci.yml`) so PRs must pass before merge.
- [ ] **Do not allow bypassing the above settings** (no force-push / delete by others; keep “Allow force pushes” off for `main`).

### 2. Security

- [ ] **Security → Code security and analysis**
  - Enable **Dependency graph** (usually on for public repos).
  - Enable **Dependabot alerts** (uses `.github/dependabot.yml`).
  - Optionally enable **Dependabot security updates** for auto-PRs on known CVEs.
- [ ] **Security → Vulnerability reporting**
  - Enable **Private vulnerability reporting** so reporters can use the Security tab instead of public issues.

### 3. General

- [ ] **Settings → General**
  - Confirm **Collaborators** only includes people who should have write access.
  - Consider **Discussions** (on/off) and **Issues** (on) as you prefer.
- [ ] Ensure **`.env` is never committed** (it’s in `.gitignore`). If it was ever committed in the past, rotate all secrets and consider cleaning history (e.g. `git filter-repo` or BFG).

### 4. Optional

- **CODEOWNERS** (`.github/CODEOWNERS`): Require review from specific people for certain paths.
- **Pull request template** (`.github/PULL_REQUEST_TEMPLATE.md`): Checklist (e.g. “No secrets”, “Tests pass locally”).
- **Actions permissions**: Settings → Actions → General → set “Workflow permissions” to “Read repository contents” if you want minimal scope.

Once these are set, your code is protected by process (PRs + CI), dependency alerts, and a clear security and license policy.
