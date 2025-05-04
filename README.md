# Ustawka

A web application for tracking Polish legislative acts from the Sejm API.

## Features

- View legislative acts organized in a Kanban board
- Filter acts by year (2021-present)
- Categorize acts by status:
  - In preparation
  - Repealed
  - In force
- View detailed information about each act

## Tech Stack

- Backend: Go
- Frontend: HTML, TailwindCSS, HTMX
- API: Sejm API (https://api.sejm.gov.pl)

## Prerequisites

- Go 1.24.2 or later
- Modern web browser
- Make (optional, for using Makefile)

## Installation

1. Clone the repository:
```bash
git clone https://github.com/bmcszk/ustawka.git
cd ustawka
```

2. Install dependencies:
```bash
make deps
```

3. Run the application:
```bash
make run
```

The application will be available at http://localhost:8080

## Development

### Using Makefile

The project includes a Makefile with common development tasks:

```bash
make build      # Build the application
make run        # Run the application
make test       # Run all tests
make test-unit  # Run unit tests only
make test-e2e   # Run end-to-end tests only
make clean      # Clean build files
make deps       # Install dependencies
make help       # Show all available commands
```

### Testing

The project includes two types of tests:
- Unit tests: Test individual components in isolation (marked with `testing.Short()`)
- End-to-end tests: Test the application with the real Sejm API

To run specific test types:
```bash
make test-unit  # Run only unit tests (marked with testing.Short())
make test-e2e   # Run only end-to-end tests
make test       # Run all tests
```

To mark a test as a unit test, use `testing.Short()`:
```go
func TestSomething(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping test in short mode")
    }
    // ... test code ...
}
```

### Project Structure

```
ustawka/
├── handlers/     # HTTP request handlers
├── server/       # Server configuration
├── sejm/         # Sejm API client
├── static/       # Static assets
├── templates/    # HTML templates
└── main.go       # Application entry point
```

### Running Tests

```bash
go test ./...
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request 
