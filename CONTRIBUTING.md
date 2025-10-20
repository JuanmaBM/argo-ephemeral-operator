# Contributing to argo-ephemeral-operator

Thank you for your interest in contributing! This document provides guidelines and instructions for contributing to the project.

## Development Principles

This project follows these core principles:

### SOLID Principles

1. **Single Responsibility Principle (SRP)**: Each component has one clear responsibility
   - Controllers handle reconciliation logic
   - ArgoCD client handles ArgoCD interactions
   - Config package handles configuration

2. **Open/Closed Principle (OCP)**: Open for extension, closed for modification
   - Use interfaces for extensibility
   - Add new features through composition

3. **Liskov Substitution Principle (LSP)**: Subtypes must be substitutable for their base types
   - Interface implementations are interchangeable
   - Mock implementations for testing

4. **Interface Segregation Principle (ISP)**: Many specific interfaces over one general interface
   - `ArgoCD Client` interface is focused and minimal
   - `NameGenerator` interface is single-purpose

5. **Dependency Inversion Principle (DIP)**: Depend on abstractions, not concretions
   - Controller depends on ArgoCD Client interface, not implementation
   - Enables easy testing and mocking

### DRY (Don't Repeat Yourself)

- Shared logic is extracted into reusable functions
- Configuration is centralized
- Common patterns are abstracted

### YAGNI (You Aren't Gonna Need It)

- Implement only what's needed now
- Avoid speculative features
- Keep the codebase lean and focused

## Code Style

- Follow standard Go conventions and idioms
- Run `go fmt` before committing
- Run `go vet` to catch common issues
- Use meaningful variable and function names
- Add comments for exported functions and types

## Testing

- Write unit tests for new functionality
- Aim for good test coverage
- Use table-driven tests where appropriate
- Mock external dependencies using interfaces

Example test structure:

```go
func TestReconcile(t *testing.T) {
    tests := []struct {
        name    string
        input   *v1alpha1.EphemeralApplication
        want    ctrl.Result
        wantErr bool
    }{
        // test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}
```

## Pull Request Process

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes following the principles above
4. Run tests: `make test`
5. Run formatting: `make fmt`
6. Run linter: `make vet`
7. Commit your changes with clear, descriptive messages
8. Push to your fork
9. Open a Pull Request with a clear description of changes

## Commit Messages

Follow conventional commits format:

- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation changes
- `refactor:` Code refactoring
- `test:` Adding or updating tests
- `chore:` Maintenance tasks

Example: `feat: add support for custom sync options`

## Questions?

Feel free to open an issue for discussion or questions.

