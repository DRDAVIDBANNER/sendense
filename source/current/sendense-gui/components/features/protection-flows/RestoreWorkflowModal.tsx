"use client";

import { useState } from "react";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import {
  ArrowLeft,
  ArrowRight,
  CheckCircle,
  AlertTriangle,
  Server,
  HardDrive,
  Download,
  Upload,
  Settings,
  Shield,
  Crown
} from "lucide-react";

interface RestoreDestination {
  id: string;
  name: string;
  type: 'source' | 'local' | 'cross-platform';
  available: boolean;
  description: string;
}

interface LicenseFeatures {
  backup_edition: boolean;
  enterprise_edition: boolean;
  replication_edition: boolean;
}

interface RestoreConfiguration {
  restoreType: 'full-vm' | 'file-level' | 'application-aware';
  destination: string;
  method: 'direct-restore' | 'download' | 'new-location';
  networkMapping: boolean;
  resourceAllocation: {
    cpu: number;
    memory: number;
    storage: number;
  };
}

interface RestoreWorkflowModalProps {
  isOpen: boolean;
  onClose: () => void;
  onRestore: (config: RestoreConfiguration) => void;
  availableDestinations: RestoreDestination[];
  licenseFeatures: LicenseFeatures;
}

const mockDestinations: RestoreDestination[] = [
  {
    id: 'source-vm',
    name: 'Original Source VM',
    type: 'source',
    available: true,
    description: 'Restore directly to the original virtual machine'
  },
  {
    id: 'local-download',
    name: 'Local Download',
    type: 'local',
    available: true,
    description: 'Download files to local storage'
  },
  {
    id: 'cross-platform',
    name: 'Cross-Platform Restore',
    type: 'cross-platform',
    available: false,
    description: 'Restore to different hypervisor platform (Enterprise only)'
  }
];

export function RestoreWorkflowModal({
  isOpen,
  onClose,
  onRestore,
  availableDestinations = mockDestinations,
  licenseFeatures
}: RestoreWorkflowModalProps) {
  const [currentStep, setCurrentStep] = useState(1);
  const [config, setConfig] = useState<RestoreConfiguration>({
    restoreType: 'full-vm',
    destination: '',
    method: 'direct-restore',
    networkMapping: false,
    resourceAllocation: {
      cpu: 2,
      memory: 4,
      storage: 100
    }
  });

  const totalSteps = 5;

  const handleNext = () => {
    if (currentStep < totalSteps) {
      setCurrentStep(currentStep + 1);
    }
  };

  const handlePrevious = () => {
    if (currentStep > 1) {
      setCurrentStep(currentStep - 1);
    }
  };

  const handleFinish = () => {
    onRestore(config);
    onClose();
    // Reset form
    setCurrentStep(1);
    setConfig({
      restoreType: 'full-vm',
      destination: '',
      method: 'direct-restore',
      networkMapping: false,
      resourceAllocation: {
        cpu: 2,
        memory: 4,
        storage: 100
      }
    });
  };

  const getStepTitle = (step: number) => {
    switch (step) {
      case 1: return 'Restore Type';
      case 2: return 'Destination';
      case 3: return 'Restore Method';
      case 4: return 'Configuration';
      case 5: return 'Confirmation';
      default: return '';
    }
  };

  const canProceedToNext = () => {
    switch (currentStep) {
      case 1: return !!config.restoreType;
      case 2: return !!config.destination;
      case 3: return !!config.method;
      case 4: return true; // Configuration is optional
      case 5: return true; // Always allow confirmation
      default: return false;
    }
  };

  const getLicenseBadge = (feature: keyof LicenseFeatures) => {
    if (licenseFeatures[feature]) {
      return <Badge className="bg-green-500/10 text-green-400 border-green-500/20">Available</Badge>;
    }
    return <Badge className="bg-yellow-500/10 text-yellow-400 border-yellow-500/20">Upgrade Required</Badge>;
  };

  const getDestinationOptions = () => {
    return availableDestinations.filter(dest => {
      if (dest.type === 'cross-platform' && !licenseFeatures.enterprise_edition) {
        return false;
      }
      return dest.available;
    });
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Download className="h-5 w-5" />
            Restore Workflow
          </DialogTitle>
          <DialogDescription>
            Configure and execute data restoration from backup
          </DialogDescription>
        </DialogHeader>

        {/* Progress Indicator */}
        <div className="mb-6">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm font-medium">Step {currentStep} of {totalSteps}</span>
            <span className="text-sm text-muted-foreground">{getStepTitle(currentStep)}</span>
          </div>
          <Progress value={(currentStep / totalSteps) * 100} className="h-2" />
        </div>

        {/* License Status */}
        <div className="mb-4 p-3 bg-muted/50 rounded-lg">
          <div className="flex items-center gap-2 mb-2">
            <Crown className="h-4 w-4 text-yellow-500" />
            <span className="font-medium text-sm">License Features</span>
          </div>
          <div className="flex gap-4 text-xs">
            <div className="flex items-center gap-1">
              <span>Backup Edition:</span>
              {getLicenseBadge('backup_edition')}
            </div>
            <div className="flex items-center gap-1">
              <span>Enterprise Edition:</span>
              {getLicenseBadge('enterprise_edition')}
            </div>
            <div className="flex items-center gap-1">
              <span>Replication Edition:</span>
              {getLicenseBadge('replication_edition')}
            </div>
          </div>
        </div>

        {/* Step Content */}
        <div className="flex-1 overflow-auto py-4">
          {currentStep === 1 && (
            <div className="space-y-4">
              <div>
                <h3 className="text-lg font-semibold mb-2">What would you like to restore?</h3>
                <p className="text-sm text-muted-foreground mb-4">
                  Choose the type of restore operation you need to perform.
                </p>
              </div>

              <div className="space-y-3">
                <div className="space-y-2">
                  <Label htmlFor="restore-type">Restore Type</Label>
                  <Select
                    value={config.restoreType}
                    onValueChange={(value) => setConfig(prev => ({ ...prev, restoreType: value as any }))}
                  >
                    <SelectTrigger id="restore-type">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="full-vm">Full VM Restore</SelectItem>
                      <SelectItem value="file-level">File-Level Restore</SelectItem>
                      <SelectItem value="application-aware">Application-Aware Restore</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div className="p-4 border border-border rounded-lg bg-muted/20">
                  {config.restoreType === 'full-vm' && (
                    <div className="flex items-start gap-3">
                      <Server className="h-5 w-5 text-muted-foreground mt-0.5" />
                      <div>
                        <h4 className="font-medium">Full VM Restore</h4>
                        <p className="text-sm text-muted-foreground">
                          Restore the entire virtual machine including all disks, configuration, and settings.
                        </p>
                      </div>
                    </div>
                  )}
                  {config.restoreType === 'file-level' && (
                    <div className="flex items-start gap-3">
                      <HardDrive className="h-5 w-5 text-muted-foreground mt-0.5" />
                      <div>
                        <h4 className="font-medium">File-Level Restore</h4>
                        <p className="text-sm text-muted-foreground">
                          Restore individual files and folders from the backup.
                        </p>
                      </div>
                    </div>
                  )}
                  {config.restoreType === 'application-aware' && (
                    <div className="flex items-start gap-3">
                      <Shield className="h-5 w-5 text-muted-foreground mt-0.5" />
                      <div>
                        <h4 className="font-medium">Application-Aware Restore</h4>
                        <p className="text-sm text-muted-foreground">
                          Restore with application consistency and transaction logs.
                        </p>
                      </div>
                    </div>
                  )}
                </div>
              </div>
            </div>
          )}

          {currentStep === 2 && (
            <div className="space-y-4">
              <div>
                <h3 className="text-lg font-semibold mb-2">Where should the data be restored?</h3>
                <p className="text-sm text-muted-foreground mb-4">
                  Select the destination for your restored data.
                </p>
              </div>

              <div className="space-y-3">
                {getDestinationOptions().map((dest) => (
                  <div
                    key={dest.id}
                    className={`p-4 border border-border rounded-lg cursor-pointer transition-colors ${
                      config.destination === dest.id ? 'border-primary bg-primary/5' : 'hover:bg-muted/50'
                    }`}
                    onClick={() => setConfig(prev => ({ ...prev, destination: dest.id }))}
                  >
                    <div className="flex items-start justify-between">
                      <div className="flex-1">
                        <div className="flex items-center gap-2 mb-1">
                          <h4 className="font-medium">{dest.name}</h4>
                          {dest.type === 'cross-platform' && (
                            <Badge className="bg-purple-500/10 text-purple-400 border-purple-500/20">
                              Enterprise
                            </Badge>
                          )}
                        </div>
                        <p className="text-sm text-muted-foreground">{dest.description}</p>
                      </div>
                      <div className="ml-4">
                        {config.destination === dest.id ? (
                          <CheckCircle className="h-5 w-5 text-primary" />
                        ) : (
                          <div className="w-5 h-5 border-2 border-muted-foreground rounded-full" />
                        )}
                      </div>
                    </div>
                  </div>
                ))}
              </div>

              {!licenseFeatures.enterprise_edition && availableDestinations.some(d => d.type === 'cross-platform') && (
                <div className="p-3 border border-yellow-500/20 bg-yellow-500/10 rounded-lg">
                  <div className="flex items-center gap-2">
                    <AlertTriangle className="h-4 w-4 text-yellow-500" />
                    <span className="text-sm font-medium text-yellow-700">Enterprise Feature Required</span>
                  </div>
                  <p className="text-sm text-yellow-600 mt-1">
                    Cross-platform restore requires Enterprise Edition. Upgrade your license to access this feature.
                  </p>
                </div>
              )}
            </div>
          )}

          {currentStep === 3 && (
            <div className="space-y-4">
              <div>
                <h3 className="text-lg font-semibold mb-2">How should the restore be performed?</h3>
                <p className="text-sm text-muted-foreground mb-4">
                  Choose the method for executing the restore operation.
                </p>
              </div>

              <div className="space-y-3">
                <div className="space-y-2">
                  <Label htmlFor="restore-method">Restore Method</Label>
                  <Select
                    value={config.method}
                    onValueChange={(value) => setConfig(prev => ({ ...prev, method: value as any }))}
                  >
                    <SelectTrigger id="restore-method">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="direct-restore">Direct Restore to Source</SelectItem>
                      <SelectItem value="download">Download to Local Storage</SelectItem>
                      <SelectItem value="new-location">Restore to New Location</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div className="p-4 border border-border rounded-lg bg-muted/20">
                  {config.method === 'direct-restore' && (
                    <div>
                      <h4 className="font-medium">Direct Restore to Source</h4>
                      <p className="text-sm text-muted-foreground mt-1">
                        Restore data directly to the original location, overwriting existing data.
                      </p>
                    </div>
                  )}
                  {config.method === 'download' && (
                    <div>
                      <h4 className="font-medium">Download to Local Storage</h4>
                      <p className="text-sm text-muted-foreground mt-1">
                        Download files to local storage for manual restoration.
                      </p>
                    </div>
                  )}
                  {config.method === 'new-location' && (
                    <div>
                      <h4 className="font-medium">Restore to New Location</h4>
                      <p className="text-sm text-muted-foreground mt-1">
                        Restore to a different location or create a new virtual machine.
                      </p>
                    </div>
                  )}
                </div>
              </div>
            </div>
          )}

          {currentStep === 4 && (
            <div className="space-y-4">
              <div>
                <h3 className="text-lg font-semibold mb-2">Advanced Configuration</h3>
                <p className="text-sm text-muted-foreground mb-4">
                  Configure network settings and resource allocation for the restore operation.
                </p>
              </div>

              <div className="space-y-4">
                <Card>
                  <CardHeader>
                    <CardTitle className="text-base">Network Configuration</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="flex items-center space-x-2">
                      <input
                        type="checkbox"
                        id="network-mapping"
                        checked={config.networkMapping}
                        onChange={(e) => setConfig(prev => ({ ...prev, networkMapping: e.target.checked }))}
                        className="rounded"
                      />
                      <Label htmlFor="network-mapping" className="text-sm">
                        Preserve network configuration and IP addresses
                      </Label>
                    </div>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader>
                    <CardTitle className="text-base">Resource Allocation</CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <div className="grid grid-cols-3 gap-4">
                      <div>
                        <Label htmlFor="cpu" className="text-sm">CPU Cores</Label>
                        <Select
                          value={config.resourceAllocation.cpu.toString()}
                          onValueChange={(value) => setConfig(prev => ({
                            ...prev,
                            resourceAllocation: { ...prev.resourceAllocation, cpu: parseInt(value) }
                          }))}
                        >
                          <SelectTrigger>
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            {[1, 2, 4, 6, 8, 12, 16].map(cpu => (
                              <SelectItem key={cpu} value={cpu.toString()}>{cpu}</SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </div>

                      <div>
                        <Label htmlFor="memory" className="text-sm">Memory (GB)</Label>
                        <Select
                          value={config.resourceAllocation.memory.toString()}
                          onValueChange={(value) => setConfig(prev => ({
                            ...prev,
                            resourceAllocation: { ...prev.resourceAllocation, memory: parseInt(value) }
                          }))}
                        >
                          <SelectTrigger>
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            {[1, 2, 4, 6, 8, 12, 16, 24, 32, 48, 64].map(mem => (
                              <SelectItem key={mem} value={mem.toString()}>{mem}</SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </div>

                      <div>
                        <Label htmlFor="storage" className="text-sm">Storage (GB)</Label>
                        <Select
                          value={config.resourceAllocation.storage.toString()}
                          onValueChange={(value) => setConfig(prev => ({
                            ...prev,
                            resourceAllocation: { ...prev.resourceAllocation, storage: parseInt(value) }
                          }))}
                        >
                          <SelectTrigger>
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            {[50, 100, 200, 500, 1000, 2000].map(storage => (
                              <SelectItem key={storage} value={storage.toString()}>{storage}</SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              </div>
            </div>
          )}

          {currentStep === 5 && (
            <div className="space-y-4">
              <div>
                <h3 className="text-lg font-semibold mb-2">Review and Confirm</h3>
                <p className="text-sm text-muted-foreground mb-4">
                  Please review your restore configuration before proceeding.
                </p>
              </div>

              <Card>
                <CardHeader>
                  <CardTitle className="text-base">Restore Summary</CardTitle>
                </CardHeader>
                <CardContent className="space-y-3">
                  <div className="grid grid-cols-2 gap-4 text-sm">
                    <div>
                      <span className="text-muted-foreground">Restore Type:</span>
                      <div className="font-medium capitalize">{config.restoreType.replace('-', ' ')}</div>
                    </div>
                    <div>
                      <span className="text-muted-foreground">Destination:</span>
                      <div className="font-medium">
                        {availableDestinations.find(d => d.id === config.destination)?.name || 'Unknown'}
                      </div>
                    </div>
                    <div>
                      <span className="text-muted-foreground">Method:</span>
                      <div className="font-medium capitalize">{config.method.replace('-', ' ')}</div>
                    </div>
                    <div>
                      <span className="text-muted-foreground">Network Mapping:</span>
                      <div className="font-medium">{config.networkMapping ? 'Enabled' : 'Disabled'}</div>
                    </div>
                  </div>

                  <div className="border-t pt-3 mt-3">
                    <span className="text-muted-foreground text-sm">Resource Allocation:</span>
                    <div className="mt-1 text-sm">
                      {config.resourceAllocation.cpu} CPU cores • {config.resourceAllocation.memory} GB RAM • {config.resourceAllocation.storage} GB Storage
                    </div>
                  </div>
                </CardContent>
              </Card>

              <div className="p-3 border border-orange-500/20 bg-orange-500/10 rounded-lg">
                <div className="flex items-center gap-2">
                  <AlertTriangle className="h-4 w-4 text-orange-500" />
                  <span className="text-sm font-medium text-orange-700">Important Warning</span>
                </div>
                <p className="text-sm text-orange-600 mt-1">
                  This operation will {config.method === 'direct-restore' ? 'overwrite existing data' : 'create new files'}.
                  Please ensure you have adequate backups before proceeding.
                </p>
              </div>
            </div>
          )}
        </div>

        {/* Footer */}
        <DialogFooter className="border-t pt-4">
          <div className="flex items-center justify-between w-full">
            <div className="text-sm text-muted-foreground">
              Step {currentStep} of {totalSteps}
            </div>

            <div className="flex gap-2">
              <Button
                variant="outline"
                onClick={currentStep === 1 ? onClose : handlePrevious}
              >
                {currentStep === 1 ? 'Cancel' : <ArrowLeft className="h-4 w-4" />}
                {currentStep === 1 ? 'Cancel' : 'Previous'}
              </Button>

              {currentStep < totalSteps ? (
                <Button
                  onClick={handleNext}
                  disabled={!canProceedToNext()}
                >
                  Next
                  <ArrowRight className="h-4 w-4 ml-2" />
                </Button>
              ) : (
                <Button
                  onClick={handleFinish}
                  className="bg-red-600 hover:bg-red-700"
                >
                  <Download className="h-4 w-4 mr-2" />
                  Start Restore
                </Button>
              )}
            </div>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
