## Product Requirements Document (PRD)

# Backend Platform v1 for Mystery Media Brand

**Scope:** Backend-first foundation to power website, admin operations, and future team workflows.
**Primary Goal:** Build a secure, scalable backend with role-based access for content management.

---

# 1. Product Summary

We are building the backend for a media/content platform that supports:

* Public website content consumption (no login required)
* Internal admin operations for you and future team
* Video content management
* Sources / references management
* Corrections management
* Suggestions intake
* User management with strict role permissions

Frontend can be built later on top of this API system.

---

# 2. User Roles

## Public Viewer (Unauthenticated)

Can:

* View public website pages
* View published videos/resources
* Submit suggestions
* Submit corrections
* Contact team

Cannot:

* Access admin APIs
* Modify content

---

## SUPER_USER (You)

Full system control.

Can:

* Create users
* Update users
* Disable users
* Delete content
* Create / edit / publish content
* Manage all corrections
* Manage all suggestions
* View analytics
* Manage settings

---

## MANAGER (Team Members)

Operational content role.

Can:

* Login to admin
* Create content
* View content
* Edit content
* Publish/unpublish content (configurable)
* Manage sources
* Review suggestions
* Review corrections

Cannot:

* Create users
* Delete users
* Delete content permanently
* Access critical settings
* Change SUPER_USER roles

---

# 3. Core Permission Matrix

| Action              | Viewer | MANAGER | SUPER_USER |
| ------------------- | ------ | ------- | ---------- |
| View public content | Yes    | Yes     | Yes        |
| Login admin         | No     | Yes     | Yes        |
| Create users        | No     | No      | Yes        |
| Edit users          | No     | No      | Yes        |
| Delete users        | No     | No      | Yes        |
| Create videos       | No     | Yes     | Yes        |
| Edit videos         | No     | Yes     | Yes        |
| Delete videos       | No     | No      | Yes        |
| Publish videos      | No     | Yes*    | Yes        |
| Manage sources      | No     | Yes     | Yes        |
| View suggestions    | No     | Yes     | Yes        |
| Resolve corrections | No     | Yes     | Yes        |
| System settings     | No     | No      | Yes        |

* configurable

---

# 4. Functional Modules

---

## Module A: Authentication & Authorization

### Requirements

* Email + password login
* JWT access token
* Refresh token support
* Password hashing
* Role-based middleware
* Session invalidation on password reset/logout
* Audit logs for admin actions

### APIs

```text id="prd001"
/auth/login
/auth/refresh
/auth/logout
/auth/me
```

---

## Module B: User Management

### Requirements

Only SUPER_USER can create users.

### Fields

* name
* email
* password
* role (SUPER_USER / MANAGER)
* status (active / disabled)

### APIs

```text id="prd002"
POST   /admin/users
GET    /admin/users
PUT    /admin/users/:id
PATCH  /admin/users/:id/status
DELETE /admin/users/:id
```

---

## Module C: Video Content Management

### Requirements

Manage website content pages tied to videos.

### Fields

* title
* slug
* youtubeUrl
* thumbnailUrl
* shortDescription
* fullDescription
* category
* tags[]
* status (draft/published)
* publishedAt
* createdBy
* updatedBy

### APIs

```text id="prd003"
POST   /admin/videos
GET    /admin/videos
GET    /admin/videos/:id
PUT    /admin/videos/:id
DELETE /admin/videos/:id   (SUPER_USER only)
PATCH  /admin/videos/:id/publish
```

Public:

```text id="prd004"
GET /videos
GET /videos/:slug
```

---

## Module D: Sources / References

### Requirements

Each video can have many sources.

### Fields

* videoId
* title
* type (article/book/archive/video)
* url
* note
* credibilityScore

### APIs

```text id="prd005"
POST   /admin/videos/:id/sources
GET    /admin/videos/:id/sources
PUT    /admin/sources/:id
DELETE /admin/sources/:id (SUPER_USER only optional)
```

Public:

```text id="prd006"
GET /videos/:slug/sources
```

---

## Module E: Corrections System

### Public Submission

Users can report factual issues.

### Fields

* videoId
* message
* sourceUrl
* email optional
* status (new/reviewing/applied/rejected)

### APIs

```text id="prd007"
POST /corrections
GET  /admin/corrections
PUT  /admin/corrections/:id
```

---

## Module F: Suggestions Intake

### Public can recommend future topics.

### Fields

* title
* description
* links[]
* email optional
* status

### APIs

```text id="prd008"
POST /suggestions
GET  /admin/suggestions
PUT  /admin/suggestions/:id
```

---

## Module G: Contact / Leads

```text id="prd009"
POST /contact
GET  /admin/contacts
```

---

# 5. Non-Functional Requirements

## Security

* Passwords hashed (bcrypt/argon2)
* JWT signed securely
* Rate limiting public forms
* Input validation
* CORS controls
* IP logging for abuse monitoring
* Role middleware mandatory

## Reliability

* Structured logs
* Health check endpoint
* Graceful shutdown
* Config via env

## Performance

* Pagination on admin lists
* Indexes on slug/status/date
* Cache public content later

---

# 6. Data Store

## Database Choice

MongoDB (initial phase)

---

## Collections

```text id="prd010"
users
videos
sources
corrections
suggestions
contacts
audit_logs
refresh_tokens
```

---

# 7. API Response Standards

## Success

```json
{
  "success": true,
  "data": {}
}
```

## Error

```json
{
  "success": false,
  "message": "Unauthorized"
}
```

---

# 8. Audit Logging

Track:

* login attempts
* user creation
* deletes
* publish actions
* role changes
* correction resolutions

Only SUPER_USER can view full logs.

---

# 9. Deployment v1

## Backend

Go API service

## Database

MongoDB Atlas

## Future

Redis, queues, analytics workers

---

# 10. MVP Build Order (Recommended)

## Week 1

* auth
* role middleware
* user management

## Week 2

* videos CRUD
* public video APIs

## Week 3

* sources
* suggestions
* corrections

## Week 4

* audit logs
* hardening
* docs
* deploy

---

# 11. Explicit Business Rules

1. Only SUPER_USER creates users
2. MANAGER can create/view/edit content
3. Delete rights only SUPER_USER
4. Public viewers never authenticate
5. Unpublished content never visible publicly

---

# 12. Success Criteria

* SUPER_USER can onboard managers
* Managers can run daily content ops
* Public can browse content + submit feedback
* Backend secure enough for production MVP
* Easy expansion for merch/store later

---

# 13. My Strong Recommendation

Use clean architecture:

```text id="prd011"
handlers/
services/
repositories/
middleware/
models/
routes/
utils/
```

This matches your backend strengths.

---
