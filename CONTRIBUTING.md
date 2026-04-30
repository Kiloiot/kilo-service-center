# Contributing to KiloCenter

Thank you for your interest in contributing to KiloCenter. This guide covers how to report issues, propose changes, and submit code.

## Code of Conduct

This project follows the [Contributor Covenant Code of Conduct](CODE_OF_CONDUCT.md). By participating, you agree to uphold its terms.

## Reporting Bugs

Open a bug report using the [Bug Report template](https://github.com/Kiloiot/KiloServiceCenter/issues/new?template=bug_report.yml) on GitHub Issues. Include reproduction steps, expected vs actual behavior, and relevant logs.

## Suggesting Features

Open a feature request using the [Feature Request template](https://github.com/Kiloiot/KiloServiceCenter/issues/new?template=feature_request.yml) on GitHub Issues. Describe the use case, not just the solution.

## Security Vulnerabilities

Do not open public issues for security vulnerabilities. See [SECURITY.md](SECURITY.md) for responsible disclosure instructions.

## Development Setup

Refer to the [Getting Started](README.md#getting-started) section in the README and the full documentation at [docs.kiloiot.io](https://docs.kiloiot.io) for environment setup, dependencies, and running the stack locally.

## Submitting Pull Requests

1. Fork the repository and clone your fork.
2. Create a feature branch from `main`:
   ```bash
   git checkout -b feat/your-feature main
   ```
3. Make your changes, ensuring tests pass and code style checks are clean.
4. Push your branch and open a pull request against `main`.
5. Fill out the pull request template with a clear description of the change.

Keep pull requests focused on a single concern. If your change touches multiple areas, consider splitting it into separate PRs.

## Code Style

### Go

- Format with `gofmt` (enforced by CI).
- Lint with `golangci-lint`. Run `make lint` from the repo root.

### TypeScript (KC-Web)

- Lint with ESLint: `bun run lint` in `KC-Web/`.
- Format with Prettier: `bun run format:check` in `KC-Web/`.

## Commit Messages

Follow the conventional commit format:

```
type(scope): description
```

Types: `feat`, `fix`, `docs`, `chore`, `refactor`, `test`

Examples:
- `feat(bssci): add session resume timeout handling`
- `fix(web): correct base station status display`
- `docs(readme): update deployment instructions`

Keep the subject line under 72 characters. Use the body for additional context when needed.

## Testing

Run tests before submitting a pull request:

```bash
# Go (from repo root)
go test ./...

# Frontend (from KC-Web/)
bun run test
```

CI runs the full test suite on every pull request. PRs with failing tests will not be merged.

## License

KiloCenter is licensed under [AGPL-3.0](LICENSE). By submitting a contribution, you agree that your work will be licensed under the same terms. If your employer holds copyright over your work, ensure you have permission to contribute under this license.
