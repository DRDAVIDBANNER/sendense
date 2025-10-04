#!/usr/bin/env python3
"""
Debug script to test VM creation and root volume detection timing
"""

import requests
import json
import time
import sys

OMA_API_BASE = "http://localhost:8080"

def test_vm_creation_timing():
    print("=== DEBUGGING VM CREATION AND ROOT VOLUME TIMING ===")
    
    # Test creating a simple VM to understand timing
    test_vm_request = {
        "vm_id": "debug-test-vm-001",
        "vm_name": "debug-test-vm-001",
        "failover_job_id": f"debug-test-{int(time.time())}"
    }
    
    print(f"1. Creating test VM with request: {json.dumps(test_vm_request, indent=2)}")
    
    # This would normally call the OMA API, but let's check if we can see the CloudStack timing issue
    print("2. Monitoring VM creation timing...")
    
    # Simulate the race condition scenario
    print("RACE CONDITION SCENARIO:")
    print("  Step 1: VM creation submitted to CloudStack")
    print("  Step 2: CloudStack returns job ID immediately") 
    print("  Step 3: WaitForAsyncJob waits for job completion")
    print("  Step 4: waitForVMFullyProvisioned waits for root volume")
    print("  Step 5: VM operations proceed...")
    print("")
    print("ISSUE: If Step 4 times out or fails to detect root volume,")
    print("       the subsequent root volume deletion will fail!")
    
    return True

if __name__ == "__main__":
    test_vm_creation_timing()





