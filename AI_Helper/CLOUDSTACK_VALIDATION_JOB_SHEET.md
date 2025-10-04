# CloudStack Validation Implementation - Job Sheet
**Date:** October 3, 2025  
**Scope:** Settings Section Integration (No Setup Wizard)  
**Target:** Production-ready CloudStack prerequisite validation

---

## Project Overview

**Goal:** Implement CloudStack validation logic into the existing settings section to prevent deployment failures due to missing/incorrect prerequisites.

**Out of Scope:**
- ❌ Setup wizard (doesn't exist)
- ❌ Automatic compute offering creation
- ❌ Auto-network selection

**In Scope:**
- ✅ OMA VM ID auto-detection with fallback
- ✅ Compute offering validation (`iscustomized: true`)
- ✅ API key account matching (hard block)
- ✅ Network selection validation
- ✅ Credential persistence (encrypted)
- ✅ Settings UI enhancements

---

## Job Breakdown

### **TASK 1: Create Validation Service** ⭐ CORE
**File:** `source/current/oma/internal/validation/cloudstack_validator.go`  
**Priority:** HIGH  
**Estimated Effort:** 3-4 hours  
**Dependencies:** None

**Subtasks:**
1. Create `internal/validation/` directory
2. Implement `CloudStackValidator` struct with client
3. Implement validation methods:
   - `DetectOMAVMID(ctx) (string, error)` - MAC address lookup
   - `ValidateComputeOffering(ctx, offeringID) error` - Check `iscustomized`
   - `ValidateAccountMatch(ctx, omaVMID) error` - Compare accounts
   - `ListAvailableNetworks(ctx) ([]Network, error)` - Network discovery
   - `ValidateNetworkExists(ctx, networkID) error` - Network validation
   - `ValidateAll(ctx) (*ValidationResult, error)` - Combined validation
4. Add comprehensive error messages (user-friendly, not technical)
5. Add logging with joblog integration

**Acceptance Criteria:**
- [ ] All validation methods implemented and tested
- [ ] Returns structured `ValidationResult` with pass/fail per check
- [ ] User-friendly error messages (no raw CloudStack errors)
- [ ] Unit tests for each validation method
- [ ] Integration with existing `ossea.Client`

**Code Structure:**
```go
type CloudStackValidator struct {
    client *ossea.Client
}

type ValidationResult struct {
    OMAVMDetection    *ValidationCheck `json:"oma_vm_detection"`
    ComputeOffering   *ValidationCheck `json:"compute_offering"`
    AccountMatch      *ValidationCheck `json:"account_match"`
    NetworkSelection  *ValidationCheck `json:"network_selection"`
    OverallStatus     string           `json:"overall_status"` // "pass", "warning", "fail"
}

type ValidationCheck struct {
    Status  string `json:"status"`  // "pass", "warning", "fail", "skipped"
    Message string `json:"message"`
    Details string `json:"details,omitempty"`
}
```

---

### **TASK 2: Add OMA API Endpoints** ⭐ CORE
**File:** `source/current/oma/api/handlers/cloudstack_settings.go`  
**Priority:** HIGH  
**Estimated Effort:** 2-3 hours  
**Dependencies:** Task 1

**Endpoints to Create:**

1. **`POST /api/v1/settings/cloudstack/validate`**
   - Purpose: Run all CloudStack validations
   - Input: CloudStack credentials (API key, secret, URL)
   - Output: `ValidationResult` JSON
   - Auth: Required

2. **`POST /api/v1/settings/cloudstack/detect-oma-vm`**
   - Purpose: Auto-detect OMA VM ID by MAC address
   - Input: CloudStack credentials
   - Output: `{ "vm_id": "...", "vm_name": "...", "mac_address": "..." }`
   - Auth: Required

3. **`GET /api/v1/settings/cloudstack/networks`**
   - Purpose: List available networks
   - Input: None (uses stored credentials)
   - Output: `[{ "id": "...", "name": "...", "zone": "...", "state": "..." }]`
   - Auth: Required

4. **`POST /api/v1/settings/cloudstack/test-connection`**
   - Purpose: Test CloudStack API connectivity
   - Input: CloudStack credentials
   - Output: `{ "success": true/false, "message": "..." }`
   - Auth: Required

**Acceptance Criteria:**
- [ ] All 4 endpoints implemented
- [ ] Proper error handling with sanitized messages
- [ ] Integration with validation service
- [ ] API documentation (comments)
- [ ] Test with curl/Postman

---

### **TASK 3: Credential Encryption & Persistence** ⭐ CORE
**Files:** 
- `source/current/oma/internal/config/cloudstack_config.go`
- `source/current/oma/database/repositories/config_repository.go`

**Priority:** HIGH  
**Estimated Effort:** 2 hours  
**Dependencies:** None

**Implementation:**
1. Add encryption methods for CloudStack credentials
   - Use existing `VMWARE_ENCRYPTION_KEY` environment variable
   - Methods: `EncryptCloudStackCredentials()`, `DecryptCloudStackCredentials()`
2. Update `ossea_configs` repository methods:
   - `SaveCloudStackConfig(config *CloudStackConfig) error`
   - `GetCloudStackConfig() (*CloudStackConfig, error)`
   - Handle encryption/decryption transparently
3. Add field validation:
   - API URL format validation
   - API key/secret non-empty checks
   - Network ID format validation

**Database Fields (Already Exist):**
```
ossea_configs table:
- vcenter_host       VARCHAR(191)  -- CloudStack API URL
- vcenter_user       VARCHAR(191)  -- API Key (encrypted)
- vcenter_password   VARCHAR(191)  -- Secret Key (encrypted)
- target_network_id  VARCHAR(191)  -- Selected Network ID
- oma_vm_id          VARCHAR(191)  -- Detected/Manual OMA VM ID
- compute_offering_id VARCHAR(191) -- Validated Compute Offering ID
```

**Acceptance Criteria:**
- [ ] Credentials encrypted before database storage
- [ ] Credentials decrypted on retrieval
- [ ] No plaintext credentials in logs
- [ ] Handle missing encryption key gracefully
- [ ] Database migrations (if schema changes needed)

---

### **TASK 4: Update Settings API Handler** ⭐ INTEGRATION
**File:** `source/current/oma/api/handlers/settings.go`  
**Priority:** MEDIUM  
**Estimated Effort:** 1-2 hours  
**Dependencies:** Task 2, Task 3

**Changes Required:**
1. Add CloudStack-specific settings routes
2. Integrate validation calls into settings save workflow
3. Add validation caching (avoid repeated API calls)
4. Return current settings with decrypted credentials for display

**Modified Endpoints:**
- `GET /api/v1/settings` - Include CloudStack config with credentials (masked)
- `POST /api/v1/settings/cloudstack` - Save CloudStack settings with validation
- `PUT /api/v1/settings/cloudstack` - Update CloudStack settings

**Validation Workflow:**
```
POST /api/v1/settings/cloudstack
  ↓
1. Decrypt existing credentials (if any)
  ↓
2. Validate required fields present
  ↓
3. Test CloudStack connection
  ↓
4. Run prerequisite validations (if requested)
  ↓
5. Encrypt new credentials
  ↓
6. Save to database
  ↓
7. Return success + validation results
```

**Acceptance Criteria:**
- [ ] Settings endpoint returns current CloudStack config
- [ ] Save endpoint validates before storing
- [ ] Credentials properly encrypted/decrypted
- [ ] Returns validation results in response
- [ ] Proper error handling for validation failures

---

### **TASK 5: GUI - Settings Page Updates** 🎨 FRONTEND
**Files:**
- `gui/src/components/Settings/CloudStackSettings.jsx` (or similar)
- `gui/src/services/cloudstack-api.js`

**Priority:** MEDIUM  
**Estimated Effort:** 3-4 hours  
**Dependencies:** Task 2, Task 4

**UI Components to Add/Update:**

1. **CloudStack Credentials Section**
   - API URL input field
   - API Key input field (masked)
   - API Secret input field (masked)
   - "Test Connection" button
   - Status indicators (✅ Connected, ❌ Failed)

2. **OMA VM Detection Section**
   - "Auto-Detect OMA VM" button
   - Display detected VM ID, name, MAC address
   - Manual VM ID input (fallback)
   - Status: ✅ Detected / ⚠️ Manual Entry Required

3. **Network Selection Section**
   - Dropdown populated from API
   - "Refresh Networks" button
   - Display: Network name, zone, state
   - Required field (no default selection)

4. **Validation Status Section**
   - "Test and Discover Resources" button
   - Display validation results:
     * ✅/❌ OMA VM Detection
     * ✅/❌ Compute Offering
     * ✅/❌ Account Match
     * ✅/❌ Network Selection
   - Error messages (user-friendly)
   - Warning messages (non-blocking)

5. **Save Button**
   - Disabled if critical validations fail
   - Shows validation summary before save
   - Success/error toasts

**Acceptance Criteria:**
- [ ] All form fields populated from database on load
- [ ] Credentials masked but editable
- [ ] Test Connection shows real-time status
- [ ] Auto-Detect OMA VM works with visual feedback
- [ ] Network dropdown populated dynamically
- [ ] Validation results displayed clearly (✅/❌)
- [ ] Save button disabled on critical failures
- [ ] User-friendly error messages (no technical jargon)
- [ ] Loading states for all async operations

---

### **TASK 6: Error Message Sanitization** 🛡️ SAFETY
**Files:** 
- `source/current/oma/internal/validation/error_messages.go`
- `source/current/oma/api/handlers/error_handlers.go`

**Priority:** MEDIUM  
**Estimated Effort:** 1 hour  
**Dependencies:** Task 1, Task 2

**Implementation:**
1. Create error message mapping:
   - Raw CloudStack errors → User-friendly messages
   - Technical errors → Actionable guidance
2. Examples:
   ```
   CloudStack: "json: cannot unmarshal string into Go struct..."
   User: "Unable to connect to CloudStack. Please verify your API URL is correct."
   
   CloudStack: "Account admin does not match account ossea"
   User: "The API key you provided belongs to a different CloudStack account than the OMA VM. Please use an API key from the same account."
   
   CloudStack: "Compute offering iscustomized=false"
   User: "The selected compute offering does not allow custom VM specifications. Please select an offering that supports custom CPU, memory, and disk sizes, or contact your CloudStack administrator."
   ```

**Acceptance Criteria:**
- [ ] All CloudStack errors have user-friendly equivalents
- [ ] Error messages include actionable guidance
- [ ] No raw API errors exposed to GUI
- [ ] Consistent error format across all endpoints
- [ ] Logging preserves technical details (for debugging)

---

### **TASK 7: Replication Blocker Logic** 🚫 SAFETY
**Files:**
- `source/current/oma/api/handlers/replication_jobs.go`
- `source/current/oma/internal/replication/job_manager.go`

**Priority:** HIGH  
**Estimated Effort:** 1-2 hours  
**Dependencies:** Task 1, Task 3

**Implementation:**
1. Add pre-flight check before starting replication jobs:
   ```go
   func (jm *JobManager) StartReplicationJob(vmName string) error {
       // Check CloudStack prerequisites
       if err := jm.validateCloudStackPrerequisites(); err != nil {
           return fmt.Errorf("cannot start replication: %w", err)
       }
       // ... existing logic
   }
   ```

2. Validation checks:
   - ✅ OMA VM ID is set (auto-detected or manual)
   - ✅ Compute offering is valid (`iscustomized: true`)
   - ✅ Network is selected
   - ✅ Account match validated

3. Clear error messages:
   ```
   "Cannot start replication: OMA VM ID not configured. Please complete CloudStack settings in the Settings page."
   
   "Cannot start replication: Selected compute offering does not support custom VM specifications. Please update your CloudStack settings."
   ```

**Acceptance Criteria:**
- [ ] Replication jobs blocked if prerequisites not met
- [ ] Clear error messages explaining what's missing
- [ ] User directed to Settings page to fix issues
- [ ] Validation cached (not repeated on every job start)
- [ ] Validation can be manually re-run from Settings

---

### **TASK 8: Documentation & Testing** 📚 QUALITY
**Files:**
- `docs/cloudstack/VALIDATION_SYSTEM.md`
- `tests/integration/cloudstack_validation_test.go`

**Priority:** LOW  
**Estimated Effort:** 2-3 hours  
**Dependencies:** All tasks

**Documentation:**
1. Create user guide:
   - How to configure CloudStack settings
   - What each validation means
   - How to resolve common errors
2. Create developer guide:
   - Validation service architecture
   - How to add new validations
   - Error message guidelines

**Testing:**
1. Unit tests for validation service
2. Integration tests for API endpoints
3. GUI testing checklist
4. Error handling scenarios

**Acceptance Criteria:**
- [ ] User documentation complete
- [ ] Developer documentation complete
- [ ] All validation methods have unit tests
- [ ] Integration tests pass
- [ ] Manual GUI testing checklist executed

---

## Implementation Order (Recommended)

### **Phase 1: Backend Core** (Day 1-2)
1. Task 1: Validation Service ⭐
2. Task 3: Credential Encryption ⭐
3. Task 2: OMA API Endpoints ⭐

**Milestone:** Backend API functional and testable with curl/Postman

### **Phase 2: Integration & Safety** (Day 2-3)
4. Task 4: Settings API Handler
5. Task 6: Error Message Sanitization
6. Task 7: Replication Blocker Logic

**Milestone:** Complete backend integration with replication blocking

### **Phase 3: Frontend** (Day 3-4)
7. Task 5: GUI Settings Page Updates 🎨

**Milestone:** Full user interface for CloudStack validation

### **Phase 4: Polish & Quality** (Day 4-5)
8. Task 8: Documentation & Testing 📚

**Milestone:** Production-ready system with documentation

---

## Risk Assessment

### **Low Risk:**
- ✅ All validation methods tested and working
- ✅ CloudStack SDK bugs already fixed
- ✅ Database schema already supports required fields
- ✅ Encryption key infrastructure exists

### **Medium Risk:**
- ⚠️ GUI integration (depends on existing frontend architecture)
- ⚠️ Error message coverage (need to handle all CloudStack error types)

### **Mitigation:**
- Start with backend (testable via API)
- Create comprehensive error message mapping
- Test with multiple CloudStack configurations
- Get user feedback early

---

## Success Criteria

### **Must Have (MVP):**
- ✅ OMA VM ID auto-detection working
- ✅ Compute offering validation blocking invalid offerings
- ✅ Account mismatch detection with hard block
- ✅ Network selection required before replication
- ✅ Credentials encrypted and persisted
- ✅ Replication jobs blocked if prerequisites not met
- ✅ User-friendly error messages in GUI

### **Nice to Have (Future):**
- 🔮 Validation result caching with TTL
- 🔮 CloudStack API health monitoring
- 🔮 Automatic compute offering creation (if permissions allow)
- 🔮 Multi-network support per VM
- 🔮 Validation history/audit trail

---

## Testing Checklist

### **Backend Testing:**
- [ ] Test OMA VM detection with correct MAC
- [ ] Test OMA VM detection with incorrect MAC (fallback)
- [ ] Test compute offering validation with valid offering
- [ ] Test compute offering validation with invalid offering
- [ ] Test account match with matching accounts
- [ ] Test account match with mismatched accounts
- [ ] Test network listing with multiple networks
- [ ] Test network validation with valid network ID
- [ ] Test network validation with invalid network ID
- [ ] Test credential encryption/decryption
- [ ] Test replication blocker with missing prerequisites
- [ ] Test replication blocker with valid prerequisites

### **Frontend Testing:**
- [ ] Settings page loads with current values
- [ ] Test Connection button shows real-time status
- [ ] Auto-Detect OMA VM displays results correctly
- [ ] Network dropdown populates from API
- [ ] Validation results display clearly (✅/❌)
- [ ] Save button disabled when validations fail
- [ ] Error messages are user-friendly
- [ ] Success toasts appear on save
- [ ] Loading states work correctly
- [ ] Form validation prevents empty submissions

### **Integration Testing:**
- [ ] End-to-end: Configure CloudStack → Validate → Start Replication
- [ ] Error path: Invalid credentials → Clear error message
- [ ] Error path: Account mismatch → Replication blocked
- [ ] Error path: No network selected → Save blocked
- [ ] Success path: All validations pass → Replication allowed

---

## Effort Estimate

**Total Estimated Time:** 15-20 hours

**Breakdown:**
- Backend Core (Tasks 1-3): 7-9 hours
- Integration (Tasks 4, 6, 7): 4-5 hours
- Frontend (Task 5): 3-4 hours
- Documentation/Testing (Task 8): 2-3 hours

**Recommended Sprint:** 1 week (part-time) or 2-3 days (full-time)

---

## Notes

- All validation logic tested against real CloudStack instance (http://10.245.241.101:8080)
- Test script available at `source/current/oma/test-cs.go` for reference
- CloudStack SDK workarounds already implemented in `ossea/client.go`
- Encryption key infrastructure already exists (`VMWARE_ENCRYPTION_KEY`)
- Database schema already supports all required fields
- No database migrations required

---

## Questions for Clarification

1. **GUI Framework:** What frontend framework are you using? (React, Vue, Angular, vanilla JS?)
2. **Settings Page Location:** Where is the current settings page located in the GUI?
3. **API Authentication:** What auth mechanism is used for OMA API endpoints?
4. **Error Handling:** Do you have a standard error response format?
5. **Logging:** Are you using joblog for all business logic or just specific operations?

---

## Ready to Start?

**Recommendation:** Start with **Phase 1 (Backend Core)** to get the validation service functional and testable via API. Once that's solid, move to integration and then GUI.

**First Task:** Task 1 (Validation Service) - Estimated 3-4 hours

Let me know when you're ready to begin, and I can start with Task 1!


