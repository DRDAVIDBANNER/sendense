"use client";

import { useState, useEffect } from "react";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Progress } from "@/components/ui/progress";
import { Server, CheckCircle, Loader2, AlertCircle } from "lucide-react";

interface VMwareCredential {
  id: number;                    // Backend returns number not string
  credential_name: string;       // Match backend field name
  vcenter_host: string;
  username: string;
  datacenter: string;            // Add datacenter field
  is_active: boolean;
  is_default: boolean;
  created_at: string;
  updated_at: string;
  created_by: string;
  last_used: string | null;
  usage_count: number;
}

interface DiscoveredVM {
  id: string;                    // ✅ Match backend field name
  name: string;                  // ✅ Match backend field name
  path: string;                  // ✅ Add path field
  power_state: 'poweredOn' | 'poweredOff' | 'suspended';
  guest_os: string;
  num_cpu: number;               // ✅ Match backend field name
  memory_mb: number;
  vmx_version?: string;          // ✅ Add version field
  disks?: any[];                 // ✅ Add disk info
  networks?: any[];              // ✅ Add network info
  existing?: boolean;            // ✅ Add existing flag
}

interface VMDiscoveryModalProps {
  isOpen: boolean;
  onClose: () => void;
  onDiscoveryComplete: () => void;
}

export function VMDiscoveryModal({ isOpen, onClose, onDiscoveryComplete }: VMDiscoveryModalProps) {
  const [currentStep, setCurrentStep] = useState(1);
  const [credentials, setCredentials] = useState<VMwareCredential[]>([]);
  const [selectedCredentialId, setSelectedCredentialId] = useState<number | null>(null);
  const [discoveredVMs, setDiscoveredVMs] = useState<DiscoveredVM[]>([]);
  const [selectedVMIds, setSelectedVMIds] = useState<string[]>([]);

  // Loading and error states
  const [isLoadingCredentials, setIsLoadingCredentials] = useState(false);
  const [isTestingConnection, setIsTestingConnection] = useState(false);
  const [isDiscovering, setIsDiscovering] = useState(false);
  const [isAddingVMs, setIsAddingVMs] = useState(false);
  const [connectionTestResult, setConnectionTestResult] = useState<{ success: boolean; message: string } | null>(null);
  const [discoveryProgress, setDiscoveryProgress] = useState(0);
  const [error, setError] = useState<string | null>(null);

  // Reset modal state when opened
  const resetModal = () => {
    setCurrentStep(1);
    setSelectedCredentialId(null);
    setDiscoveredVMs([]);
    setSelectedVMIds([]);
    setConnectionTestResult(null);
    setError(null);
    setDiscoveryProgress(0);
  };

  // Load credentials when modal opens
  const loadCredentials = async () => {
    if (!isOpen) return;

    setIsLoadingCredentials(true);
    try {
      const response = await fetch('/api/v1/vmware-credentials');
      if (response.ok) {
        const data = await response.json();
        setCredentials(data.credentials || []);
      } else {
        setError('Failed to load VMware credentials');
      }
    } catch (err) {
      setError('Failed to load VMware credentials');
      console.error('Error loading credentials:', err);
    } finally {
      setIsLoadingCredentials(false);
    }
  };

  // Test connection to selected vCenter
  const testConnection = async () => {
    if (!selectedCredentialId) return;

    setIsTestingConnection(true);
    setConnectionTestResult(null);

    try {
      const response = await fetch(`/api/v1/vmware-credentials/${selectedCredentialId}/test`, {
        method: 'POST'
      });
      const result = await response.json();

      setConnectionTestResult({
        success: response.ok,
        message: result.message || (response.ok ? 'Connection successful' : 'Connection failed')
      });
    } catch (err) {
      setConnectionTestResult({
        success: false,
        message: 'Failed to test connection'
      });
    } finally {
      setIsTestingConnection(false);
    }
  };

  // Discover VMs from selected vCenter
  const discoverVMs = async () => {
    if (!selectedCredentialId) return;

    setIsDiscovering(true);
    setDiscoveryProgress(0);
    setError(null);

    try {
      const response = await fetch('/api/v1/discovery/discover-vms', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          credential_id: selectedCredentialId
        }),
      });

      if (response.ok) {
        const result = await response.json();
        setDiscoveredVMs(result.discovered_vms || []);
        setDiscoveryProgress(100);
      } else {
        const errorResult = await response.json();
        setError(errorResult.message || 'Failed to discover VMs');
      }
    } catch (err) {
      setError('Failed to discover VMs');
      console.error('Error discovering VMs:', err);
    } finally {
      setIsDiscovering(false);
    }
  };

  // Add selected VMs to management
  const addVMsToManagement = async () => {
    if (selectedVMIds.length === 0) return;

    setIsAddingVMs(true);
    setError(null);

    try {
      // Get VM names from IDs
      const selectedVMNames = selectedVMIds
        .map(id => discoveredVMs.find(vm => vm.id === id)?.name)
        .filter((name): name is string => !!name);

      const response = await fetch('/api/v1/discovery/add-vms', {  // ✅ CORRECT ENDPOINT
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          credential_id: selectedCredentialId, // ✅ Supported by this endpoint
          vm_names: selectedVMNames,           // ✅ Correct field name
          added_by: 'gui-user'                 // ✅ Track who added these VMs
        }),
      });

      if (response.ok) {
        const result = await response.json();
        console.log(`Successfully added ${result.vms_added || selectedVMNames.length} VMs to management`);
        onDiscoveryComplete();
        onClose();
        resetModal();
      } else {
        const errorResult = await response.json();
        setError(errorResult.error || errorResult.message || 'Failed to add VMs to management');
      }
    } catch (err) {
      setError('Failed to add VMs to management');
      console.error('Error adding VMs:', err);
    } finally {
      setIsAddingVMs(false);
    }
  };

  // Handle modal open/close
  // Load credentials when modal opens
  useEffect(() => {
    if (isOpen) {
      resetModal();
      loadCredentials();
    }
  }, [isOpen]);

  const handleOpenChange = (open: boolean) => {
    if (!open) {
      onClose();
    }
  };

  const handleNext = () => {
    if (currentStep < 3) {
      setCurrentStep(currentStep + 1);

      // Auto-discover VMs when moving to step 2
      if (currentStep === 1 && selectedCredentialId) {
        discoverVMs();
      }
    }
  };

  const handlePrevious = () => {
    if (currentStep > 1) {
      setCurrentStep(currentStep - 1);
    }
  };

  const handleVMSelection = (vmId: string, checked: boolean) => {
    setSelectedVMIds(prev =>
      checked
        ? [...prev, vmId]
        : prev.filter(id => id !== vmId)
    );
  };

  const canProceedToNext = () => {
    switch (currentStep) {
      case 1:
        return selectedCredentialId && connectionTestResult?.success;
      case 2:
        return !isDiscovering && discoveredVMs.length >= 0; // Allow proceeding even with 0 VMs
      case 3:
        return selectedVMIds.length > 0;
      default:
        return false;
    }
  };

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'poweredOn':
        return <Badge className="bg-green-500/10 text-green-400 border-green-500/20">Running</Badge>;
      case 'poweredOff':
        return <Badge className="bg-gray-500/10 text-gray-400 border-gray-500/20">Stopped</Badge>;
      case 'suspended':
        return <Badge className="bg-yellow-500/10 text-yellow-400 border-yellow-500/20">Suspended</Badge>;
      default:
        return <Badge variant="secondary">Unknown</Badge>;
    }
  };

  return (
    <Dialog open={isOpen} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-[700px] max-h-[90vh] overflow-hidden">
        <DialogHeader>
          <DialogTitle>Discover Virtual Machines</DialogTitle>
          <DialogDescription>
            Connect to vCenter and discover VMs to add to management.
          </DialogDescription>
        </DialogHeader>

        {/* Progress Indicator */}
        <div className="flex items-center justify-center mb-6">
          <div className="flex items-center space-x-4">
            {[1, 2, 3].map((step) => (
              <div key={step} className="flex items-center">
                <div className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium ${
                  step <= currentStep
                    ? 'bg-primary text-primary-foreground'
                    : 'bg-muted text-muted-foreground'
                }`}>
                  {step}
                </div>
                {step < 3 && (
                  <div className={`w-12 h-0.5 mx-2 ${
                    step < currentStep ? 'bg-primary' : 'bg-muted'
                  }`} />
                )}
              </div>
            ))}
          </div>
        </div>

        <div className="flex-1 overflow-auto">
          {error && (
            <div className="mb-4 p-3 bg-destructive/10 border border-destructive/20 rounded-lg flex items-center gap-2">
              <AlertCircle className="h-4 w-4 text-destructive" />
              <span className="text-sm text-destructive">{error}</span>
            </div>
          )}

          {currentStep === 1 && (
            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="vcenter-credential">vCenter Connection</Label>
                <Select
                  value={selectedCredentialId?.toString() || ""}
                  onValueChange={(value) => setSelectedCredentialId(value ? parseInt(value) : null)}
                  disabled={isLoadingCredentials}
                >
                  <SelectTrigger>
                    <SelectValue placeholder={isLoadingCredentials ? "Loading credentials..." : "Select vCenter connection"} />
                  </SelectTrigger>
                  <SelectContent>
                    {credentials.map((cred) => (
                      <SelectItem key={cred.id} value={cred.id.toString()}>
                        {cred.credential_name} ({cred.vcenter_host})
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              {selectedCredentialId && (
                <div className="space-y-3">
                  <Button
                    type="button"
                    variant="outline"
                    onClick={testConnection}
                    disabled={isTestingConnection}
                    className="w-full"
                  >
                    {isTestingConnection && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
                    Test Connection
                  </Button>

                  {connectionTestResult && (
                    <div className={`p-3 rounded-lg flex items-center gap-2 ${
                      connectionTestResult.success
                        ? 'bg-green-500/10 border border-green-500/20'
                        : 'bg-destructive/10 border border-destructive/20'
                    }`}>
                      {connectionTestResult.success ? (
                        <CheckCircle className="h-4 w-4 text-green-500" />
                      ) : (
                        <AlertCircle className="h-4 w-4 text-destructive" />
                      )}
                      <span className="text-sm">
                        {connectionTestResult.message}
                      </span>
                    </div>
                  )}
                </div>
              )}
            </div>
          )}

          {currentStep === 2 && (
            <div className="space-y-4">
              <div>
                <h3 className="text-lg font-medium mb-2">Discovering Virtual Machines</h3>
                <p className="text-sm text-muted-foreground mb-4">
                  Scanning vCenter for available VMs...
                </p>

                {isDiscovering && (
                  <div className="space-y-2">
                    <Progress value={discoveryProgress} className="w-full" />
                    <p className="text-sm text-muted-foreground">Discovering VMs...</p>
                  </div>
                )}

                {!isDiscovering && discoveredVMs.length > 0 && (
                  <div className="space-y-3 max-h-60 overflow-y-auto">
                    <p className="text-sm font-medium">Found {discoveredVMs.length} VMs:</p>
                    {discoveredVMs.slice(0, 5).map((vm) => (
                      <div key={vm.id} className="flex items-center gap-3 p-2 rounded border">
                        <Server className="h-4 w-4 text-muted-foreground" />
                        <div className="flex-1">
                          <div className="flex items-center gap-2">
                            <span className="font-medium text-sm">{vm.name}</span>
                            {getStatusBadge(vm.power_state)}
                          </div>
                          <div className="text-xs text-muted-foreground space-y-0.5">
                            <div>{vm.guest_os} • {vm.num_cpu} CPU • {vm.memory_mb} MB RAM</div>
                            {vm.disks && vm.disks.length > 0 && (
                              <div>
                                💾 {vm.disks.length} disk{vm.disks.length !== 1 ? 's' : ''}
                                ({vm.disks.reduce((total: number, disk: any) => total + (disk.size_gb || 0), 0)} GB total)
                              </div>
                            )}
                            {vm.networks && vm.networks.length > 0 && (
                              <div>🌐 {vm.networks.length} network{vm.networks.length !== 1 ? 's' : ''}</div>
                            )}
                          </div>
                        </div>
                      </div>
                    ))}
                    {discoveredVMs.length > 5 && (
                      <p className="text-sm text-muted-foreground">
                        ... and {discoveredVMs.length - 5} more VMs
                      </p>
                    )}
                  </div>
                )}

                {!isDiscovering && discoveredVMs.length === 0 && (
                  <div className="text-center py-8 text-muted-foreground">
                    <Server className="h-12 w-12 mx-auto mb-4 opacity-50" />
                    <p>No VMs found in the selected vCenter.</p>
                    <p className="text-sm">Try selecting a different vCenter connection.</p>
                  </div>
                )}
              </div>
            </div>
          )}

          {currentStep === 3 && (
            <div className="space-y-4">
              <div>
                <h3 className="text-lg font-medium mb-2">Select VMs to Add</h3>
                <p className="text-sm text-muted-foreground mb-4">
                  Choose which VMs to add to management ({selectedVMIds.length} selected)
                </p>

                <div className="space-y-3 max-h-60 overflow-y-auto">
                  {discoveredVMs.map((vm) => (
                    <div key={vm.id} className="flex items-center space-x-3 p-3 rounded-lg border hover:bg-muted/50">
                      <Checkbox
                        id={`vm-${vm.id}`}
                        checked={selectedVMIds.includes(vm.id)}
                        onCheckedChange={(checked) => handleVMSelection(vm.id, checked as boolean)}
                      />
                      <div className="flex items-center gap-3 flex-1">
                        <Server className="h-4 w-4 text-muted-foreground" />
                        <div className="flex-1">
                          <div className="flex items-center gap-2">
                            <span className="font-medium">{vm.name}</span>
                            {getStatusBadge(vm.power_state)}
                          </div>
                          <div className="text-sm text-muted-foreground space-y-0.5">
                            <div>{vm.guest_os} • {vm.num_cpu} CPU • {vm.memory_mb} MB RAM</div>
                            {vm.disks && vm.disks.length > 0 && (
                              <div className="text-xs">
                                💾 {vm.disks.length} disk{vm.disks.length !== 1 ? 's' : ''}
                                ({vm.disks.reduce((total: number, disk: any) => total + (disk.size_gb || 0), 0)} GB total)
                              </div>
                            )}
                            {vm.networks && vm.networks.length > 0 && (
                              <div className="text-xs">
                                🌐 {vm.networks.length} network{vm.networks.length !== 1 ? 's' : ''}
                                {vm.networks.map((net: any, i: number) => ` ${net.network_name}`).join(',')}
                              </div>
                            )}
                          </div>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              {selectedVMIds.length > 0 && (
                <Card>
                  <CardHeader>
                    <CardTitle className="text-sm">Selected VMs ({selectedVMIds.length})</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="space-y-2">
                      {selectedVMIds.map((vmId) => {
                        const vm = discoveredVMs.find(v => v.id === vmId);
                        return vm ? (
                          <div key={vmId} className="flex items-center gap-2 text-sm">
                            <CheckCircle className="h-4 w-4 text-green-500" />
                            <span>{vm.name}</span>
                          </div>
                        ) : null;
                      })}
                    </div>
                  </CardContent>
                </Card>
              )}
            </div>
          )}
        </div>

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => {
              if (currentStep === 1) {
                handleOpenChange(false);
              } else {
                handlePrevious();
              }
            }}
          >
            {currentStep === 1 ? 'Cancel' : 'Previous'}
          </Button>

          {currentStep < 3 ? (
            <Button
              type="button"
              onClick={handleNext}
              disabled={!canProceedToNext() || isDiscovering}
            >
              {currentStep === 1 ? 'Discover VMs' : 'Next'}
            </Button>
          ) : (
            <Button
              type="button"
              onClick={addVMsToManagement}
              disabled={!canProceedToNext() || isAddingVMs}
            >
              {isAddingVMs && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
              Add to Management
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
