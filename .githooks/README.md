# Git Hooks for Ustawka

This directory contains git hooks to maintain code quality and enforce development workflows.

## Available Hooks

### pre-commit
- **Purpose**: Ensures code quality before commits
- **Features**:
  - Prevents direct commits to protected branches (`master`, `main`, `RELEASE`)
  - Runs `make check` to verify linting and tests pass
  - Provides helpful error messages and suggestions

## Installation

### Automatic Setup (Recommended)
```bash
# From project root
./scripts/setup-hooks.sh
```

### Manual Setup
```bash
# Copy the hook to your local git hooks directory
cp .githooks/pre-commit .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit
```

## Usage

Once installed, the hooks run automatically:

```bash
# This will trigger the pre-commit hook
git commit -m "your commit message"
```

### Protected Branches
The pre-commit hook prevents direct commits to:
- `master`
- `main` 
- `RELEASE`

If you try to commit to these branches, you'll see:
```
❌ ERROR: Direct commits to 'master' branch are not allowed!
💡 Please use a feature branch and create a pull request instead.
```

### Code Quality Checks
The hook runs `make check` which includes:
- Go linting (`golangci-lint`)
- Unit tests
- Code formatting verification

If checks fail, you'll see detailed error messages and suggestions for fixes.

## Recommended Workflow

1. **Create a feature branch**:
   ```bash
   git checkout -b feat/your-feature-name
   ```

2. **Make your changes and commit**:
   ```bash
   git add .
   git commit -m "feat: add your feature description"
   ```

3. **Push and create PR**:
   ```bash
   git push -u origin feat/your-feature-name
   # Create pull request via GitHub/GitLab
   ```

## Bypassing Hooks (Emergency Only)

In rare cases where you need to bypass the hook:
```bash
git commit --no-verify -m "emergency commit"
```

**⚠️ Warning**: Only use `--no-verify` in true emergencies. The hooks exist to maintain code quality.

## Troubleshooting

### Hook not running
- Ensure the hook file is executable: `chmod +x .git/hooks/pre-commit`
- Check that you're in the project root directory
- Verify the hook file exists in `.git/hooks/pre-commit`

### Make check failures
- Run `make check` manually to see detailed errors
- Common fixes:
  - `go fmt ./...` for formatting issues
  - `goimports -w .` for import organization
  - Fix test failures shown in output

### Missing dependencies
- Ensure you have all required tools:
  - `golangci-lint` for linting
  - `gotestsum` for test execution (optional)
  - Go toolchain properly installed

## Contributing

When adding new hooks:
1. Add the hook file to `.githooks/`
2. Update this README
3. Update `scripts/setup-hooks.sh` if needed
4. Test the hook thoroughly before committing