# Contributing to Terraform Import Helper

Thank you for your interest in contributing to driftmgr! This document provides guidelines and information for contributors.

## Development Setup

### Prerequisites
- Go 1.21 or later
- Git
- Terraform (for testing import functionality)
- Cloud provider CLI tools (AWS CLI, Azure CLI, gcloud) for testing

### Getting Started

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/yourusername/driftmgr.git
   cd driftmgr
   ```

3. Install dependencies:
   ```bash
   make deps
   ```

4. Set up development environment:
   ```bash
   make dev-setup
   ```

5. Build the project:
   ```bash
   make build
   ```

6. Run tests:
   ```bash
   make test
   ```

## Project Structure

```
driftmgr/
├── cmd/                    # Application entry point
├── internal/
│   ├── cmd/               # CLI commands
│   ├── discovery/         # Resource discovery engine
│   ├── importer/          # Import orchestration
│   ├── models/            # Data models
│   └── tui/               # Terminal UI components
├── examples/              # Example files and configurations
├── docs/                  # Documentation
├── tests/                 # Test files
├── Makefile              # Build automation
├── go.mod                # Go module definition
└── README.md             # Project overview
```

## Code Style and Standards

### Go Code Style
- Follow standard Go formatting (`gofmt`)
- Use meaningful variable and function names
- Add comments for public functions and complex logic
- Keep functions small and focused
- Use interfaces for testability

### Commit Messages
Use conventional commit format:
```
type(scope): description

[optional body]

[optional footer]
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes
- `refactor`: Code refactoring
- `test`: Test additions or modifications
- `chore`: Maintenance tasks

Examples:
```
feat(discovery): add GCP resource discovery
fix(importer): handle terraform import errors gracefully
docs(readme): update installation instructions
```

## Testing

### Unit Tests
- Write unit tests for all public functions
- Use table-driven tests where appropriate
- Mock external dependencies
- Aim for >80% test coverage

```bash
# Run tests
make test

# Run tests with coverage
make test-coverage
```

### Integration Tests
- Test end-to-end workflows
- Use real cloud provider APIs in test environments
- Clean up resources after tests

### TUI Testing
- Test keyboard interactions
- Verify screen rendering
- Test responsive behavior

## Adding New Features

### Adding a New Cloud Provider

1. Create a new provider file in `internal/discovery/`
2. Implement the `Provider` interface
3. Add provider to the engine's provider map
4. Update CLI help and documentation
5. Add example configurations
6. Write comprehensive tests

### Adding New Resource Types

1. Update the provider's `SupportedResourceTypes()` method
2. Add discovery logic for the new resource type
3. Add terraform resource type mapping
4. Update configuration generation templates
5. Add tests for the new resource type

### Adding New CLI Commands

1. Create command file in `internal/cmd/`
2. Add command to root command in `init()`
3. Follow existing pattern for flags and configuration
4. Add help text and examples
5. Write tests for the command

### Adding New TUI Views

1. Create view model in `internal/tui/`
2. Implement the Bubble Tea model interface
3. Add view to the main app router
4. Follow existing styling patterns
5. Test keyboard interactions

## Documentation

### Code Documentation
- Add godoc comments for all public functions
- Include usage examples in documentation
- Document complex algorithms and business logic

### User Documentation
- Update README.md for user-facing changes
- Add or update command help text
- Create examples for new features
- Update configuration documentation

## Pull Request Process

1. **Create a Feature Branch**
   ```bash
   git checkout -b feat/your-feature-name
   ```

2. **Make Your Changes**
   - Write clean, well-tested code
   - Follow coding standards
   - Update documentation

3. **Test Your Changes**
   ```bash
   make check
   make test
   ```

4. **Commit Your Changes**
   ```bash
   git add .
   git commit -m "feat(scope): description"
   ```

5. **Push and Create PR**
   ```bash
   git push origin feat/your-feature-name
   ```

6. **PR Requirements**
   - Clear description of changes
   - Link to related issues
   - All tests passing
   - Code review approval
   - Up-to-date with main branch

## Release Process

### Version Numbering
- Follow semantic versioning (SemVer)
- Major: Breaking changes
- Minor: New features (backward compatible)
- Patch: Bug fixes

### Release Checklist
1. Update version in code
2. Update CHANGELOG.md
3. Create git tag
4. Build release binaries
5. Create GitHub release
6. Update documentation

## Getting Help

### Questions and Discussions
- Open a GitHub Discussion for questions
- Join our community chat (link TBD)
- Check existing issues and documentation

### Reporting Bugs
- Use GitHub Issues
- Include reproduction steps
- Provide system information
- Include relevant logs

### Feature Requests
- Open a GitHub Issue with the "enhancement" label
- Describe the use case and benefits
- Provide mockups or examples if applicable

## Code of Conduct

This project follows the [Contributor Covenant Code of Conduct](https://www.contributor-covenant.org/version/2/1/code_of_conduct/).

## License

By contributing to this project, you agree that your contributions will be licensed under the same license as the project (MIT License).

## Recognition

Contributors will be recognized in:
- CONTRIBUTORS.md file
- Release notes
- Project documentation

Thank you for contributing to driftmgr!
