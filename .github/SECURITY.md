# Security Policy

## Supported Versions

We release security updates for the current main branch. No Long-Term Support (LTS) branches are maintained at this time.

## Reporting a Vulnerability

**Please do not report security vulnerabilities in public issues.**

To report a security issue:

1. Go to this repository’s **Security** tab on GitHub.
2. Click **Report a vulnerability** (or use [GitHub Security Advisories](https://github.com/kartikbazzad/bunbase/security/advisories/new)).
3. Describe the issue and steps to reproduce.

We will acknowledge your report and work on a fix. We ask that you allow time for a patch before any public disclosure.

## What we consider in scope

- BunBase services (BunAuth, Bundoc, Functions, BunKMS, Platform API, Bunder, etc.)
- Authentication, authorization, or data isolation bugs
- Secret or credential handling
- Injection or deserialization issues in our code or documented APIs

Out-of-scope: issues in third-party services (Postgres, MinIO, Traefik) that are not caused by our configuration or code.
