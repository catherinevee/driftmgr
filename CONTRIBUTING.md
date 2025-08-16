# Contributing to DriftMgr

## Development Setup

1. Clone the repository
2. Install Go 1.21 or later
3. Run `make setup` to install dependencies
4. Run `make test` to verify everything works

## Code Style

- Follow Go formatting standards (`gofmt`)
- Use meaningful variable and function names
- Add comments for exported functions
- Keep functions small and focused

## Testing

- Write unit tests for new functionality
- Run `make test` before submitting PRs
- Ensure test coverage doesn't decrease

## Pull Request Process

1. Create a feature branch
2. Make your changes
3. Add tests
4. Update documentation
5. Submit a pull request

## Commit Messages

Use conventional commit format:
- `feat:` for new features
- `fix:` for bug fixes
- `docs:` for documentation
- `test:` for tests
- `refactor:` for refactoring
