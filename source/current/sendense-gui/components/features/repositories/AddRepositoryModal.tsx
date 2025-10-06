"use client";

import { useState } from "react";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { Badge } from "@/components/ui/badge";
import {
  Database,
  HardDrive,
  Cloud,
  Server,
  FolderOpen,
  CheckCircle,
  XCircle,
  Loader2
} from "lucide-react";
import { Repository, RepositoryCapacity } from "./RepositoryCard";

interface RepositoryType {
  id: 'local' | 's3' | 'nfs' | 'cifs' | 'azure';
  name: string;
  description: string;
  icon: React.ReactNode;
  fields: string[];
}

const repositoryTypes: RepositoryType[] = [
  {
    id: 'local',
    name: 'Local Storage',
    description: 'Direct attached storage or local filesystem path',
    icon: <HardDrive className="h-5 w-5" />,
    fields: ['path', 'description']
  },
  {
    id: 's3',
    name: 'Amazon S3',
    description: 'Amazon S3 bucket for cloud storage',
    icon: <Cloud className="h-5 w-5" />,
    fields: ['bucket', 'region', 'accessKey', 'secretKey', 'endpoint']
  },
  {
    id: 'nfs',
    name: 'NFS Share',
    description: 'Network File System mount point',
    icon: <Server className="h-5 w-5" />,
    fields: ['host', 'path', 'mountOptions']
  },
  {
    id: 'cifs',
    name: 'CIFS/SMB Share',
    description: 'Windows file share (SMB/CIFS)',
    icon: <FolderOpen className="h-5 w-5" />,
    fields: ['host', 'share', 'username', 'password', 'domain']
  },
  {
    id: 'azure',
    name: 'Azure Blob Storage',
    description: 'Microsoft Azure Blob Storage',
    icon: <Cloud className="h-5 w-5" />,
    fields: ['accountName', 'container', 'accountKey']
  }
];

interface AddRepositoryModalProps {
  isOpen: boolean;
  onClose: () => void;
  onCreate: (repository: Omit<Repository, 'id' | 'status' | 'lastTested'>) => Promise<void>;
  editingRepository?: Repository | null;
}

export function AddRepositoryModal({ isOpen, onClose, onCreate, editingRepository }: AddRepositoryModalProps) {
  const [currentStep, setCurrentStep] = useState(1);
  const [selectedType, setSelectedType] = useState<RepositoryType | null>(null);
  const [isTesting, setIsTesting] = useState(false);
  const [testResult, setTestResult] = useState<'success' | 'error' | null>(null);
  const [isCreating, setIsCreating] = useState(false);

  const [formData, setFormData] = useState({
    name: '',
    description: '',
    // Local
    path: '',
    // S3
    bucket: '',
    region: 'us-east-1',
    accessKey: '',
    secretKey: '',
    endpoint: '',
    // NFS
    host: '',
    mountOptions: '',
    // CIFS
    share: '',
    username: '',
    password: '',
    domain: '',
    // Azure
    accountName: '',
    container: '',
    accountKey: ''
  });

  const resetForm = () => {
    setCurrentStep(1);
    setSelectedType(null);
    setTestResult(null);
    setIsTesting(false);
    setFormData({
      name: '',
      description: '',
      path: '',
      bucket: '',
      region: 'us-east-1',
      accessKey: '',
      secretKey: '',
      endpoint: '',
      host: '',
      mountOptions: '',
      share: '',
      username: '',
      password: '',
      domain: '',
      accountName: '',
      container: '',
      accountKey: ''
    });
  };

  const handleClose = () => {
    resetForm();
    onClose();
  };

  const handleTypeSelect = (type: RepositoryType) => {
    setSelectedType(type);
    setCurrentStep(2);
  };

  const handleInputChange = (field: string, value: string) => {
    setFormData(prev => ({ ...prev, [field]: value }));
  };

  const handleTestConnection = async () => {
    if (!selectedType) return;

    setIsTesting(true);
    setTestResult(null);

    try {
      // Build config object based on selected type and form data
      const buildTestConfig = () => {
        const config: any = {};

        switch (selectedType.id) {
          case 'local':
            config.path = formData.path;
            break;
          case 'nfs':
            config.server = formData.host;
            config.export_path = formData.path;
            if (formData.mountOptions) {
              config.mount_options = formData.mountOptions.split(',').map(o => o.trim());
            }
            break;
          case 'cifs':
            config.server = formData.host;
            config.share_name = formData.share;
            config.username = formData.username;
            config.password = formData.password;
            if (formData.domain) {
              config.domain = formData.domain;
            }
            break;
          case 's3':
            config.bucket = formData.bucket;
            config.region = formData.region;
            config.access_key = formData.accessKey;
            config.secret_key = formData.secretKey;
            if (formData.endpoint) {
              config.endpoint = formData.endpoint;
            }
            break;
          case 'azure':
            config.account_name = formData.accountName;
            config.container = formData.container;
            config.account_key = formData.accountKey;
            break;
        }

        return config;
      };

      const requestBody = {
        type: selectedType.id,
        config: buildTestConfig()
      };

      const response = await fetch('/api/v1/repositories/test', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(requestBody)
      });

      const data = await response.json();

      if (data.success) {
        setTestResult('success');

        // Optionally update capacity estimate if backend returns it
        if (data.storage) {
          console.log('Storage info:', {
            total: Math.round(data.storage.total_bytes / 1073741824),
            available: Math.round(data.storage.available_bytes / 1073741824),
            writable: data.storage.writable
          });
        }
      } else {
        setTestResult('error');
        console.error('Test failed:', data.error || data.message);
      }
    } catch (error) {
      console.error('Test connection error:', error);
      setTestResult('error');
    } finally {
      setIsTesting(false);
    }
  };

  const handleCreate = async () => {
    if (!selectedType || !formData.name) return;

    setIsCreating(true);
    try {
      // Build location string for parent component
      const getLocationString = () => {
        switch (selectedType.id) {
          case 'local':
            return formData.path;
          case 's3':
            return `${formData.bucket} (${formData.region})`;
          case 'nfs':
            return `${formData.host}:${formData.path}`;
          case 'cifs':
            return `\\\\${formData.host}\\${formData.share}`;
          case 'azure':
            return `${formData.accountName}/${formData.container}`;
          default:
            return '';
        }
      };

      // Pass repository data to parent's onCreate handler
      // Parent will transform this into backend format and make API call
      await onCreate({
        name: formData.name,
        type: selectedType.id,
        capacity: { total: 0, used: 0, available: 0, unit: 'GB' }, // Will be filled by backend
        description: formData.description,
        location: getLocationString()
      });

      handleClose();
    } catch (error) {
      console.error('Failed to create repository:', error);
    } finally {
      setIsCreating(false);
    }
  };

  const renderStep1 = () => (
    <div className="space-y-4">
      <div>
        <h3 className="text-lg font-semibold mb-2">Choose Repository Type</h3>
        <p className="text-muted-foreground">Select the type of storage repository you want to add.</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {repositoryTypes.map((type) => (
          <Card
            key={type.id}
            className="cursor-pointer hover:shadow-md transition-shadow"
            onClick={() => handleTypeSelect(type)}
          >
            <CardHeader className="pb-3">
              <div className="flex items-center gap-3">
                {type.icon}
                <div>
                  <CardTitle className="text-base">{type.name}</CardTitle>
                  <CardDescription>{type.description}</CardDescription>
                </div>
              </div>
            </CardHeader>
          </Card>
        ))}
      </div>
    </div>
  );

  const renderStep2 = () => {
    if (!selectedType) return null;

    return (
      <div className="space-y-6">
        <div>
          <h3 className="text-lg font-semibold mb-2">Configure Repository</h3>
          <p className="text-muted-foreground">
            Enter the details for your {selectedType.name.toLowerCase()} repository.
          </p>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-2">
            <Label htmlFor="name">Repository Name *</Label>
            <Input
              id="name"
              value={formData.name}
              onChange={(e) => handleInputChange('name', e.target.value)}
              placeholder="My Repository"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="description">Description</Label>
            <Input
              id="description"
              value={formData.description}
              onChange={(e) => handleInputChange('description', e.target.value)}
              placeholder="Optional description"
            />
          </div>
        </div>

        {/* Type-specific fields */}
        {selectedType.id === 'local' && (
          <div className="space-y-2">
            <Label htmlFor="path">Storage Path *</Label>
            <Input
              id="path"
              value={formData.path}
              onChange={(e) => handleInputChange('path', e.target.value)}
              placeholder="/mnt/storage or C:\\Storage"
            />
          </div>
        )}

        {selectedType.id === 's3' && (
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="bucket">Bucket Name *</Label>
              <Input
                id="bucket"
                value={formData.bucket}
                onChange={(e) => handleInputChange('bucket', e.target.value)}
                placeholder="my-backup-bucket"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="region">Region</Label>
              <Select value={formData.region} onValueChange={(value) => handleInputChange('region', value)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="us-east-1">US East (N. Virginia)</SelectItem>
                  <SelectItem value="us-west-2">US West (Oregon)</SelectItem>
                  <SelectItem value="eu-west-1">EU (Ireland)</SelectItem>
                  <SelectItem value="ap-southeast-1">Asia Pacific (Singapore)</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="accessKey">Access Key *</Label>
              <Input
                id="accessKey"
                type="password"
                value={formData.accessKey}
                onChange={(e) => handleInputChange('accessKey', e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="secretKey">Secret Key *</Label>
              <Input
                id="secretKey"
                type="password"
                value={formData.secretKey}
                onChange={(e) => handleInputChange('secretKey', e.target.value)}
              />
            </div>
            <div className="col-span-2 space-y-2">
              <Label htmlFor="endpoint">Custom Endpoint (optional)</Label>
              <Input
                id="endpoint"
                value={formData.endpoint}
                onChange={(e) => handleInputChange('endpoint', e.target.value)}
                placeholder="https://s3.custom.endpoint.com"
              />
            </div>
          </div>
        )}

        {selectedType.id === 'nfs' && (
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="host">NFS Server *</Label>
              <Input
                id="host"
                value={formData.host}
                onChange={(e) => handleInputChange('host', e.target.value)}
                placeholder="nfs-server.example.com"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="path">Export Path *</Label>
              <Input
                id="path"
                value={formData.path}
                onChange={(e) => handleInputChange('path', e.target.value)}
                placeholder="/export/backups"
              />
            </div>
            <div className="col-span-2 space-y-2">
              <Label htmlFor="mountOptions">Mount Options</Label>
              <Input
                id="mountOptions"
                value={formData.mountOptions}
                onChange={(e) => handleInputChange('mountOptions', e.target.value)}
                placeholder="vers=4,soft,timeo=30"
              />
            </div>
          </div>
        )}

        {selectedType.id === 'cifs' && (
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="host">Server *</Label>
              <Input
                id="host"
                value={formData.host}
                onChange={(e) => handleInputChange('host', e.target.value)}
                placeholder="fileserver.example.com"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="share">Share Name *</Label>
              <Input
                id="share"
                value={formData.share}
                onChange={(e) => handleInputChange('share', e.target.value)}
                placeholder="Backups"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="username">Username</Label>
              <Input
                id="username"
                value={formData.username}
                onChange={(e) => handleInputChange('username', e.target.value)}
                placeholder="domain\\user"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="password">Password</Label>
              <Input
                id="password"
                type="password"
                value={formData.password}
                onChange={(e) => handleInputChange('password', e.target.value)}
              />
            </div>
            <div className="col-span-2 space-y-2">
              <Label htmlFor="domain">Domain (optional)</Label>
              <Input
                id="domain"
                value={formData.domain}
                onChange={(e) => handleInputChange('domain', e.target.value)}
                placeholder="EXAMPLE"
              />
            </div>
          </div>
        )}

        {selectedType.id === 'azure' && (
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="accountName">Account Name *</Label>
              <Input
                id="accountName"
                value={formData.accountName}
                onChange={(e) => handleInputChange('accountName', e.target.value)}
                placeholder="mystorageaccount"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="container">Container Name *</Label>
              <Input
                id="container"
                value={formData.container}
                onChange={(e) => handleInputChange('container', e.target.value)}
                placeholder="backups"
              />
            </div>
            <div className="col-span-2 space-y-2">
              <Label htmlFor="accountKey">Account Key *</Label>
              <Input
                id="accountKey"
                type="password"
                value={formData.accountKey}
                onChange={(e) => handleInputChange('accountKey', e.target.value)}
              />
            </div>
          </div>
        )}
      </div>
    );
  };

  const renderStep3 = () => {
    if (!selectedType) return null;

    return (
      <div className="space-y-6">
        <div>
          <h3 className="text-lg font-semibold mb-2">Test & Create Repository</h3>
          <p className="text-muted-foreground">
            Test the repository connection before creating it.
          </p>
        </div>

        <Card>
          <CardHeader>
            <CardTitle className="text-base flex items-center gap-2">
              {selectedType.icon}
              {formData.name}
            </CardTitle>
            <CardDescription>{selectedType.name} Repository</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium">Connection Test:</span>
              <div className="flex items-center gap-2">
                {testResult === 'success' && (
                  <Badge className="bg-green-500/10 text-green-400 border-green-500/20">
                    <CheckCircle className="h-3 w-3 mr-1" />
                    Success
                  </Badge>
                )}
                {testResult === 'error' && (
                  <Badge className="bg-red-500/10 text-red-400 border-red-500/20">
                    <XCircle className="h-3 w-3 mr-1" />
                    Failed
                  </Badge>
                )}
                {!testResult && !isTesting && (
                  <Badge variant="outline">Not tested</Badge>
                )}
                {isTesting && (
                  <Badge variant="outline">
                    <Loader2 className="h-3 w-3 mr-1 animate-spin" />
                    Testing...
                  </Badge>
                )}
              </div>
            </div>

            <Button
              onClick={handleTestConnection}
              disabled={isTesting}
              variant="outline"
              className="w-full"
            >
              {isTesting ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  Testing Connection...
                </>
              ) : (
                'Test Connection'
              )}
            </Button>
          </CardContent>
        </Card>
      </div>
    );
  };

  const canProceedToNext = () => {
    switch (currentStep) {
      case 1: return selectedType !== null;
      case 2: return formData.name.trim() !== '' &&
        (selectedType?.id === 'local' ? formData.path.trim() !== '' :
         selectedType?.id === 's3' ? formData.bucket.trim() !== '' && formData.accessKey.trim() !== '' && formData.secretKey.trim() !== '' :
         selectedType?.id === 'nfs' ? formData.host.trim() !== '' && formData.path.trim() !== '' :
         selectedType?.id === 'cifs' ? formData.host.trim() !== '' && formData.share.trim() !== '' :
         selectedType?.id === 'azure' ? formData.accountName.trim() !== '' && formData.container.trim() !== '' && formData.accountKey.trim() !== '' :
         false);
      case 3: return testResult === 'success';
      default: return false;
    }
  };

  const getStepProgress = () => {
    return (currentStep / 3) * 100;
  };

  return (
    <Dialog open={isOpen} onOpenChange={handleClose}>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <DialogTitle>
            {editingRepository ? 'Edit Repository' : 'Add New Repository'}
          </DialogTitle>
          <DialogDescription>
            Configure a new storage repository for your backup system.
          </DialogDescription>
        </DialogHeader>

        {/* Progress Bar */}
        <div className="space-y-2">
          <div className="flex justify-between text-sm">
            <span>Step {currentStep} of 3</span>
            <span>{Math.round(getStepProgress())}% complete</span>
          </div>
          <Progress value={getStepProgress()} className="h-2" />
        </div>

        {/* Step Content */}
        <div className="flex-1 overflow-y-auto py-4">
          {currentStep === 1 && renderStep1()}
          {currentStep === 2 && renderStep2()}
          {currentStep === 3 && renderStep3()}
        </div>

        {/* Actions */}
        <div className="flex justify-between border-t pt-4">
          <Button
            variant="outline"
            onClick={currentStep === 1 ? handleClose : () => setCurrentStep(currentStep - 1)}
          >
            {currentStep === 1 ? 'Cancel' : 'Back'}
          </Button>

          <div className="flex gap-2">
            {currentStep === 3 && (
              <Button
                onClick={handleCreate}
                disabled={!canProceedToNext() || isCreating}
              >
                {isCreating ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                    Creating...
                  </>
                ) : (
                  'Create Repository'
                )}
              </Button>
            )}
            {currentStep < 3 && (
              <Button
                onClick={() => setCurrentStep(currentStep + 1)}
                disabled={!canProceedToNext()}
              >
                Next
              </Button>
            )}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
