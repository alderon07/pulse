# TODO

## Completed

- [x] Neon Serverless Postgres integration
  - Neon project created (`ep-blue-boat-ai4sxosz-pooler.c-4.us-east-1.aws.neon.tech`)
  - Connection string configured in `backend/.env.local`
  - All migrations applied (00001_init, 00002_auto_schedule_learning)
  - Using pooled connection endpoint for serverless compatibility
- [x] Root-level justfile and mise.toml
  - Moved from `backend/` to project root
  - `just dev` runs frontend + backend API in parallel (Ctrl-C stops both)
  - `just dev-all` also includes the worker process
  - All backend, frontend, migration, and docker recipes available
  - mise manages: just, node, go, goose, sqlc, oapi-codegen, golangci-lint

## In Progress

## Planned

- [ ] Configure Neon MCP server for in-editor database management
- [ ] Set up Neon branching for preview/staging environments
- [ ] Configure connection pooling settings for production
