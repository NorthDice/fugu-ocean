# Fugu Ocean — MVP Plan

A learning project — a Google Drive-like app built on Go and S3-compatible object storage.
The main goal isn't a polished product but practice: object storage (S3/MinIO), Go backend
architecture, authentication, and making deliberate architectural decisions that don't get
in the way of extending the project later.

## Goals

- Work with the S3 API: presigned URLs, multipart upload.
- Layered Go service architecture (handler → service → repository).
- Authentication/authorization (JWT), an access-control model.
- Async processing via a queue (background worker, producer/consumer pattern).
- Make architectural decisions so the MVP stays easy to extend without a rewrite.

## Tech Stack

| Component            | Choice                     | Why |
|-----------------------|-----------------------------|-----|
| Backend                | Go, layered architecture (handler/service/repo) | Simple and clear, easy to extend layer by layer |
| API                    | REST + JSON                 | Easiest to wire up with a hand-rolled UI |
| Metadata (DB)          | PostgreSQL                   | Relational model fits folder trees, permissions, queue jobs |
| Object storage          | MinIO (S3-compatible API)   | Free, local, same protocol as real AWS S3 |
| Full-text search        | Meilisearch                  | Lightweight, simple REST API, spins up fast in docker-compose |
| Task queue              | `index_jobs` table in Postgres (`SELECT … FOR UPDATE SKIP LOCKED`) | No extra infra needed, but still practices the queue pattern |
| Authentication           | JWT (access + refresh), passwords hashed with bcrypt | Standard approach, enough for a pet project |
| Local deployment         | docker-compose (Postgres + MinIO + Meilisearch + backend) | One `docker-compose up` brings up the whole stack |

## MVP Scope

### Auth
- Registration (email + password, bcrypt hash).
- Login → access JWT (short TTL) + refresh token (long TTL, stored in DB so it can be revoked).
- Middleware that validates the JWT on protected routes.

### Files and folders
- Folder tree (self-referencing `parent_id`), every folder and file has an owner.
- Create folder, rename, move, delete (soft-delete → trash).
- Upload a file via presigned URL (see "Upload flow" below), including multipart upload for large files.
- Download a file via presigned GET URL.
- Rename/move/delete a file (soft-delete → trash).

### Trash (soft-delete)
- Deleted files/folders are marked with `deleted_at`, not removed immediately.
- View trash, restore, permanently delete (only then do we actually remove the object from MinIO).

### File sharing
- The owner can share a specific file with another user (by email) with `read` or `write` permission.
- A "shared with me" list.
- **Design decision:** in the MVP only the file itself is shared, not the whole folder recursively —
  this keeps the permission model much simpler and is enough for practice purposes. Sharing entire
  folders is pushed to the v2 backlog (see below), for when it's worth working out permission
  inheritance across the tree.

### Full-text search
- Search over file contents via Meilisearch.
- In the MVP, text is extracted only from plain text files (`.txt`, `.md`) — no PDF/DOCX parsers.
- Indexing is asynchronous: once an upload is confirmed, a job is created in the DB; a background
  worker reads the file from MinIO, extracts the text, and pushes the document into Meilisearch.
- Search results respect access control: only files the user owns or has been shared are returned.

## Non-goals (deliberately deferred)

- File versioning (change history).
- Per-user storage quotas.
- Previews/thumbnails (images, documents).
- Text extraction from PDF/DOCX for search.
- Recursive sharing of entire folders.
- A real message broker (Redis/RabbitMQ/NATS) — using a table-based queue instead.
- Real-time updates (websockets, sync across tabs/devices).
- Rate limiting, antivirus scanning of uploaded files.
- Admin panel.

## Architecture

```
Client (curl/Postman/hand-rolled UI)
        │
        ▼
   HTTP Handler layer   (request parsing, validation, HTTP status codes)
        │
        ▼
   Service layer         (business logic: access control, orchestration)
        │
        ▼
   Repository layer      (interfaces: UserRepo, FileRepo, FolderRepo, ShareRepo, StorageRepo, SearchRepo)
        │
   ┌────┼────────────┬─────────────┐
   ▼    ▼             ▼             ▼
Postgres  MinIO   Meilisearch   (future: other adapters)
```

Repositories sit behind interfaces — the service layer doesn't know storage is specifically MinIO
or that search is specifically Meilisearch. This makes it painless to later swap MinIO for real AWS S3
or Meilisearch for Elasticsearch without touching business logic.

## Data model (core entities)

- **User**: id, email, password_hash, created_at
- **RefreshToken**: id, user_id, token_hash, expires_at, revoked_at
- **Folder**: id, name, parent_id (nullable, self-FK), owner_id, created_at, deleted_at
- **File**: id, name, folder_id, owner_id, size, content_type, storage_key, status (`pending`/`ready`), created_at, deleted_at
- **Share**: id, file_id, owner_id, shared_with_user_id, permission (`read`/`write`), created_at
- **IndexJob**: id, file_id, status (`pending`/`processing`/`done`/`failed`), created_at, processed_at

## Upload flow (presigned URL)

1. Client → `POST /files` with metadata (name, folder_id, size, content-type). The backend creates a
   record with status `pending` and generates a presigned PUT URL (or initiates a multipart upload
   for large files).
2. Client uploads the file **directly to MinIO** using that URL — file traffic never passes through
   the backend.
3. Client → `PATCH /files/:id/complete`. The backend verifies the object in MinIO (a HEAD request,
   size check), flips the status to `ready`, and creates a job in `index_jobs`.
4. A background worker polls `index_jobs` (`SELECT … FOR UPDATE SKIP LOCKED`); for `.txt`/`.md` files
   it reads the content from MinIO and indexes it in Meilisearch.

For large files — multipart upload: the client splits the file into parts, each part is uploaded via
its own presigned URL, then `POST /files/:id/multipart/complete` stitches the parts together on MinIO's side.

## Download flow

`GET /files/:id/download` → the backend checks permissions (owner or a matching Share entry),
generates a presigned GET URL with a short TTL → the client downloads the file directly from MinIO.

## API (draft endpoint list)

```
POST   /auth/register
POST   /auth/login
POST   /auth/refresh

POST   /folders
GET    /folders/:id              (folder contents)
PATCH  /folders/:id              (rename/move)
DELETE /folders/:id              (soft-delete)

POST   /files                    (init upload, presigned PUT URL)
POST   /files/:id/multipart/complete
PATCH  /files/:id/complete       (confirm single-part upload)
GET    /files/:id/download       (presigned GET URL)
PATCH  /files/:id                (rename/move)
DELETE /files/:id                (soft-delete)

GET    /trash
POST   /trash/:id/restore
DELETE /trash/:id                (permanent delete)

POST   /files/:id/share          {email, permission}
DELETE /files/:id/share/:user_id
GET    /shared-with-me

GET    /search?q=...
```

## Local deployment

`docker-compose.yml` brings up:
- `postgres` — metadata
- `minio` (+ an `mc` init container to create the bucket)
- `meilisearch` — search index
- `backend` — the Go service itself (HTTP API + background indexing worker, can run as a separate process/binary)

## v2 backlog (post-MVP)

- File versioning.
- Storage quotas.
- Previews/thumbnails.
- Text extraction from PDF/DOCX.
- Recursive folder sharing with permission inheritance.
- Replace the table-based queue with a real broker (if worker scaling becomes necessary).
- Real-time notifications (websockets).
