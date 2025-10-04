API Documentation

Scope: OMA API and VMA API endpoints currently implemented in `sendense/source/current`. Each endpoint lists path, method, handler, description, known callsites, and classification:

- Key: actively used by core flows (replication, discovery, failover, progress, credentials)
- Auxiliary: used for health/debug/admin, not on critical path
- Potentially legacy: present but no in-repo callsites or superseded; may be GUI-only or slated for consolidation
- Legacy (avoid): duplicates older engine paths or unimplemented expectations; keep for backward compatibility only

Files:
- OMA endpoints: `./OMA.md`
- VMA endpoints: `./VMA.md`

Cross-links/References:
- OMA routes defined in `oma/api/server.go`; handlers under `oma/api/handlers/*`
- VMA routes defined in `vma/api/server.go`; progress routes in `vma/api/progress_handler.go`
- Known callsites include: `oma/services/*`, `oma/failover/*`, `oma/workflows/*`, `migratekit/internal/*`, `vma/client/*`, `vma/services/*`

Notes:
- Canonical source per project rules is under `sendense/source/current/`.
- Classifications are based on code references in this repo; externally-consumed GUI calls may not appear here but could still be active in production.

