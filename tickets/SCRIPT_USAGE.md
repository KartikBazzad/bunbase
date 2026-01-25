# GitHub Issues Creation Script

## Fixed Issues

The script has been updated to:

1. **Prevent duplicates**: Checks if an issue with the same title already exists before creating
2. **Exclude non-ticket files**: Skips README.md and GITHUB_ISSUES.md files
3. **Better error handling**: Handles missing labels gracefully

## Usage

```bash
./create-github-issues.sh
```

## What It Does

1. Finds all ticket files (excluding README.md and GITHUB_ISSUES.md)
2. For each ticket:
   - Extracts the title from the first line
   - Checks if an issue with that title already exists
   - If exists: Skips and shows "⊘ Skipped"
   - If not exists: Creates the issue
3. Shows progress and summary

## Output

The script will show:

- `[X/52] Processing: filename.md` - Current progress
- `✓ Created: https://github.com/.../issues/N` - Successfully created
- `⊘ Skipped: Issue already exists` - Duplicate detected and skipped
- `✗ Failed: error message` - Failed to create

## Notes

- Issues are created without labels (labels can be added manually later)
- The script includes a 0.5s delay between requests to avoid rate limits
- All ticket content is included in the issue body
- The script is idempotent - safe to run multiple times

## Troubleshooting

If you see TLS certificate errors:

- This is a local environment issue
- The script will still work, but may show warnings
- Issues are being created successfully despite the warnings
