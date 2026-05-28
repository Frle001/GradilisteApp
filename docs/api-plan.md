# API Plan

## Base URL

```
http://localhost:8080/api
```

## Response Format

All responses return JSON with consistent structure.

### Success Response

```json
{
  "success": true,
  "data": {},
  "message": "Operation successful"
}
```

### Error Response

```json
{
  "success": false,
  "error": "error_code",
  "message": "Human-readable error message"
}
```

## Authentication (Phase 2)

```
Authorization: Bearer <jwt_token>
```

## Endpoints Overview

### Phase 1 (Foundation)

- [x] Health check

### Phase 2 (Auth)

- [ ] POST `/auth/login` — Login and get token
- [ ] POST `/auth/register` — Register new user
- [ ] POST `/auth/refresh` — Refresh expired token
- [ ] POST `/auth/logout` — Logout

### Phase 3 (Employees)

- [ ] GET `/employees` — List employees (paginated)
- [ ] GET `/employees/{id}` — Get employee details
- [ ] POST `/employees` — Create employee
- [ ] PUT `/employees/{id}` — Update employee
- [ ] DELETE `/employees/{id}` — Deactivate employee

### Phase 3 (Projects)

- [ ] GET `/projects` — List projects (paginated)
- [ ] GET `/projects/{id}` — Get project details
- [ ] POST `/projects` — Create project
- [ ] PUT `/projects/{id}` — Update project
- [ ] DELETE `/projects/{id}` — Archive project
- [ ] GET `/projects/{id}/assignments` — Get team assignments

### Phase 3 (Assignments)

- [ ] POST `/projects/{id}/assignments` — Assign employee to project
- [ ] DELETE `/projects/{id}/assignments/{employeeId}` — Remove from project
- [ ] GET `/employees/{id}/assignments` — Get projects assigned to employee

### Phase 3 (Daily Reports)

- [ ] GET `/reports` — List reports (paginated, filtered by project/employee)
- [ ] GET `/reports/{id}` — Get report details
- [ ] POST `/reports` — Create daily report
- [ ] PUT `/reports/{id}` — Update report
- [ ] DELETE `/reports/{id}` — Delete report

### Phase 4 (Materials)

- [ ] GET `/materials` — List material activities
- [ ] POST `/materials` — Record material activity
- [ ] GET `/materials/inventory` — Get inventory summary

### Phase 4 (Documents & Files)

- [ ] POST `/uploads` — Upload file (image, PDF, Excel)
- [ ] GET `/uploads/{id}` — Download/view file

## Detailed Endpoints

### Health Check

**Endpoint:** `GET /api/health`

**Authentication:** None

**Response:**
```json
{
  "status": "ok",
  "message": "Gradilište API is running"
}
```

**HTTP Status:** 200

---

### Authentication (Phase 2)

#### Login

**Endpoint:** `POST /api/auth/login`

**Request:**
```json
{
  "email": "direktor@example.com",
  "password": "password123"
}
```

**Response (Success):**
```json
{
  "success": true,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "user": {
      "id": 1,
      "email": "direktor@example.com",
      "firstName": "Marko",
      "lastName": "Direktor",
      "role": "direktor"
    }
  }
}
```

**Response (Error):**
```json
{
  "success": false,
  "error": "invalid_credentials",
  "message": "Invalid email or password"
}
```

**HTTP Status:** 200 (success), 401 (invalid)

---

### Employees (Phase 3)

#### List Employees

**Endpoint:** `GET /api/employees?page=1&limit=20&active=true`

**Authentication:** Required (all roles can read)

**Query Parameters:**
- `page` — Page number (default: 1)
- `limit` — Results per page (default: 20, max: 100)
- `active` — Filter by active status (true/false/all)
- `role` — Filter by role (optional)

**Response:**
```json
{
  "success": true,
  "data": {
    "employees": [
      {
        "id": 1,
        "firstName": "Marko",
        "lastName": "Marković",
        "role": "direktor",
        "active": true,
        "createdAt": "2026-01-01T10:00:00Z",
        "updatedAt": "2026-01-01T10:00:00Z"
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 20,
      "total": 100,
      "pages": 5
    }
  }
}
```

---

#### Create Employee

**Endpoint:** `POST /api/employees`

**Authentication:** Required (Direktor, Inženjer, Administracija only)

**Request:**
```json
{
  "firstName": "Ivan",
  "lastName": "Ivanović",
  "role": "radnik"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": 8,
    "firstName": "Ivan",
    "lastName": "Ivanović",
    "role": "radnik",
    "active": true,
    "createdAt": "2026-05-28T10:00:00Z",
    "updatedAt": "2026-05-28T10:00:00Z"
  }
}
```

---

### Projects (Phase 3)

#### List Projects

**Endpoint:** `GET /api/projects?page=1&limit=20&active=true`

**Authentication:** Required

**Authorization:**
- Direktor/Inženjer: See all projects
- Poslovođa: See only assigned projects
- Administracija: No access

**Query Parameters:**
- `page` — Page number
- `limit` — Results per page
- `active` — Filter by active status

**Response:**
```json
{
  "success": true,
  "data": {
    "projects": [
      {
        "id": 1,
        "name": "Nove poslovne prostorije",
        "description": "Izgradnja novog poslovnog kompleksa",
        "active": true,
        "createdAt": "2026-01-15T08:00:00Z",
        "updatedAt": "2026-01-15T08:00:00Z"
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 20,
      "total": 3,
      "pages": 1
    }
  }
}
```

---

#### Create Project

**Endpoint:** `POST /api/projects`

**Authentication:** Required (Direktor, Inženjer only)

**Request:**
```json
{
  "name": "Novi projekat",
  "description": "Opis projekta"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": 4,
    "name": "Novi projekat",
    "description": "Opis projekta",
    "active": true,
    "createdAt": "2026-05-28T10:00:00Z",
    "updatedAt": "2026-05-28T10:00:00Z"
  }
}
```

---

### Daily Reports (Phase 3)

#### List Reports

**Endpoint:** `GET /api/reports?projectId=1&employeeId=4&dateFrom=2026-01-01&dateTo=2026-12-31`

**Authentication:** Required

**Query Parameters:**
- `page` — Page number
- `limit` — Results per page
- `projectId` — Filter by project (optional)
- `employeeId` — Filter by employee (optional)
- `dateFrom` — Start date (YYYY-MM-DD)
- `dateTo` — End date (YYYY-MM-DD)

**Response:**
```json
{
  "success": true,
  "data": {
    "reports": [
      {
        "id": 1,
        "projectId": 1,
        "employeeId": 4,
        "date": "2026-05-28",
        "description": "Radovi na postavljanju armature",
        "workers": [
          {
            "employeeId": 5,
            "hours": 8
          }
        ],
        "createdAt": "2026-05-28T10:00:00Z",
        "updatedAt": "2026-05-28T10:00:00Z"
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 20,
      "total": 50,
      "pages": 3
    }
  }
}
```

---

## Pagination

All list endpoints support pagination:

- `page` — Current page (1-indexed)
- `limit` — Results per page (default: 20, max: 100)

Response includes:
```json
{
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 150,
    "pages": 8
  }
}
```

---

## Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `invalid_request` | 400 | Malformed request or invalid input |
| `unauthorized` | 401 | Missing or invalid authentication token |
| `forbidden` | 403 | User doesn't have permission for this resource |
| `not_found` | 404 | Resource not found |
| `conflict` | 409 | Resource conflict (e.g., duplicate email) |
| `server_error` | 500 | Server error |

---

## Rate Limiting (Phase 2)

Future implementation:
- 100 requests per minute per IP
- 1000 requests per hour per user (authenticated)
- Returns `X-RateLimit-*` headers

---

## Versioning

Currently on **v1** (implicit in `/api` path).

Future versions would use `/api/v2/`, etc.

---

## Documentation Tools

Future additions:
- **Swagger/OpenAPI** — Auto-generated API docs
- **Postman Collection** — For testing
- **Mock Server** — For frontend development

---

## Phase-by-Phase Rollout

### Phase 1 ✓
- [x] Health check endpoint

### Phase 2 (Next)
- [ ] Authentication (login, register, token refresh)
- [ ] Authorization middleware
- [ ] Error handling & validation

### Phase 3
- [ ] Employee CRUD
- [ ] Project CRUD
- [ ] Assignment management
- [ ] Daily reports

### Phase 4
- [ ] Material tracking
- [ ] File uploads
- [ ] Inventory management
- [ ] Excel import/export

---

## Testing Endpoints

### cURL

```bash
# Health check
curl http://localhost:8080/api/health

# Future: Login
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "direktor@example.com", "password": "password"}'
```

### Postman

Import the collection once available (Phase 3).

### Frontend (JavaScript/TypeScript)

```typescript
import apiClient from '@/lib/api-client'

// Health check
const health = await apiClient.get('/health')

// Future: Login
const login = await apiClient.post('/auth/login', {
  email: 'direktor@example.com',
  password: 'password'
})
```
