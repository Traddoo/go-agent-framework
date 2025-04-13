# Go Agent Framework Guidelines

## Build & Test Commands
- Build: `go build ./...`
- Run all tests: `go test ./...`
- Run single test: `go test -v github.com/traddoo/go-agent-framework/pkg/agent -run TestAgentName`
- Test with coverage: `go test -cover ./...`
- Format code: `go fmt ./...`
- Lint code: `golint ./...`
- Vet code: `go vet ./...`

## Code Style Guidelines
- **Imports**: Group standard library, third-party, and internal imports with blank lines between groups.
- **Naming**: Use PascalCase for exported symbols, camelCase for unexported. Use short but descriptive names.
- **Error Handling**: Use structured errors with codes. Return errors rather than using panics.
- **Type Conventions**: Define interfaces at consumer, implementations at provider. Use pointer receivers for mutable types.
- **Documentation**: All exported functions and types must have doc comments starting with the symbol name.
- **Concurrency**: Use mutexes for shared state. Document thread-safety assumptions for exported functions.
- **Context**: Pass context.Context as first parameter to all methods that perform I/O operations.
- **Testing**: Write table-driven tests. Mock external dependencies where appropriate.