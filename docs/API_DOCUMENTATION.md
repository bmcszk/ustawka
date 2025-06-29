# API Documentation
## Ustawka - Polish Legislative Tracking System

### Document Information
- **API Version**: v1.0
- **Last Updated**: June 28, 2025
- **Base URL**: `https://api.ustawka.gov.pl/api/v1`
- **Content Type**: `application/json`
- **Authentication**: Not required for public endpoints

---

## Table of Contents

1. [Overview](#1-overview)
2. [Authentication](#2-authentication)
3. [Response Format](#3-response-format)
4. [Error Handling](#4-error-handling)
5. [Rate Limiting](#5-rate-limiting)
6. [Acts API](#6-acts-api)
7. [Search API](#7-search-api)
8. [Export API](#8-export-api)
9. [Comparison API](#9-comparison-api)
10. [System API](#10-system-api)
11. [WebSocket API](#11-websocket-api)
12. [SDKs and Examples](#12-sdks-and-examples)

---

## 1. Overview

The Ustawka API provides programmatic access to Polish legislative data, including parliamentary acts, voting records, and detailed tracking information. The API follows REST principles and returns JSON responses.

### 1.1 Key Features

- **Real-time Data**: Live synchronization with official Sejm and Senate APIs
- **Comprehensive Search**: Advanced filtering and full-text search capabilities
- **Export Functionality**: Multiple export formats (JSON, CSV, PDF)
- **Comparison Tools**: Side-by-side analysis of legislative acts
- **Performance Optimized**: Cached responses with sub-second response times

### 1.2 Data Sources

- **Primary**: Sejm API (`api.sejm.gov.pl`)
- **Secondary**: Senate API (`senat.gov.pl/api`)
- **Update Frequency**: Every 30 minutes for incremental updates
- **Data Retention**: Complete historical data from 1989

---

## 2. Authentication

### 2.1 Public Access

Most endpoints are publicly accessible without authentication:

```bash
curl -X GET "https://api.ustawka.gov.pl/api/v1/acts/2024"
```

### 2.2 API Key Authentication (Future)

For advanced features and higher rate limits:

```bash
curl -X GET "https://api.ustawka.gov.pl/api/v1/premium/analytics" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### 2.3 Rate Limiting

- **Anonymous Users**: 100 requests per minute
- **Authenticated Users**: 1000 requests per minute
- **Headers**: Rate limit info included in response headers

---

## 3. Response Format

### 3.1 Standard Response Structure

All API responses follow a consistent format:

```json
{
  "success": true,
  "data": {
    // Response data here
  },
  "meta": {
    "timestamp": "2025-06-28T14:30:00Z",
    "duration": "0.045s",
    "page": 1,
    "limit": 50,
    "total": 1234
  }
}
```

### 3.2 Error Response Structure

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid year parameter",
    "details": {
      "field": "year",
      "value": "invalid",
      "expected": "integer between 1989 and 2025"
    }
  },
  "meta": {
    "timestamp": "2025-06-28T14:30:00Z",
    "duration": "0.012s"
  }
}
```

### 3.3 Response Headers

```http
Content-Type: application/json; charset=utf-8
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1640995200
X-Response-Time: 45ms
Cache-Control: public, max-age=3600
ETag: "33a64df551"
```

---

## 4. Error Handling

### 4.1 HTTP Status Codes

| Status | Code | Description |
|--------|------|-------------|
| 200 | OK | Request successful |
| 400 | Bad Request | Invalid request parameters |
| 401 | Unauthorized | Authentication required |
| 403 | Forbidden | Insufficient permissions |
| 404 | Not Found | Resource not found |
| 429 | Too Many Requests | Rate limit exceeded |
| 500 | Internal Server Error | Server error |
| 502 | Bad Gateway | External API error |
| 503 | Service Unavailable | Service temporarily unavailable |

### 4.2 Error Codes

| Code | Description |
|------|-------------|
| `INVALID_REQUEST` | Malformed request |
| `VALIDATION_ERROR` | Parameter validation failed |
| `NOT_FOUND` | Resource not found |
| `RATE_LIMIT_EXCEEDED` | Too many requests |
| `EXTERNAL_API_ERROR` | External service error |
| `DATABASE_ERROR` | Internal database error |
| `TIMEOUT_ERROR` | Request timeout |

---

## 5. Rate Limiting

### 5.1 Limits

- **Default**: 100 requests per minute per IP
- **Burst**: Up to 10 requests per second
- **Window**: 60-second sliding window

### 5.2 Headers

```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1640995200
```

### 5.3 Exceeded Response

```json
{
  "success": false,
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Rate limit exceeded. Try again in 60 seconds.",
    "details": {
      "limit": 100,
      "remaining": 0,
      "reset_time": "2025-06-28T14:31:00Z"
    }
  }
}
```

---

## 6. Acts API

### 6.1 Get Available Years

Get list of years with available legislative data.

**Endpoint**: `GET /acts/years`

**Response**:
```json
{
  "success": true,
  "data": {
    "years": [2020, 2021, 2022, 2023, 2024, 2025],
    "current_year": 2025,
    "total_acts": 45678
  },
  "meta": {
    "timestamp": "2025-06-28T14:30:00Z",
    "duration": "0.023s"
  }
}
```

**Example**:
```bash
curl -X GET "https://api.ustawka.gov.pl/api/v1/acts/years"
```

### 6.2 Get Acts by Year

Retrieve all acts for a specific year.

**Endpoint**: `GET /acts/{year}`

**Parameters**:
- `year` (required): Year (1989-2025)
- `enhanced` (optional): Include enhanced data (default: false)
- `limit` (optional): Number of results (default: 50, max: 500)
- `offset` (optional): Pagination offset (default: 0)

**Response**:
```json
{
  "success": true,
  "data": {
    "acts": [
      {
        "id": "DU/2024/1",
        "title": "Ustawa o zmianie ustawy o podatku dochodowym",
        "status": "obowiązujący",
        "published": true,
        "position": 1,
        "year": 2024,
        "type": "ustawa",
        "address": "https://sejm.gov.pl/sejm10.nsf/druk.xsp?nr=1",
        "created_at": "2024-01-15T10:30:00Z",
        "updated_at": "2024-01-15T10:30:00Z"
      }
    ],
    "year": 2024,
    "total_count": 1234
  },
  "meta": {
    "page": 1,
    "limit": 50,
    "total": 1234,
    "timestamp": "2025-06-28T14:30:00Z",
    "duration": "0.156s"
  }
}
```

**Examples**:
```bash
# Basic request
curl -X GET "https://api.ustawka.gov.pl/api/v1/acts/2024"

# With enhanced data
curl -X GET "https://api.ustawka.gov.pl/api/v1/acts/2024?enhanced=true"

# With pagination
curl -X GET "https://api.ustawka.gov.pl/api/v1/acts/2024?limit=100&offset=200"
```

### 6.3 Get Enhanced Acts

Retrieve acts with enriched data including voting records and timeline.

**Endpoint**: `GET /acts/{year}/enhanced`

**Parameters**:
- `year` (required): Year (1989-2025)
- `limit` (optional): Number of results (default: 50)
- `offset` (optional): Pagination offset (default: 0)

**Response**:
```json
{
  "success": true,
  "data": {
    "acts": [
      {
        "id": "DU/2024/1",
        "title": "Ustawa o zmianie ustawy o podatku dochodowym",
        "status": "obowiązujący",
        "detailed_status": "in_force",
        "current_stage": "Weszła w życie",
        "stage_date": "2024-03-15T00:00:00Z",
        "days_in_stage": 45,
        "sejm_votes": [
          {
            "vote_date": "2024-02-20T15:30:00Z",
            "yes_votes": 245,
            "no_votes": 180,
            "abstain_votes": 25,
            "absent_votes": 10,
            "total_voted": 450,
            "passed": true
          }
        ],
        "senate_votes": [],
        "party_breakdowns": {
          "PiS": {"yes": 195, "no": 0, "abstain": 5},
          "KO": {"yes": 50, "no": 130, "abstain": 20}
        },
        "stages": [
          {
            "name": "Wpłynął",
            "date": "2024-01-15T00:00:00Z",
            "status": "completed"
          },
          {
            "name": "Komisja",
            "date": "2024-02-01T00:00:00Z",
            "status": "completed"
          }
        ],
        "tags": ["podatki", "ekonomia", "finanse"],
        "links": {
          "sejm": "https://sejm.gov.pl/sejm10.nsf/druk.xsp?nr=1",
          "senate": null,
          "rcl": "https://rcl.gov.pl/eli/DU/2024/1"
        }
      }
    ]
  }
}
```

### 6.4 Get Specific Act

Retrieve detailed information about a specific act.

**Endpoint**: `GET /acts/{year}/{id}`

**Parameters**:
- `year` (required): Year
- `id` (required): Act ID (e.g., "DU/2024/1")

**Response**:
```json
{
  "success": true,
  "data": {
    "act": {
      "id": "DU/2024/1",
      "title": "Ustawa o zmianie ustawy o podatku dochodowym",
      // ... full enhanced act data
    },
    "related_acts": [
      {
        "id": "DU/2023/15",
        "title": "Ustawa o podatku dochodowym",
        "relationship": "amends"
      }
    ],
    "amendments": [
      {
        "id": "DU/2024/45",
        "title": "Ustawa o zmianie ustawy o zmianie ustawy o podatku dochodowym",
        "date": "2024-05-10T00:00:00Z"
      }
    ]
  }
}
```

**Example**:
```bash
curl -X GET "https://api.ustawka.gov.pl/api/v1/acts/2024/DU%2F2024%2F1"
```

---

## 7. Search API

### 7.1 Search Acts

Perform advanced search with multiple filters.

**Endpoint**: `GET /search`

**Parameters**:
- `q` (optional): General search query
- `title` (optional): Title search
- `initiator` (optional): Initiator search
- `status` (optional): Act status (multiple allowed)
- `detailed_status` (optional): Detailed status (multiple allowed)
- `stage` (optional): Current stage (multiple allowed)
- `year_from` (optional): Start year
- `year_to` (optional): End year
- `date_from` (optional): Start date (YYYY-MM-DD)
- `date_to` (optional): End date (YYYY-MM-DD)
- `has_sejm_votes` (optional): Boolean
- `has_senate_votes` (optional): Boolean
- `voting_result` (optional): "passed", "failed", "pending"
- `committee` (optional): Committee code (multiple allowed)
- `tag` (optional): Tag (multiple allowed)
- `sort` (optional): Sort field ("title", "date", "position", "stage_date")
- `order` (optional): Sort order ("asc", "desc")
- `limit` (optional): Results per page (default: 50, max: 500)
- `offset` (optional): Pagination offset

**Response**:
```json
{
  "success": true,
  "data": {
    "acts": [
      {
        "id": "DU/2024/15",
        "title": "Ustawa o ochronie środowiska",
        "status": "w toku",
        "current_stage": "Komisja",
        "score": 0.95,
        "highlights": {
          "title": "Ustawa o ochronie <mark>środowiska</mark>"
        }
      }
    ],
    "facets": {
      "available_statuses": [
        {"value": "w toku", "count": 45, "label": "W toku"},
        {"value": "obowiązujący", "count": 1234, "label": "Obowiązujący"}
      ],
      "available_stages": [
        {"value": "Komisja", "count": 23, "label": "Komisja"},
        {"value": "II czytanie", "count": 12, "label": "Drugie czytanie"}
      ],
      "year_range": {"min": 2020, "max": 2025},
      "days_in_stage_range": {"min": 0, "max": 365}
    },
    "total_count": 1567,
    "filtered_count": 45
  },
  "meta": {
    "search_time": "0.089s",
    "page": 1,
    "limit": 50,
    "total": 45
  }
}
```

**Examples**:
```bash
# Basic text search
curl -X GET "https://api.ustawka.gov.pl/api/v1/search?q=podatek"

# Advanced search with filters
curl -X GET "https://api.ustawka.gov.pl/api/v1/search?q=środowisko&status=w%20toku&year_from=2024&sort=date&order=desc"

# Search with multiple statuses
curl -X GET "https://api.ustawka.gov.pl/api/v1/search?status=w%20toku&status=obowiązujący"
```

### 7.2 Search Suggestions

Get auto-complete suggestions for search queries.

**Endpoint**: `GET /search/suggestions`

**Parameters**:
- `q` (required): Partial query
- `field` (optional): Field to suggest for ("title", "initiator", "committee")
- `limit` (optional): Number of suggestions (default: 10)

**Response**:
```json
{
  "success": true,
  "data": {
    "suggestions": [
      "podatek dochodowy",
      "podatek od towarów i usług",
      "podatek akcyzowy"
    ],
    "query": "podat",
    "field": "title"
  }
}
```

**Example**:
```bash
curl -X GET "https://api.ustawka.gov.pl/api/v1/search/suggestions?q=podat&field=title"
```

---

## 8. Export API

### 8.1 Export Acts

Export act data in various formats.

**Endpoint**: `POST /export`

**Request Body**:
```json
{
  "format": "json",
  "filters": {
    "year": 2024,
    "status": ["w toku", "obowiązujący"],
    "has_sejm_votes": true
  },
  "fields": ["id", "title", "status", "sejm_votes"],
  "options": {
    "include_metadata": true,
    "filename": "acts_2024.json"
  }
}
```

**Parameters**:
- `format` (required): "json", "csv", "pdf"
- `filters` (optional): Search filters (same as search API)
- `fields` (optional): Fields to include
- `options` (optional): Export options

**Response** (JSON format):
```json
{
  "success": true,
  "data": {
    "export_id": "exp_2024_abc123",
    "download_url": "https://api.ustawka.gov.pl/api/v1/exports/exp_2024_abc123/download",
    "format": "json",
    "size": "2.4MB",
    "record_count": 1234,
    "expires_at": "2025-06-29T14:30:00Z"
  }
}
```

**Response** (CSV format):
```csv
id,title,status,year,position
DU/2024/1,"Ustawa o podatku dochodowym",obowiązujący,2024,1
DU/2024/2,"Ustawa o ochronie środowiska",w toku,2024,2
```

**Examples**:
```bash
# Export as JSON
curl -X POST "https://api.ustawka.gov.pl/api/v1/export" \
  -H "Content-Type: application/json" \
  -d '{"format": "json", "filters": {"year": 2024}}'

# Export as CSV with specific fields
curl -X POST "https://api.ustawka.gov.pl/api/v1/export" \
  -H "Content-Type: application/json" \
  -d '{
    "format": "csv",
    "filters": {"year": 2024},
    "fields": ["id", "title", "status"]
  }'
```

### 8.2 Download Export

Download a previously generated export.

**Endpoint**: `GET /exports/{export_id}/download`

**Response**: Binary file download with appropriate content type headers.

**Example**:
```bash
curl -X GET "https://api.ustawka.gov.pl/api/v1/exports/exp_2024_abc123/download" \
  -o acts_export.json
```

### 8.3 Export Status

Check the status of an export job.

**Endpoint**: `GET /exports/{export_id}`

**Response**:
```json
{
  "success": true,
  "data": {
    "export_id": "exp_2024_abc123",
    "status": "completed",
    "progress": 100,
    "created_at": "2025-06-28T14:30:00Z",
    "completed_at": "2025-06-28T14:32:15Z",
    "download_url": "https://api.ustawka.gov.pl/api/v1/exports/exp_2024_abc123/download",
    "expires_at": "2025-06-29T14:30:00Z"
  }
}
```

---

## 9. Comparison API

### 9.1 Compare Acts

Compare multiple acts side-by-side.

**Endpoint**: `POST /compare`

**Request Body**:
```json
{
  "act_ids": ["DU/2024/1", "DU/2024/15", "DU/2023/45"],
  "comparison_type": "detailed",
  "include_voting": true,
  "include_timeline": true
}
```

**Response**:
```json
{
  "success": true,
  "data": {
    "comparison": {
      "acts": [
        {
          "id": "DU/2024/1",
          "title": "Ustawa o podatku dochodowym",
          "similarities": ["finansowe", "podatkowe"],
          "differences": ["zakres_stosowania", "stawki"]
        }
      ],
      "summary": {
        "total_similarities": 15,
        "total_differences": 8,
        "similarity_score": 0.85
      },
      "voting_comparison": {
        "similar_patterns": true,
        "party_alignment": 0.72
      },
      "timeline_comparison": {
        "average_duration": "45 days",
        "stages_comparison": [
          {
            "stage": "Komisja",
            "durations": [15, 23, 18]
          }
        ]
      }
    }
  }
}
```

### 9.2 Get Comparison Suggestions

Get suggestions for acts to compare with a given act.

**Endpoint**: `GET /compare/suggestions/{act_id}`

**Parameters**:
- `act_id` (required): Base act ID
- `limit` (optional): Number of suggestions (default: 10)
- `similarity_threshold` (optional): Minimum similarity (0.0-1.0)

**Response**:
```json
{
  "success": true,
  "data": {
    "suggestions": [
      {
        "id": "DU/2024/15",
        "title": "Ustawa o zmianie ustawy o podatku dochodowym",
        "similarity_score": 0.92,
        "reason": "Similar topic and legislative approach"
      }
    ],
    "base_act": {
      "id": "DU/2024/1",
      "title": "Ustawa o podatku dochodowym"
    }
  }
}
```

---

## 10. System API

### 10.1 Health Check

Check system health and status.

**Endpoint**: `GET /health`

**Response**:
```json
{
  "success": true,
  "data": {
    "status": "healthy",
    "timestamp": "2025-06-28T14:30:00Z",
    "version": "1.0.0",
    "uptime": "72h15m30s",
    "checks": {
      "database": {
        "status": "healthy",
        "duration": "0.012s"
      },
      "sejm_api": {
        "status": "healthy", 
        "duration": "0.234s"
      },
      "cache": {
        "status": "healthy",
        "duration": "0.003s"
      }
    }
  }
}
```

### 10.2 System Metrics

Get system performance metrics.

**Endpoint**: `GET /metrics`

**Response**:
```json
{
  "success": true,
  "data": {
    "requests": {
      "total": 1234567,
      "per_minute": 45,
      "error_rate": 0.02
    },
    "database": {
      "connections_active": 5,
      "query_duration_avg": "0.015s",
      "cache_hit_rate": 0.94
    },
    "external_apis": {
      "sejm_api_calls": 1234,
      "senate_api_calls": 567,
      "success_rate": 0.998
    },
    "background_services": {
      "last_sync": "2025-06-28T14:00:00Z",
      "acts_processed_today": 1234,
      "monitoring_active": true
    }
  }
}
```

### 10.3 Background Service Status

Get status of background processing services.

**Endpoint**: `GET /status`

**Response**:
```json
{
  "success": true,
  "data": {
    "is_running": true,
    "last_full_sync": "2025-06-28T06:00:00Z",
    "last_enrichment_run": "2025-06-28T14:00:00Z",
    "last_health_check": "2025-06-28T14:29:45Z",
    "error_count": 0,
    "processed_today": 1234,
    "active_jobs": 2,
    "queued_jobs": 5,
    "health_status": "healthy",
    "next_scheduled_sync": "2025-06-28T15:00:00Z",
    "estimated_processing_time": "~5 minutes"
  }
}
```

---

## 11. WebSocket API

### 11.1 Real-time Updates

Connect to WebSocket for real-time act updates.

**Endpoint**: `wss://api.ustawka.gov.pl/ws/v1/updates`

**Connection**:
```javascript
const ws = new WebSocket('wss://api.ustawka.gov.pl/ws/v1/updates');

ws.onopen = function(event) {
  // Subscribe to specific updates
  ws.send(JSON.stringify({
    type: 'subscribe',
    filters: {
      years: [2024, 2025],
      statuses: ['w toku'],
      act_ids: ['DU/2024/1', 'DU/2024/15']
    }
  }));
};

ws.onmessage = function(event) {
  const update = JSON.parse(event.data);
  console.log('Act update:', update);
};
```

**Message Types**:

**Subscription Message**:
```json
{
  "type": "subscribe",
  "filters": {
    "years": [2024, 2025],
    "statuses": ["w toku"],
    "keywords": ["podatek"],
    "act_ids": ["DU/2024/1"]
  }
}
```

**Status Update Message**:
```json
{
  "type": "status_update",
  "data": {
    "act_id": "DU/2024/1",
    "previous_status": "Komisja",
    "new_status": "II czytanie",
    "change_time": "2025-06-28T14:30:00Z",
    "metadata": {
      "committee_decision": "positive",
      "next_session_date": "2025-07-01T10:00:00Z"
    }
  }
}
```

**Voting Update Message**:
```json
{
  "type": "voting_update",
  "data": {
    "act_id": "DU/2024/1",
    "chamber": "sejm",
    "vote_result": {
      "yes_votes": 245,
      "no_votes": 180,
      "abstain_votes": 25,
      "passed": true
    },
    "vote_time": "2025-06-28T14:30:00Z"
  }
}
```

---

## 12. SDKs and Examples

### 12.1 JavaScript/TypeScript SDK

**Installation**:
```bash
npm install @ustawka/api-client
```

**Usage**:
```typescript
import { UstawkaClient } from '@ustawka/api-client';

const client = new UstawkaClient({
  baseURL: 'https://api.ustawka.gov.pl/api/v1',
  apiKey: 'your-api-key' // optional
});

// Get acts for 2024
const acts = await client.acts.getByYear(2024, {
  enhanced: true,
  limit: 100
});

// Search for acts
const searchResults = await client.search.acts({
  query: 'podatek',
  status: ['w toku'],
  yearFrom: 2024
});

// Compare acts
const comparison = await client.compare.acts([
  'DU/2024/1',
  'DU/2024/15'
]);

// Real-time updates
const ws = client.websocket.connect();
ws.subscribe({
  years: [2024],
  statuses: ['w toku']
});

ws.on('status_update', (update) => {
  console.log('Act status changed:', update);
});
```

### 12.2 Python SDK

**Installation**:
```bash
pip install ustawka-api
```

**Usage**:
```python
from ustawka import UstawkaClient

client = UstawkaClient(
    base_url='https://api.ustawka.gov.pl/api/v1',
    api_key='your-api-key'  # optional
)

# Get acts for 2024
acts = client.acts.get_by_year(2024, enhanced=True)

# Search for acts
results = client.search.acts(
    query='podatek',
    status=['w toku'],
    year_from=2024
)

# Export data
export_job = client.export.create(
    format='csv',
    filters={'year': 2024},
    fields=['id', 'title', 'status']
)

# Download when ready
if export_job.status == 'completed':
    data = client.export.download(export_job.id)
```

### 12.3 Go SDK

**Installation**:
```bash
go get github.com/ustawka/go-client
```

**Usage**:
```go
package main

import (
    "context"
    "fmt"
    "github.com/ustawka/go-client"
)

func main() {
    client := ustawka.NewClient(&ustawka.Config{
        BaseURL: "https://api.ustawka.gov.pl/api/v1",
        APIKey:  "your-api-key", // optional
    })
    
    // Get acts for 2024
    acts, err := client.Acts.GetByYear(context.Background(), 2024, &ustawka.ActsOptions{
        Enhanced: true,
        Limit:    100,
    })
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Found %d acts\n", len(acts))
    
    // Search for acts
    results, err := client.Search.Acts(context.Background(), &ustawka.SearchCriteria{
        Query:    "podatek",
        Statuses: []string{"w toku"},
        YearFrom: ustawka.Int(2024),
    })
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Found %d matching acts\n", results.TotalCount)
}
```

### 12.4 cURL Examples

**Get all acts for 2024**:
```bash
curl -X GET "https://api.ustawka.gov.pl/api/v1/acts/2024" \
  -H "Accept: application/json"
```

**Search with multiple filters**:
```bash
curl -X GET "https://api.ustawka.gov.pl/api/v1/search" \
  -G \
  -d "q=podatek" \
  -d "status=w toku" \
  -d "year_from=2024" \
  -d "limit=50" \
  -H "Accept: application/json"
```

**Export acts as CSV**:
```bash
curl -X POST "https://api.ustawka.gov.pl/api/v1/export" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  -d '{
    "format": "csv",
    "filters": {
      "year": 2024,
      "status": ["obowiązujący"]
    },
    "fields": ["id", "title", "status", "year"]
  }'
```

**Compare multiple acts**:
```bash
curl -X POST "https://api.ustawka.gov.pl/api/v1/compare" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  -d '{
    "act_ids": ["DU/2024/1", "DU/2024/15"],
    "comparison_type": "detailed",
    "include_voting": true
  }'
```

---

## Appendix

### A. Act Status Values

| Status | Description |
|--------|-------------|
| `wpłynął` | Submitted to parliament |
| `w toku` | Currently being processed |
| `obowiązujący` | In force |
| `uchylony` | Repealed |
| `odrzucony` | Rejected |

### B. Detailed Status Values

| Status | Description |
|--------|-------------|
| `submitted` | Initial submission |
| `committee_work` | In committee review |
| `second_reading` | Second reading in progress |
| `third_reading` | Third reading in progress |
| `passed_sejm` | Passed by Sejm |
| `senate_review` | Under Senate review |
| `senate_accepted` | Accepted by Senate |
| `presidential_review` | Under presidential review |
| `in_force` | Entered into force |
| `rejected` | Rejected |

### C. Field Descriptions

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique act identifier |
| `title` | string | Act title |
| `status` | string | Current status |
| `detailed_status` | string | Detailed processing status |
| `current_stage` | string | Current stage in Polish |
| `stage_date` | datetime | Date of current stage |
| `days_in_stage` | integer | Days in current stage |
| `sejm_votes` | array | Sejm voting records |
| `senate_votes` | array | Senate voting records |
| `party_breakdowns` | object | Voting by political party |
| `stages` | array | Processing timeline |
| `tags` | array | Classification tags |
| `links` | object | Related URLs |

---

**Document Version**: 1.0  
**Last Updated**: June 28, 2025  
**Contact**: api-support@ustawka.gov.pl