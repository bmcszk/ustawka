# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Essential Commands

```bash
# Development workflow
make check          # Run linters and unit tests (required before commits)
make run           # Start development server on :8080
make build         # Build binary

# Testing
make test-unit     # Unit tests only (marked with testing.Short())
make test-e2e      # End-to-end tests (real Sejm API calls)
make test          # All tests

# Dependencies
make deps          # Install Go module dependencies
```

## Architecture Overview

**Ustawka** is a web application for tracking Polish legislative acts from the Sejm API, presenting them in a Kanban-style board.

### Core Architecture
- **Clean layered architecture** with dependency injection
- **Cache-first strategy** with configurable TTL (24h default)
- **Dual response format**: JSON APIs + HTMX template responses
- **SQLite database** with intelligent caching layer

### Key Components
- `server/`: Chi router setup and middleware configuration
- `handlers/`: HTTP handlers supporting both JSON and HTMX responses  
- `service/`: Business logic layer with caching and external API integration
- `sejm/`: Sejm API client with structured data models
- `db/`: SQLite database layer with automatic timestamps and JSON storage
- `templates/`: HTMX-enabled HTML templates (base, board, act_details)

### Data Flow
```
Browser → Chi Router → Handlers → Service Layer → [Cache/DB ↔ Sejm API]
```

## Important Patterns

### Testing Strategy
- **Unit tests**: Use `testing.Short()` for mock-based isolation tests
- **E2E tests**: Prefix with `TestRealAPI` for integration with real Sejm API
- **Mocking**: Use testify/mock for service dependencies

### Database
- Uses UPSERT patterns for cache updates
- JSON storage for complex fields (references, texts)
- Automatic indexing on year/status for board queries

### Frontend Integration
- **HTMX + TailwindCSS** for interactive UI without heavy JavaScript
- Templates support both full page loads and HTMX partial updates
- Year filtering via query parameters

### Configuration
Key environment variables:
- `USTAWKA_PORT`: Server port (default: 8080)
- `SEJM_DB_PATH`: Database file path
- `SEJM_API_TIMEOUT`: External API timeout
- `SEJM_CACHE_TTL`: Cache expiration time

## Development Notes

- Always run `make check` before commits (linting + unit tests required)
- Service layer implements timeout management for external API calls
- Metrics tracking available at `/metrics` endpoint
- Database schema automatically handles migrations via triggers