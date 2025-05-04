# Ustawka Project Map (Enhanced)

## Project Overview
A Go application that fetches and displays legislative acts from the Polish Sejm API in a Kanban-style board.

## AI Assistant Rules for Using This Map
1. **Context Reference**:
   - Always check this map before making changes
   - Use it to understand the project's architecture and constraints
   - Reference performance characteristics when optimizing
   - Follow established patterns for error handling

2. **Change Guidelines**:
   - Maintain consistency with existing architecture
   - Respect performance characteristics and limits
   - Follow established error handling patterns
   - Update the map when making significant changes

3. **Decision Making**:
   - Use documented response times as benchmarks
   - Consider known issues when implementing fixes
   - Reference testing strategy for new features
   - Align with future improvements roadmap

4. **Documentation Updates**:
   - Update performance characteristics if they change
   - Add new known issues as they are discovered
   - Document new patterns and solutions
   - Keep future improvements list current

## Architecture & Components

### 1. API Integration (`sejm/`)
- **Client Interface**:
  ```go
  type SejmClient interface {
      GetActs(ctx context.Context, year int) ([]sejm.Act, error)
      GetActDetails(ctx context.Context, actID string) (*sejm.ActDetails, error)
  }
  ```
- **API Endpoints**:
  - Base URL: `https://api.sejm.gov.pl/eli/acts/DU/{year}`
  - Response times observed:
    - 2025: ~100-200ms
    - 2024: ~600-1500ms
    - 2021: ~1500ms

### 2. Cache Layer (`db/`)
- **Database**: SQLite
- **Tables**:
  - `acts`: Cached acts by year
  - `act_details`: Cached act details
- **Features**:
  - 24-hour cache expiration
  - Automatic cache updates
  - Transaction support
  - Indexed queries

### 3. Service Layer (`service/`)
- **Core Components**:
  ```go
  type ActService struct {
      sejmClient SejmClient
      db         *db.DB
      timeout    time.Duration
  }
  ```
- **Key Features**:
  - Configurable timeout via `SEJM_API_TIMEOUT`
  - Default: 5s
  - Docker default: 15s
  - Context management for concurrent requests
  - Cache-first data retrieval
- **Data Organization**:
  ```go
  type KanbanData struct {
      Obowiazujace []sejm.Act  // Active acts
      Pending      []sejm.Act  // In progress
      Uchylone     []sejm.Act  // Repealed
  }
  ```

### 4. HTTP Layer
- **Server**:
  - Port: 8080
  - Framework: Chi router
  - Response times:
    - Static files: < 1ms
    - API endpoints: 1-4s
- **Endpoints**:
  ```
  GET /                    # Main page (4.4KB)
  GET /static/css/style.css
  GET /api/years          # Available years
  GET /api/acts/DU/{year} # Acts for year
  GET /api/acts/DU/{year}/{position} # Act details
  ```

### 5. Frontend
- **Technologies**:
  - HTMX for dynamic updates
  - CSS for styling
  - Responsive design
- **Features**:
  - Real-time updates
  - Kanban board layout
  - Year selection dropdown

### 6. Infrastructure
- **Docker**:
  ```dockerfile
  # Multi-stage build
  FROM golang:1.24.2-alpine AS builder
  FROM gcr.io/distroless/static-debian12:nonroot
  ```
- **Environment Variables**:
  - `SEJM_API_TIMEOUT`: Request timeout
    - Default: 5s
    - Docker: 15s
  - `SEJM_DB_PATH`: Database path
    - Default: sejm.db
    - Docker: /app/data/sejm.db

## Performance Characteristics
1. **Response Times**:
   - Static content: < 1ms
   - API calls: 100ms - 4s
   - Year checks: 1-3s
   - Cached responses: < 100ms

2. **Error Patterns**:
   - Context cancellation errors
   - Timeout issues with multiple concurrent requests
   - Response body reading errors
   - Cache read/write errors

3. **Data Sizes**:
   - 2025: ~434KB
   - 2024: ~1.4MB
   - 2021: ~1.8MB
   - SQLite database: ~10-20MB

## Testing Strategy
1. **Unit Tests**:
   - Service layer with mocks
   - Client layer with test responses
   - Database layer with SQLite
   - Error handling scenarios

2. **Integration Tests**:
   - Real API tests (skipped in short mode)
   - End-to-end testing
   - Cache integration tests

3. **Automation**:
   - Cursor rules for test execution
   - Makefile targets

## Known Issues & Patterns
1. **Context Cancellation**:
   ```
   Error checking year: error reading response body: context canceled
   Error checking year: error fetching acts: Get "https://api.sejm.gov.pl/eli/acts/DU/2023": context canceled
   ```

2. **Performance Bottlenecks**:
   - Year checking takes 1-3s
   - Large response sizes for older years
   - Concurrent request handling
   - Cache updates during high load

## Future Improvements
1. **Performance**:
   - Implement caching (✓)
   - Optimize concurrent requests
   - Add request rate limiting
   - Cache warming strategies

2. **Error Handling**:
   - Better timeout management
   - Retry mechanisms
   - Circuit breaker pattern
   - Cache error recovery

3. **Monitoring**:
   - Add metrics collection
   - Implement health checks
   - Performance monitoring
   - Cache hit/miss tracking

4. **Documentation**:
   - API documentation
   - Deployment guides
   - Development guidelines
   - Cache management guide 
