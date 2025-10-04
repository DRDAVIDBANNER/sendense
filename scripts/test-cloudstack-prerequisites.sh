#!/bin/bash
# CloudStack Prerequisite Testing Script
# Tests actual CloudStack behavior before implementing validation
#
# Usage: ./test-cloudstack-prerequisites.sh <cloudstack_url> <api_key> <secret_key>

set -e

CLOUDSTACK_URL="${1}"
API_KEY="${2}"
SECRET_KEY="${3}"

if [ -z "$CLOUDSTACK_URL" ] || [ -z "$API_KEY" ] || [ -z "$SECRET_KEY" ]; then
    echo "Usage: $0 <cloudstack_url> <api_key> <secret_key>"
    echo "Example: $0 http://10.245.241.101:8080 your-api-key your-secret-key"
    exit 1
fi

# Ensure URL ends with /client/api
if [[ ! "$CLOUDSTACK_URL" =~ /client/api$ ]]; then
    CLOUDSTACK_URL="${CLOUDSTACK_URL}/client/api"
fi

echo "========================================"
echo "CloudStack Prerequisite Testing"
echo "========================================"
echo "CloudStack: $CLOUDSTACK_URL"
echo ""

# Function to make CloudStack API calls
cloudstack_api() {
    local command="$1"
    local params="$2"
    
    # CloudStack signature generation would go here
    # For now, using basic auth (adjust as needed)
    curl -s "${CLOUDSTACK_URL}?command=${command}&response=json&apiKey=${API_KEY}${params}" \
        -H "Content-Type: application/json"
}

echo "========================================"
echo "TEST 1: MAC Address VM Detection"
echo "========================================"
echo ""

echo "1.1 Getting OMA's MAC addresses..."
echo "Available network interfaces:"
ip link show | grep -E "^[0-9]+:" | awk '{print $2}' | tr -d ':'

echo ""
echo "MAC addresses:"
for iface in $(ip link show | grep -E "^[0-9]+:" | awk '{print $2}' | tr -d ':'); do
    mac=$(ip link show "$iface" | grep "link/ether" | awk '{print $2}')
    if [ ! -z "$mac" ]; then
        echo "  $iface: $mac"
    fi
done

echo ""
echo "1.2 Testing CloudStack VM listing with NIC info..."
echo "Querying CloudStack for all VMs..."

VMS_RESPONSE=$(cloudstack_api "listVirtualMachines" "")
echo "$VMS_RESPONSE" | jq -r '.listvirtualmachinesresponse.virtualmachine[]? | 
    {
        id: .id, 
        name: .name, 
        account: .account,
        nics: [.nic[]? | {macaddress, ipaddress, networkname}]
    }' 2>/dev/null || echo "Failed to parse VM response"

echo ""
echo "Question: Can we reliably find OMA VM by MAC address in this output?"
echo ""

echo "========================================"
echo "TEST 2: Service Offering Custom Disk Check"
echo "========================================"
echo ""

echo "2.1 Listing all service offerings..."
OFFERINGS_RESPONSE=$(cloudstack_api "listServiceOfferings" "")

echo "Service offerings with disk customization info:"
echo "$OFFERINGS_RESPONSE" | jq -r '.listserviceofferingsresponse.serviceoffering[]? | 
    {
        id: .id,
        name: .name,
        displaytext: .displaytext,
        cpunumber: .cpunumber,
        memory: .memory,
        customized: .customized,
        customizeddisk: (.customizeddisk // "not_present"),
        rootdisksize: (.rootdisksize // 0)
    }' 2>/dev/null || echo "Failed to parse offerings"

echo ""
echo "Question: Which field indicates 'allows custom root disk size'?"
echo "  - customized: true?"
echo "  - customizeddisk: true?"
echo "  - rootdisksize: 0?"
echo ""

echo "========================================"
echo "TEST 3: Disk Offerings"
echo "========================================"
echo ""

echo "3.1 Listing disk offerings..."
DISK_OFFERINGS_RESPONSE=$(cloudstack_api "listDiskOfferings" "")

echo "Disk offerings:"
echo "$DISK_OFFERINGS_RESPONSE" | jq -r '.listdiskofferingsresponse.diskoffering[]? | 
    {
        id: .id,
        name: .name,
        displaytext: .displaytext,
        disksize: .disksize,
        customized: .customized,
        customizediops: (.customizediops // "not_present")
    }' 2>/dev/null || echo "Failed to parse disk offerings"

echo ""

echo "========================================"
echo "TEST 4: Account Validation"
echo "========================================"
echo ""

echo "4.1 Getting current API session account..."
ACCOUNT_RESPONSE=$(cloudstack_api "listAccounts" "&listall=true")

echo "Accounts accessible with these API keys:"
echo "$ACCOUNT_RESPONSE" | jq -r '.listaccountsresponse.account[]? | 
    {
        id: .id,
        name: .name,
        accounttype: .accounttype,
        domain: .domain,
        domainid: .domainid
    }' 2>/dev/null || echo "Failed to parse accounts"

echo ""
echo "4.2 How to determine which account owns the API key?"
echo ""

echo "========================================"
echo "TEST 5: Current User/Account Info"
echo "========================================"
echo ""

echo "5.1 Checking if CloudStack has 'whoami' equivalent..."

# Try listUsers to see current user
USERS_RESPONSE=$(cloudstack_api "listUsers" "")
echo "Users response:"
echo "$USERS_RESPONSE" | jq -r '.listusersresponse.user[]? | 
    {
        id: .id,
        username: .username,
        account: .account,
        accounttype: .accounttype,
        domain: .domain
    }' 2>/dev/null || echo "Failed to parse users"

echo ""

echo "========================================"
echo "TEST 6: Creating Service Offering (Admin Required)"
echo "========================================"
echo ""

echo "6.1 Testing if we can create a service offering..."
echo "This requires admin permissions - will likely fail"
echo ""
echo "Params needed for createServiceOffering:"
echo "  - name: Migration-Custom"
echo "  - displaytext: Custom CPU/Memory/Disk for Migrations"
echo "  - cpunumber: customizable"
echo "  - cpuspeed: customizable" 
echo "  - memory: customizable"
echo "  - customized: true"
echo ""
echo "Skipping actual creation in test - would be:"
echo '  createServiceOffering&name=Migration-Custom&displaytext=Custom&customized=true'
echo ""

echo "========================================"
echo "TEST 7: VM Details with Account Info"
echo "========================================"
echo ""

echo "7.1 Getting a sample VM with full details..."
SAMPLE_VM_ID=$(echo "$VMS_RESPONSE" | jq -r '.listvirtualmachinesresponse.virtualmachine[0]?.id // empty' 2>/dev/null)

if [ ! -z "$SAMPLE_VM_ID" ]; then
    echo "Sample VM ID: $SAMPLE_VM_ID"
    VM_DETAIL=$(cloudstack_api "listVirtualMachines" "&id=$SAMPLE_VM_ID")
    
    echo "VM account information:"
    echo "$VM_DETAIL" | jq -r '.listvirtualmachinesresponse.virtualmachine[0]? | 
        {
            id: .id,
            name: .name,
            account: .account,
            accountid: .accountid,
            domain: .domain,
            domainid: .domainid
        }' 2>/dev/null || echo "Failed to parse VM details"
else
    echo "No VMs found to test with"
fi

echo ""

echo "========================================"
echo "SUMMARY OF FINDINGS"
echo "========================================"
echo ""
echo "Please review the output above and answer:"
echo ""
echo "1. MAC Detection:"
echo "   - Can we find VMs by MAC address? Y/N"
echo "   - Which interface MAC should we use?"
echo ""
echo "2. Custom Disk Offerings:"
echo "   - Which field indicates custom root disk support?"
echo "   - Do any existing offerings support it?"
echo ""
echo "3. Account Validation:"
echo "   - How do we determine which account the API key belongs to?"
echo "   - Can we compare it to VM owner account?"
echo ""
echo "4. Service Offering Creation:"
echo "   - Do we have permission to create offerings?"
echo "   - What would happen if we try?"
echo ""
echo "========================================"


