# CLI Tool Requirements

## Overview

The BunBase CLI provides a command-line interface for managing BunBase projects, deploying functions, managing databases, and automating development workflows.

## Installation

### Package Managers

```bash
# npm
npm install -g @bunbase/cli

# yarn
yarn global add @bunbase/cli

# pnpm
pnpm add -g @bunbase/cli

# Homebrew (macOS/Linux)
brew install bunbase

# Scoop (Windows)
scoop install bunbase

# Direct download
curl -fsSL https://bunbase.io/install.sh | sh
```

### Verification

```bash
bunbase --version
bunbase --help
```

## Core Commands

### 1. Authentication & Initialization

#### Login

```bash
# Login to BunBase account
bunbase login

# Login with API key
bunbase login --api-key bunbase_pk_live_xxx

# Login to specific region
bunbase login --region us-east-1

# Logout
bunbase logout
```

#### Initialize Project

```bash
# Initialize new project
bunbase init

# Initialize with template
bunbase init --template nextjs
bunbase init --template express
bunbase init --template python-fastapi

# Initialize in current directory
bunbase init .

# Non-interactive mode
bunbase init --name my-project --region us-east-1 -y
```

### 2. Project Management

#### List Projects

```bash
# List all projects
bunbase projects list

# List with details
bunbase projects list --detailed

# Output as JSON
bunbase projects list --json
```

#### Create Project

```bash
# Create new project
bunbase projects create my-project

# Create with options
bunbase projects create my-project \
  --region us-east-1 \
  --plan pro \
  --description "My awesome project"
```

#### Delete Project

```bash
# Delete project
bunbase projects delete my-project

# Force delete without confirmation
bunbase projects delete my-project --force
```

#### Switch Project

```bash
# Switch active project
bunbase projects switch my-project

# Set default project
bunbase projects default my-project
```

### 3. Database Management

#### Collections/Tables

```bash
# List collections
bunbase db list

# Create collection
bunbase db create users --schema schema.json

# Delete collection
bunbase db delete users

# Describe collection
bunbase db describe users

# Export collection schema
bunbase db schema users > users-schema.json
```

#### Data Operations

```bash
# Query data
bunbase db query users "SELECT * FROM users WHERE age > 18"

# Import data
bunbase db import users data.json
bunbase db import users data.csv --format csv

# Export data
bunbase db export users > users.json
bunbase db export users --format csv > users.csv

# Truncate collection
bunbase db truncate users --force
```

#### Migrations

```bash
# Create migration
bunbase db migrate create add-users-table

# Run migrations
bunbase db migrate up

# Rollback migration
bunbase db migrate down

# Migration status
bunbase db migrate status

# Reset database
bunbase db migrate reset --force
```

#### Indexes

```bash
# List indexes
bunbase db indexes users

# Create index
bunbase db index create users email --unique

# Create compound index
bunbase db index create users "firstName,lastName"

# Delete index
bunbase db index delete users email_idx
```

### 4. Storage Management

#### Buckets

```bash
# List buckets
bunbase storage buckets

# Create bucket
bunbase storage create-bucket images --public

# Delete bucket
bunbase storage delete-bucket images --force

# Bucket info
bunbase storage info images
```

#### File Operations

```bash
# Upload file
bunbase storage upload avatars ./avatar.jpg public/user-123.jpg

# Upload directory
bunbase storage upload images ./local-images/ remote-images/ --recursive

# Download file
bunbase storage download avatars public/user-123.jpg ./downloaded.jpg

# Download directory
bunbase storage download images remote-images/ ./local-images/ --recursive

# List files
bunbase storage ls avatars
bunbase storage ls avatars public/ --recursive

# Delete file
bunbase storage rm avatars public/user-123.jpg

# Delete directory
bunbase storage rm avatars public/ --recursive

# Copy file
bunbase storage cp avatars public/old.jpg public/new.jpg

# Move file
bunbase storage mv avatars public/old.jpg public/new.jpg

# Sync directories
bunbase storage sync ./local-dir avatars/remote-dir/
```

#### Signed URLs

```bash
# Generate signed URL
bunbase storage signed-url private-files contract.pdf --expires 3600

# Batch signed URLs
bunbase storage signed-urls private-files file1.pdf file2.pdf --expires 7200
```

### 5. Functions Management

#### List Functions

```bash
# List all functions
bunbase functions list

# List with details
bunbase functions list --detailed
```

#### Deploy Functions

```bash
# Deploy function
bunbase functions deploy my-function

# Deploy from directory
bunbase functions deploy my-function --source ./functions/my-function

# Deploy with environment variables
bunbase functions deploy my-function --env KEY1=value1 --env KEY2=value2

# Deploy all functions
bunbase functions deploy --all

# Deploy with config file
bunbase functions deploy --config functions.yaml
```

#### Invoke Function

```bash
# Invoke function
bunbase functions invoke my-function

# Invoke with data
bunbase functions invoke my-function --data '{"key":"value"}'

# Invoke with file input
bunbase functions invoke my-function --data @input.json

# Invoke and save output
bunbase functions invoke my-function > output.json
```

#### Function Logs

```bash
# View logs
bunbase functions logs my-function

# Tail logs (real-time)
bunbase functions logs my-function --tail

# Filter logs
bunbase functions logs my-function --level error --since 1h

# Export logs
bunbase functions logs my-function --since 24h > logs.txt
```

#### Function Development

```bash
# Create function
bunbase functions create my-function --runtime nodejs20

# Run function locally
bunbase functions dev my-function

# Test function
bunbase functions test my-function --event test-event.json

# Delete function
bunbase functions delete my-function
```

### 6. Authentication Management

#### Users

```bash
# List users
bunbase auth users

# Get user
bunbase auth user user@example.com

# Create user
bunbase auth create-user user@example.com --password secret123

# Delete user
bunbase auth delete-user user@example.com

# Disable user
bunbase auth disable-user user@example.com

# Enable user
bunbase auth enable-user user@example.com
```

#### Roles & Permissions

```bash
# List roles
bunbase auth roles

# Create role
bunbase auth create-role admin --permissions "db:*,storage:*"

# Assign role
bunbase auth assign-role user@example.com admin

# Revoke role
bunbase auth revoke-role user@example.com admin
```

### 7. API Keys Management

```bash
# List API keys
bunbase keys list

# Create API key
bunbase keys create production --permissions "db:read,db:write"

# Create API key with expiry
bunbase keys create temp-key --expires 30d

# Revoke API key
bunbase keys revoke key_abc123

# Rotate API key
bunbase keys rotate key_abc123
```

### 8. Secrets Management

```bash
# Set secret
bunbase secrets set API_KEY=xxx

# Set from file
bunbase secrets set DATABASE_URL=@.env

# List secrets
bunbase secrets list

# Get secret
bunbase secrets get API_KEY

# Delete secret
bunbase secrets delete API_KEY

# Bulk import
bunbase secrets import .env
```

### 9. Environment Management

```bash
# List environments
bunbase env list

# Create environment
bunbase env create staging

# Switch environment
bunbase env use production

# Delete environment
bunbase env delete development

# Copy environment variables
bunbase env copy production staging
```

### 10. Monitoring & Logs

```bash
# View project logs
bunbase logs --tail

# Filter by service
bunbase logs --service database --tail

# Filter by level
bunbase logs --level error --since 1h

# Export logs
bunbase logs --since 24h --format json > logs.json
```

### 11. Type Generation

```bash
# Generate TypeScript types from database schema
bunbase types generate

# Generate for specific collection
bunbase types generate users

# Output to file
bunbase types generate --output ./types/database.ts

# Watch mode
bunbase types generate --watch
```

### 12. Development

#### Local Development

```bash
# Start local development server
bunbase dev

# Start with specific port
bunbase dev --port 3000

# Start with environment
bunbase dev --env .env.local
```

#### Link/Unlink

```bash
# Link local project to BunBase project
bunbase link my-project

# Unlink project
bunbase unlink
```

### 13. Deployment

```bash
# Deploy entire project
bunbase deploy

# Deploy with tag
bunbase deploy --tag v1.0.0

# Deploy to specific environment
bunbase deploy --env production

# Deploy with confirmation
bunbase deploy --confirm

# Rollback deployment
bunbase rollback

# Rollback to specific version
bunbase rollback --version v1.0.0
```

### 14. Status & Info

```bash
# Project status
bunbase status

# Detailed info
bunbase info

# Check health
bunbase health

# Usage statistics
bunbase usage

# Usage for specific period
bunbase usage --since 30d
```

## Configuration

### Config File

```yaml
# bunbase.config.yaml
project: my-project
region: us-east-1
environment: production

functions:
  runtime: nodejs20
  memory: 512
  timeout: 30

database:
  defaultCollection: users

storage:
  defaultBucket: uploads

auth:
  providers:
    - google
    - github
```

### Global Config

```bash
# Set config value
bunbase config set default-region us-east-1

# Get config value
bunbase config get default-region

# List all config
bunbase config list

# Reset config
bunbase config reset
```

## Output Formats

### JSON Output

```bash
# Most commands support --json flag
bunbase projects list --json
bunbase db query users "SELECT *" --json
bunbase functions list --json
```

### Table Output

```bash
# Default table format
bunbase projects list

# Compact table
bunbase projects list --compact

# Wide format (no truncation)
bunbase projects list --wide
```

## Advanced Features

### Aliases

```bash
# Create command alias
bunbase alias create deploy-prod "deploy --env production"

# Use alias
bunbase deploy-prod

# List aliases
bunbase alias list

# Delete alias
bunbase alias delete deploy-prod
```

### Scripts

```bash
# Define scripts in bunbase.config.yaml
scripts:
  deploy-all: |
    bunbase functions deploy --all
    bunbase db migrate up
    bunbase deploy

# Run script
bunbase run deploy-all
```

### Plugins

```bash
# Install plugin
bunbase plugins install @bunbase/plugin-analytics

# List plugins
bunbase plugins list

# Uninstall plugin
bunbase plugins uninstall @bunbase/plugin-analytics

# Update plugin
bunbase plugins update @bunbase/plugin-analytics
```

### Autocomplete

```bash
# Enable shell autocomplete
bunbase completion bash >> ~/.bashrc
bunbase completion zsh >> ~/.zshrc
bunbase completion fish > ~/.config/fish/completions/bunbase.fish
```

## CI/CD Integration

### GitHub Actions

```yaml
# .github/workflows/deploy.yml
name: Deploy to BunBase
on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: bunbase/setup-cli@v1
      - run: bunbase login --api-key ${{ secrets.BUNBASE_API_KEY }}
      - run: bunbase deploy
```

### GitLab CI

```yaml
# .gitlab-ci.yml
deploy:
  image: bunbase/cli:latest
  script:
    - bunbase login --api-key $BUNBASE_API_KEY
    - bunbase deploy
  only:
    - main
```

## Error Handling

### Verbose Mode

```bash
# Enable verbose output
bunbase --verbose deploy

# Debug mode
bunbase --debug functions invoke my-function
```

### Error Codes

- Exit 0: Success
- Exit 1: General error
- Exit 2: Authentication error
- Exit 3: Permission error
- Exit 4: Not found error
- Exit 5: Validation error

## Performance Features

- Command caching
- Parallel operations
- Progress indicators
- Compression for uploads
- Resumable operations
- Connection pooling

## Documentation Requirements

- Getting started guide
- Command reference (auto-generated)
- CI/CD integration guide
- Migration guide
- Troubleshooting guide
- Video tutorials
- Example workflows
