OMA API Endpoints (OMA)

Base: /api/v1 (router in `oma/api/server.go`)

Authentication
- POST /auth/login → `handlers.Auth.Login`
  - Description: Issue bearer token for OMA API
  - Callsites: GUI/auth expected; no direct backend calls detected
  - Classification: Key (gateway)

Health/Swagger
- GET /health → inline `handleHealth`
  - Classification: Auxiliary
- GET /swagger/* → swagger handler
  - Classification: Auxiliary

VM Inventory
- GET /vms → `handlers.VM.List`
- POST /vms/inventory → `handlers.VM.ReceiveInventory`
- GET /vms/{id} → `handlers.VM.GetByID`
  - Classification: Key (GUI-driven)

Replications
- GET /replications → `handlers.Replication.List`
- POST /replications → `handlers.Replication.Create`
- GET /replications/{id} → `handlers.Replication.GetByID`
- PUT /replications/{id} → `handlers.Replication.Update`
- DELETE /replications/{id} → `handlers.Replication.Delete`
  - Callsites: `vma/client/oma_client.go` (POST), scheduler service (POST), unified failover engine (POST)
  - Classification: Key

ChangeID (CBT)
- GET /replications/changeid?vm_path={}&disk_id={} → `handlers.Replication.GetPreviousChangeID`
  - Callsites: migratekit `internal/target/cloudstack.go` and `vma/client`
  - Classification: Key
- POST /replications/{job_id}/changeid → `handlers.Replication.StoreChangeID`
  - Callsites: migratekit `internal/target/cloudstack.go` stores after completion
  - Classification: Key

Progress Proxy and Job Status
- GET /progress/{job_id} → `handlers.Replication.GetVMAProgressProxy`
  - Callsites: GUI; OMA constructs `http://localhost:9081/api/v1/progress/{job_id}` to VMA
  - Classification: Key
- GET /replications/{job_id}/progress → `handlers.Replication.GetReplicationProgress`
  - Classification: Key (enhanced progress via DB + VMA fields)

VM-Centric Architecture
- GET /vm-contexts → `handlers.VMContext.ListVMContexts`
- GET /vm-contexts/{vm_name} → `handlers.VMContext.GetVMContext`
- GET /vm-contexts/{context_id}/recent-jobs → `handlers.VMContext.GetRecentJobs`
  - Classification: Key (GUI relies on these)

Discovery (Enhanced)
- POST /discovery/discover-vms → `handlers.EnhancedDiscovery.DiscoverVMs`
- POST /discovery/add-vms → `handlers.EnhancedDiscovery.AddVMs`
- POST /discovery/bulk-add → `handlers.EnhancedDiscovery.BulkAddVMs`
- GET /discovery/ungrouped-vms → `handlers.EnhancedDiscovery.GetUngroupedVMs`
- POST /discovery/preview → `handlers.EnhancedDiscovery.GetDiscoveryPreview`
- GET /vm-contexts/ungrouped → alias to ungrouped-vms
  - Callsites: services use VMA `/api/v1/discover` and add entries without jobs
  - Classification: Key

VMware Credentials Management
- GET /vmware-credentials → list
- POST /vmware-credentials → create
- GET /vmware-credentials/{id} → get
- PUT /vmware-credentials/{id} → update
- DELETE /vmware-credentials/{id} → delete
- PUT /vmware-credentials/{id}/set-default → set default
- POST /vmware-credentials/{id}/test → test
- GET /vmware-credentials/default → get default
  - Handler: `handlers.VMwareCredentials.*`
  - Classification: Key (feeds discovery flows)

CloudStack (OSSEA) Settings
- POST /settings/cloudstack/test-connection → `handlers.CloudStackSettings.TestConnection`
- POST /settings/cloudstack/detect-oma-vm → `handlers.CloudStackSettings.DetectOMAVM`
- GET /settings/cloudstack/networks → `handlers.CloudStackSettings.ListNetworks`
- POST /settings/cloudstack/validate → `handlers.CloudStackSettings.ValidateSettings`
- POST /settings/cloudstack/discover-all → `handlers.CloudStackSettings.DiscoverAllResources`
  - Classification: Key/Auxiliary (setup flows)

OSSEA Config
- POST /ossea/config → `handlers.OSSEA.HandleConfig`
- POST /ossea/discover-resources → `handlers.StreamlinedOSSEA.DiscoverResources`
- POST /ossea/config-streamlined → `handlers.StreamlinedOSSEA.SaveStreamlinedConfig`
  - Classification: Key (platform setup)

Network Mapping and Service Offerings
- POST /network-mappings → create
- GET /network-mappings → list all
- GET /network-mappings/{vm_id} → get by VM
- GET /network-mappings/{vm_id}/status → status
- DELETE /network-mappings/{vm_id}/{source_network_name} → delete
- GET /networks/available → list OSSEA networks
- POST /networks/resolve → resolve network IDs
- GET /service-offerings/available → list offerings
  - Handlers: `handlers.NetworkMapping.*`
  - Classification: Key

Failover (Enhanced + Unified)
- POST /failover/live → `handlers.Failover.InitiateEnhancedLiveFailover`
- POST /failover/test → `handlers.Failover.InitiateEnhancedTestFailover`
- DELETE /failover/test/{job_id} → `handlers.Failover.EndTestFailover`
- POST /failover/cleanup/{vm_name} → `handlers.Failover.CleanupTestFailover`
- POST /failover/{vm_name}/cleanup-failed → `handlers.Failover.CleanupFailedExecution`
- GET /failover/{job_id}/status → `handlers.Failover.GetFailoverJobStatus`
- GET /failover/{vm_id}/readiness → `handlers.Failover.ValidateFailoverReadiness`
- GET /failover/jobs → `handlers.Failover.ListFailoverJobs`
- POST /failover/unified → unified orchestrator
- GET /failover/preflight/config/{failover_type}/{vm_name} → preflight config
- POST /failover/preflight/validate → preflight validate
- POST /failover/rollback → enhanced rollback
- GET /failover/rollback/decision/{failover_type}/{vm_name} → rollback decision
  - Callsites: unified engine invokes VMA `/discover` and OMA `/replications`; cleanup uses Volume Daemon APIs
  - Classification: Key (core), with some Auxiliary (preflight/rollback decision)

Scheduler Ecosystem
- POST /schedules, GET /schedules, GET/PUT/DELETE /schedules/{id}
- POST /schedules/{id}/enable, POST /schedules/{id}/trigger, GET /schedules/{id}/executions
  - Handlers: `handlers.ScheduleManagement.*`
  - Classification: Key (automation)

Machine Groups and VM Assignments
- CRUD /machine-groups and VM assignment endpoints
  - Handlers: `handlers.MachineGroupManagement.*`, `handlers.VMGroupAssignment.*`
  - Classification: Key

Validation
- GET /vms/{vm_id}/failover-readiness
- GET /vms/{vm_id}/sync-status
- GET /vms/{vm_id}/network-mapping-status
- GET /vms/{vm_id}/volume-status
- GET /vms/{vm_id}/active-jobs
- GET /vms/{vm_id}/configuration-check
  - Handlers: `handlers.Validation.*` (also exposes RegisterValidationRoutes)
  - Classification: Key/Auxiliary

Debug
- GET /debug/health, /debug/failover-jobs, /debug/endpoints, /debug/logs
  - Classification: Auxiliary

VMA Enrollment (OMA side)
- Admin: POST /admin/vma/pairing-code; GET /admin/vma/pending; POST /admin/vma/approve/{id}; GET /admin/vma/active; POST /admin/vma/reject/{id}; DELETE /admin/vma/revoke/{id}; GET /admin/vma/audit
- Public: POST /vma/enroll; POST /vma/enroll/verify; GET /vma/enroll/result
  - Handler: `handlers.VMAReal.*`
  - Callsites: VMA `services/enrollment_client.go` uses public endpoints via 443 proxy
  - Classification: Key

Legacy/Potentially Legacy Notes
- Original failover handlers exist alongside enhanced; enhanced/unified are primary. The `RegisterFailoverRoutes` exports classic paths; prefer enhanced/unified.
- `vma_simple.go` defines simple enrollment handlers but is not wired in `handlers.NewHandlers`; Classification: Legacy (unused).

