# Deferred Decisions

Track decisions intentionally postponed so they don't get lost. Update this file as new
"later" items surface during the build.

## Frontend

Robust UI planned as a separate effort after the backend is built and tested.
Backend exposes a clean read API (rollups vs targets, recent meals, manual correction);
the frontend sits on top. Core must boot fully headless (API only) with the dashboard
feature flag off.

## Food DBs

Decide whether to bundle datasets in the image, fetch on first boot, or fetch on demand.
Resolution impacts cold-start latency and image size.

- **Open Food Facts (OFF):** ODbL license. Check attribution and share-alike requirements
  if bundling or redistributing.
- **TACO (Brazilian):** Check the current TACO terms of use for redistribution rights.

Decision deferred until the NutritionSource adapter is implemented and we can measure
bundle size vs. fetch latency.

## Embedding index storage

In-DB (SQLite with a vector extension or a simple flat index in a BLOB) vs. a dedicated
sidecar vector store (Chromadb, Qdrant, or pgvector behind Postgres).

Decide once Tier-1 deterministic parsing is working and we have real embedding dimensions
and throughput numbers.

## Auth / multi-user

All schema tables are keyed by `user_id` from day one, but the system currently runs as
single-user. The auth mechanism (API keys, OAuth, simple token) and multi-user gating
will be added when the `MULTI_USER` feature flag is introduced.

## Per-user timezone override

Currently handled by the single `DEFAULT_TIMEZONE` env var for the sole user. Per-user
override lands alongside multi-user support.
