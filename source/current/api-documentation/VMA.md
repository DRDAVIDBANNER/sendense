VMA API Endpoints (VMA)

Base: /api/v1 (router in `vma/api/server.go`; progress routes in `vma/api/progress_handler.go`)

Health
- GET /health → `VMAControlServer.handleHealth`
  - Classification: Auxiliary

Progress
- GET /progress/{jobId} → `ProgressHandler.GetJobProgress`
  - Callsites: OMA `GET /api/v1/progress/{job_id}` proxy, GUI
  - Classification: Key
- POST /progress/{jobId}/update → `ProgressHandler.UpdateJobProgress`
  - Callsites: migratekit `internal/progress/vma_client.go`
  - Classification: Key

Job Status and Cleanup
- GET /status/{job_id} → `VMAControlServer.handleStatus`
  - Callsites: OMA `services/VMAProgressClient` uses `/status/{job_id}`
  - Classification: Key
- POST /cleanup → `VMAControlServer.handleCleanup`
  - Callsites: none detected in OMA; exists for future/maintenance
  - Classification: Auxiliary/Potentially legacy

Configuration
- PUT /config → `VMAControlServer.handleConfig`
  - Classification: Auxiliary

VMware Discovery and Replication
- POST /discover → `VMAControlServer.handleDiscover`
  - Callsites: OMA scheduler service, unified failover engine, enhanced discovery service
  - Classification: Key
- POST /replicate → `VMAControlServer.handleReplicate`
  - Callsites: OMA legacy workflow `workflows/migration.go` (direct call)
  - Classification: Potentially legacy (newer path creates OMA job, not VMA replicate)

VM Spec Changes
- POST /vm-spec-changes → `VMAControlServer.handleVMSpecChanges`
  - Callsites: none detected; used by change detection feature
  - Classification: Auxiliary

Power Management (Unified Failover)
- POST /vm/{vm_id}/power-off → `handleVMPowerOff`
- POST /vm/{vm_id}/power-on → `handleVMPowerOn`
- GET  /vm/{vm_id}/power-state → `handleVMPowerState`
  - Callsites: OMA `failover/vma_client.go`
  - Classification: Key

CBT (Change Block Tracking)
- GET /vms/{vm_path}/cbt-status → `handleCBTStatus`
  - Callsites: migratekit VMA client expects enable-cbt and cbt-status; only cbt-status is implemented here
  - Classification: Key (status); Note: enable-cbt path referenced by migratekit client is not present in VMA API server

Enrollment (VMA ↔ OMA)
- POST /enrollment/enroll → `handleEnrollWithOMA`
- GET  /enrollment/status → `handleEnrollmentStatus`
  - Callsites: VMA side initiates against OMA public endpoints (`/api/v1/vma/enroll*`) through `services/enrollment_client.go`
  - Classification: Key

Implementation Notes and Legacy Flags
- Progress route previously existed in server; now centralized in `ProgressHandler.RegisterRoutes`. Server comment indicates old route removed.
- `/replicate` usage appears only in older OMA workflow (`workflows/migration.go`); primary flow uses OMA `/replications` and VMA only for discovery/progress.
- CBT enable endpoint is referenced by migratekit client (`/api/v1/vms{vm_path}/enable-cbt`) but is not implemented in current VMA API; treat client call as legacy/unused unless wired elsewhere.

