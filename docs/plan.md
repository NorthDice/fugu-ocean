# Fugu Ocean — Implementation Plan

Implementation is broken down by module, not by file. Each module is a checkpoint —
check it off only once every sub-item below it is done and working end-to-end.
Sub-items are the breakdown of what that module covers; they are not individual
checkpoints themselves. Follow the modules top to bottom — each one builds on the
previous.

Reference: [docs/mvp.md](./mvp.md) for scope, architecture, and data model.

---

- [x] **Module 0 — Project Bootstrap & Infrastructure**
  - Repo layout: `cmd/`, `internal/{handler,service,repository}`, `migrations/`, `config/`
  - Config loading from env vars (DB DSN, MinIO creds/endpoint, Meilisearch endpoint, JWT secret)
  - `docker-compose.yml`: postgres, minio (+ `mc` init container to create the bucket), meilisearch
  - Migration tooling set up (golang-migrate or goose) and wired to run on startup or via a make target
  - HTTP router (chi/echo) with base middleware: request logging, panic recovery, JSON error responses
  - `GET /health` endpoint (checks DB + MinIO + Meilisearch connectivity)

- [ ] **Module 1 — Auth**
  - `users` table + migration
  - Password hashing (bcrypt) on registration
  - `POST /auth/register`
  - `POST /auth/login` — issues access JWT (short TTL) + refresh token
  - `refresh_tokens` table (hashed token, expiry, revoked_at) for revocation support
  - `POST /auth/refresh` — rotates refresh token, issues new access JWT
  - JWT auth middleware for protected routes (extracts `user_id` into request context)
  - Consistent 401/403 error responses

- [ ] **Module 2 — Storage Integration Foundation (MinIO/S3)**
  - `StorageRepo` interface (abstracts MinIO behind a port, so it can be swapped for real S3 later)
  - MinIO client wrapper implementing `StorageRepo`
  - Bucket existence check / auto-create on startup
  - Presigned PUT URL generation (with TTL)
  - Presigned GET URL generation (with TTL)
  - Multipart upload support: init, per-part presigned URLs, complete, abort
  - `HEAD` object check (used later to verify an upload actually landed)

- [ ] **Module 3 — Folders**
  - `folders` table (self-referencing `parent_id`, `owner_id`, `deleted_at`) + migration
  - `POST /folders` — create folder
  - `GET /folders/:id` — list children (files + subfolders), breadcrumb path
  - `PATCH /folders/:id` — rename / move (reparent)
  - `DELETE /folders/:id` — soft-delete (sets `deleted_at`, no physical cleanup yet)
  - Ownership check in the service layer (only the owner can mutate a folder)
  - Guard against cycles when moving a folder (can't move a folder into its own descendant)

- [ ] **Module 4 — Files: Upload & Metadata**
  - `files` table (`status`: `pending`/`ready`, `storage_key`, `owner_id`, `folder_id`, `deleted_at`) + migration
  - `POST /files` — creates a `pending` record, returns presigned PUT URL (or multipart init for large files)
  - `PATCH /files/:id/complete` — `HEAD`-verifies the object in MinIO, flips status to `ready`
  - `POST /files/:id/multipart/complete` — completes a multipart upload
  - `PATCH /files/:id` — rename / move to another folder
  - `DELETE /files/:id` — soft-delete
  - Ownership check in the service layer

- [ ] **Module 5 — Files: Download**
  - `GET /files/:id/download` — access check (owner, or see Module 7 for shared access), returns presigned GET URL
  - Short TTL on the download URL

- [ ] **Module 6 — Trash**
  - `GET /trash` — lists soft-deleted files and folders owned by the user
  - `POST /trash/:id/restore` — clears `deleted_at`
  - `DELETE /trash/:id` — permanent delete: removes the DB row and the underlying MinIO object
  - Decide and implement whether restoring a folder cascades to restore its children, and whether
    permanently deleting a folder cascades to its contents

- [ ] **Module 7 — Sharing**
  - `shares` table (`file_id`, `owner_id`, `shared_with_user_id`, `permission`: `read`/`write`) + migration
  - `POST /files/:id/share` — owner shares a file with another user by email
  - `DELETE /files/:id/share/:user_id` — revoke access
  - `GET /shared-with-me` — list files shared with the current user
  - Update the file access-control check (Modules 4/5) to also allow users with a matching `share` entry

- [ ] **Module 8 — Full-Text Search**
  - `index_jobs` table (`file_id`, `status`) + migration
  - Meilisearch client wrapper + index setup/config
  - On `PATCH /files/:id/complete`, enqueue an `index_jobs` row for `.txt`/`.md` files
  - Background worker process: polls `index_jobs` with `SELECT ... FOR UPDATE SKIP LOCKED`,
    reads the file from MinIO, extracts plain text, pushes the document into Meilisearch,
    marks the job `done`/`failed`
  - `GET /search?q=...` — queries Meilisearch, filters results down to files the user owns or has been shared

- [ ] **Module 9 — Hardening & Wrap-up**
  - Consistent request validation across all endpoints (required fields, size/type limits)
  - Consistent JSON error response shape across the whole API
  - Basic integration tests covering the upload → complete → download flow, auth flow, and sharing access checks
  - README: how to run `docker-compose up`, run migrations, and hit the API
  - Lightweight API reference (OpenAPI spec or a plain endpoint list with example payloads) for wiring up the UI
