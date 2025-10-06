"use client";

import { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Progress } from "@/components/ui/progress";
import { PageHeader } from "@/components/common/PageHeader";
import {
  Plus,
  HardDrive,
  CheckCircle,
  XCircle,
  AlertTriangle,
  Database,
  TestTube,
  MoreHorizontal,
  Edit,
  Trash2,
  Cloud,
  Server
} from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

interface Destination {
  id: string;
  name: string;
  type: 'nfs' | 'cifs' | 'local' | 's3' | 'azure' | 'gcs';
  host?: string;
  port?: number;
  path: string;
  username?: string;
  status: 'connected' | 'disconnected' | 'error';
  lastConnected: string;
  totalCapacity: number; // GB
  usedCapacity: number; // GB
  activeJobs: number;
  totalJobs: number;
}

const mockDestinations: Destination[] = [
  {
    id: '1',
    name: 'Production NFS Storage',
    type: 'nfs',
    host: 'nfs-prod.company.com',
    port: 2049,
    path: '/backup/production',
    status: 'connected',
    lastConnected: '2025-10-06T08:00:00Z',
    totalCapacity: 2000,
    usedCapacity: 1200,
    activeJobs: 3,
    totalJobs: 156
  },
  {
    id: '2',
    name: 'Development CIFS Share',
    type: 'cifs',
    host: 'fileserver.company.com',
    port: 445,
    path: '\\\\fileserver\\backup\\dev',
    username: 'backup_user',
    status: 'connected',
    lastConnected: '2025-10-06T07:30:00Z',
    totalCapacity: 500,
    usedCapacity: 280,
    activeJobs: 1,
    totalJobs: 89
  },
  {
    id: '3',
    name: 'Local Storage',
    type: 'local',
    path: '/mnt/backup/local',
    status: 'connected',
    lastConnected: '2025-10-06T06:00:00Z',
    totalCapacity: 1000,
    usedCapacity: 650,
    activeJobs: 0,
    totalJobs: 234
  },
  {
    id: '4',
    name: 'Cloud Archive S3',
    type: 's3',
    host: 's3.amazonaws.com',
    port: 443,
    path: 's3://sendense-archive',
    username: 'AKIAIOSFODNN7EXAMPLE',
    status: 'error',
    lastConnected: '2025-10-05T15:00:00Z',
    totalCapacity: 5000,
    usedCapacity: 3200,
    activeJobs: 0,
    totalJobs: 45
  }
];

export default function DestinationsPage() {
  const [destinations, setDestinations] = useState<Destination[]>(mockDestinations);
  const [isAddModalOpen, setIsAddModalOpen] = useState(false);
  const [editingDestination, setEditingDestination] = useState<Destination | null>(null);

  const [formData, setFormData] = useState({
    name: '',
    type: 'nfs' as Destination['type'],
    host: '',
    port: '2049',
    path: '',
    username: '',
    password: '',
    accessKey: '',
    secretKey: ''
  });

  const handleInputChange = (field: string, value: string) => {
    setFormData(prev => ({ ...prev, [field]: value }));
  };

  const handleTestConnection = async (destinationId: string) => {
    // Simulate connection test
    setDestinations(prev => prev.map(dest =>
      dest.id === destinationId
        ? { ...dest, status: 'connected' as const, lastConnected: new Date().toISOString() }
        : dest
    ));
  };

  const handleAddDestination = () => {
    const newDestination: Destination = {
      id: Date.now().toString(),
      name: formData.name,
      type: formData.type,
      host: formData.host || undefined,
      port: formData.port ? parseInt(formData.port) : undefined,
      path: formData.path,
      username: formData.username || undefined,
      status: 'disconnected',
      lastConnected: '',
      totalCapacity: 1000,
      usedCapacity: 0,
      activeJobs: 0,
      totalJobs: 0
    };

    setDestinations(prev => [...prev, newDestination]);
    setFormData({
      name: '',
      type: 'nfs',
      host: '',
      port: '2049',
      path: '',
      username: '',
      password: '',
      accessKey: '',
      secretKey: ''
    });
    setIsAddModalOpen(false);
  };

  const handleEditDestination = (destination: Destination) => {
    setEditingDestination(destination);
    setFormData({
      name: destination.name,
      type: destination.type,
      host: destination.host || '',
      port: destination.port?.toString() || '2049',
      path: destination.path,
      username: destination.username || '',
      password: '',
      accessKey: '',
      secretKey: ''
    });
  };

  const handleUpdateDestination = () => {
    if (!editingDestination) return;

    setDestinations(prev => prev.map(dest =>
      dest.id === editingDestination.id
        ? {
            ...dest,
            name: formData.name,
            type: formData.type,
            host: formData.host || undefined,
            port: formData.port ? parseInt(formData.port) : undefined,
            path: formData.path,
            username: formData.username || undefined
          }
        : dest
    ));

    setEditingDestination(null);
    setFormData({
      name: '',
      type: 'nfs',
      host: '',
      port: '2049',
      path: '',
      username: '',
      password: '',
      accessKey: '',
      secretKey: ''
    });
  };

  const handleDeleteDestination = (destinationId: string) => {
    setDestinations(prev => prev.filter(dest => dest.id !== destinationId));
  };

  const getTypeIcon = (type: string) => {
    switch (type) {
      case 'nfs':
        return <Database className="h-5 w-5 text-blue-500" />;
      case 'cifs':
        return <Server className="h-5 w-5 text-green-500" />;
      case 'local':
        return <HardDrive className="h-5 w-5 text-purple-500" />;
      case 's3':
      case 'azure':
      case 'gcs':
        return <Cloud className="h-5 w-5 text-orange-500" />;
      default:
        return <HardDrive className="h-5 w-5 text-gray-500" />;
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'connected':
        return <CheckCircle className="h-5 w-5 text-green-500" />;
      case 'disconnected':
        return <XCircle className="h-5 w-5 text-gray-500" />;
      case 'error':
        return <AlertTriangle className="h-5 w-5 text-red-500" />;
      default:
        return <XCircle className="h-5 w-5 text-gray-500" />;
    }
  };

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'connected':
        return <Badge className="bg-green-500/10 text-green-400 border-green-500/20">Connected</Badge>;
      case 'disconnected':
        return <Badge variant="secondary">Disconnected</Badge>;
      case 'error':
        return <Badge className="bg-red-500/10 text-red-400 border-red-500/20">Error</Badge>;
      default:
        return <Badge variant="secondary">Unknown</Badge>;
    }
  };

  const formatCapacity = (used: number, total: number) => {
    const percentage = (used / total) * 100;
    return {
      used: `${(used / 1000).toFixed(1)}TB`,
      total: `${(total / 1000).toFixed(1)}TB`,
      percentage: percentage.toFixed(1)
    };
  };

  const formatLastConnected = (timestamp: string) => {
    if (!timestamp) return 'Never';

    const date = new Date(timestamp);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffHours = diffMs / (1000 * 60 * 60);

    if (diffHours < 1) {
      return 'Recently';
    } else if (diffHours < 24) {
      return `${Math.floor(diffHours)}h ago`;
    } else {
      return date.toLocaleDateString();
    }
  };

  const getTypeDisplayName = (type: string) => {
    switch (type) {
      case 'nfs':
        return 'NFS';
      case 'cifs':
        return 'CIFS/SMB';
      case 'local':
        return 'Local';
      case 's3':
        return 'Amazon S3';
      case 'azure':
        return 'Azure Blob';
      case 'gcs':
        return 'Google Cloud';
      default:
        return type.toUpperCase();
    }
  };

  const getCapacityColor = (percentage: number) => {
    if (percentage >= 90) return 'bg-red-500';
    if (percentage >= 75) return 'bg-yellow-500';
    return 'bg-green-500';
  };

  return (
    <div className="h-full flex flex-col">
      <PageHeader
        title="Destinations"
        breadcrumbs={[
          { label: "Dashboard", href: "/dashboard" },
          { label: "Settings", href: "/settings/destinations" },
          { label: "Destinations" }
        ]}
        actions={
          <Button onClick={() => setIsAddModalOpen(true)} className="gap-2">
            <Plus className="h-4 w-4" />
            Add Destination
          </Button>
        }
      />

      <div className="flex-1 overflow-auto">
        <div className="p-6">
          <div className="mb-6">
            <h2 className="text-lg font-semibold text-foreground mb-2">
              Backup Destinations
            </h2>
            <p className="text-muted-foreground">
              Manage storage destinations for backup operations and replication.
            </p>
          </div>

          {/* Summary Cards */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Total Destinations</CardTitle>
                <HardDrive className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{destinations.length}</div>
                <p className="text-xs text-muted-foreground">
                  Storage locations configured
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Connected</CardTitle>
                <CheckCircle className="h-4 w-4 text-green-500" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold text-green-600">
                  {destinations.filter(d => d.status === 'connected').length}
                </div>
                <p className="text-xs text-muted-foreground">
                  Active destinations
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Total Capacity</CardTitle>
                <Database className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">
                  {(destinations.reduce((sum, dest) => sum + dest.totalCapacity, 0) / 1000).toFixed(1)}TB
                </div>
                <p className="text-xs text-muted-foreground">
                  Total storage available
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Active Jobs</CardTitle>
                <Cloud className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">
                  {destinations.reduce((sum, dest) => sum + dest.activeJobs, 0)}
                </div>
                <p className="text-xs text-muted-foreground">
                  Currently running backups
                </p>
              </CardContent>
            </Card>
          </div>

          {/* Destinations Grid */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {destinations.map((destination) => {
              const capacity = formatCapacity(destination.usedCapacity, destination.totalCapacity);
              const capacityPercentage = parseFloat(capacity.percentage);

              return (
                <Card key={destination.id} className="relative">
                  <CardHeader className="pb-3">
                    <div className="flex items-start justify-between">
                      <div className="flex items-center gap-3">
                        {getTypeIcon(destination.type)}
                        <div className="flex items-center gap-2">
                          {getStatusIcon(destination.status)}
                          <div>
                            <CardTitle className="text-lg">{destination.name}</CardTitle>
                            <p className="text-sm text-muted-foreground">
                              {getTypeDisplayName(destination.type)} • {destination.host || 'Local'}{destination.port ? `:${destination.port}` : ''}
                            </p>
                          </div>
                        </div>
                      </div>
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button
                            variant="ghost"
                            size="sm"
                            className="h-8 w-8 p-0"
                          >
                            <MoreHorizontal className="h-4 w-4" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem onClick={() => handleTestConnection(destination.id)}>
                            <TestTube className="h-4 w-4 mr-2" />
                            Test Connection
                          </DropdownMenuItem>
                          <DropdownMenuItem onClick={() => handleEditDestination(destination)}>
                            <Edit className="h-4 w-4 mr-2" />
                            Edit Destination
                          </DropdownMenuItem>
                          <DropdownMenuSeparator />
                          <DropdownMenuItem
                            onClick={() => handleDeleteDestination(destination.id)}
                            className="text-destructive focus:text-destructive"
                          >
                            <Trash2 className="h-4 w-4 mr-2" />
                            Delete Destination
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </div>
                  </CardHeader>

                  <CardContent className="space-y-4">
                    <div className="flex items-center justify-between">
                      <span className="text-sm text-muted-foreground">Status</span>
                      {getStatusBadge(destination.status)}
                    </div>

                    <div>
                      <div className="flex justify-between text-sm mb-2">
                        <span className="text-muted-foreground">Capacity Used</span>
                        <span className="font-medium">
                          {capacity.used} / {capacity.total} ({capacity.percentage}%)
                        </span>
                      </div>
                      <Progress
                        value={capacityPercentage}
                        className="h-2"
                        // Note: Progress component styling would need custom CSS for color changes
                      />
                    </div>

                    <div className="grid grid-cols-2 gap-4 text-sm">
                      <div>
                        <span className="text-muted-foreground">Active Jobs:</span>
                        <span className="ml-2 font-medium">{destination.activeJobs}</span>
                      </div>
                      <div>
                        <span className="text-muted-foreground">Total Jobs:</span>
                        <span className="ml-2 font-medium">{destination.totalJobs}</span>
                      </div>
                    </div>

                    <div>
                      <span className="text-sm text-muted-foreground">Path:</span>
                      <span className="ml-2 text-sm font-mono break-all">{destination.path}</span>
                    </div>

                    <div>
                      <span className="text-sm text-muted-foreground">Last Connected:</span>
                      <span className="ml-2 text-sm font-medium">{formatLastConnected(destination.lastConnected)}</span>
                    </div>

                    <div className="pt-2">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleTestConnection(destination.id)}
                        className="w-full gap-2"
                      >
                        <TestTube className="h-4 w-4" />
                        Test Connection
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              );
            })}

            {/* Add New Destination Card */}
            <Card
              className="border-2 border-dashed border-muted-foreground/20 hover:border-primary/50 cursor-pointer transition-colors"
              onClick={() => setIsAddModalOpen(true)}
            >
              <CardContent className="flex flex-col items-center justify-center py-12">
                <div className="w-12 h-12 rounded-full bg-muted flex items-center justify-center mb-4">
                  <Plus className="h-6 w-6 text-muted-foreground" />
                </div>
                <h3 className="text-lg font-medium text-foreground mb-2">Add New Destination</h3>
                <p className="text-sm text-muted-foreground text-center">
                  Configure a storage destination for backups
                </p>
              </CardContent>
            </Card>
          </div>
        </div>
      </div>

      {/* Add Destination Modal */}
      <Dialog open={isAddModalOpen} onOpenChange={setIsAddModalOpen}>
        <DialogContent className="sm:max-w-[600px]">
          <DialogHeader>
            <DialogTitle>Add Storage Destination</DialogTitle>
            <DialogDescription>
              Configure a new destination for backup operations.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="dest-name">Destination Name</Label>
              <Input
                id="dest-name"
                placeholder="e.g., Production NFS Storage"
                value={formData.name}
                onChange={(e) => handleInputChange('name', e.target.value)}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="dest-type">Storage Type</Label>
              <Select value={formData.type} onValueChange={(value) => handleInputChange('type', value)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="nfs">NFS</SelectItem>
                  <SelectItem value="cifs">CIFS/SMB</SelectItem>
                  <SelectItem value="local">Local Storage</SelectItem>
                  <SelectItem value="s3">Amazon S3</SelectItem>
                  <SelectItem value="azure">Azure Blob Storage</SelectItem>
                  <SelectItem value="gcs">Google Cloud Storage</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {(formData.type === 'nfs' || formData.type === 'cifs' || formData.type === 's3' || formData.type === 'azure' || formData.type === 'gcs') && (
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="dest-host">Host</Label>
                  <Input
                    id="dest-host"
                    placeholder={formData.type === 's3' ? 's3.amazonaws.com' : formData.type === 'nfs' ? 'nfs.company.com' : 'hostname'}
                    value={formData.host}
                    onChange={(e) => handleInputChange('host', e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="dest-port">Port</Label>
                  <Input
                    id="dest-port"
                    type="number"
                    placeholder={formData.type === 'nfs' ? '2049' : formData.type === 'cifs' ? '445' : '443'}
                    value={formData.port}
                    onChange={(e) => handleInputChange('port', e.target.value)}
                  />
                </div>
              </div>
            )}

            <div className="space-y-2">
              <Label htmlFor="dest-path">Path</Label>
              <Input
                id="dest-path"
                placeholder={
                  formData.type === 'nfs' ? '/backup/production' :
                  formData.type === 'cifs' ? '\\\\server\\share' :
                  formData.type === 'local' ? '/mnt/backup' :
                  formData.type === 's3' ? 's3://bucket-name/path' :
                  formData.type === 'azure' ? 'https://account.blob.core.windows.net/container' :
                  'gs://bucket-name/path'
                }
                value={formData.path}
                onChange={(e) => handleInputChange('path', e.target.value)}
              />
            </div>

            {(formData.type === 'cifs' || formData.type === 'nfs') && (
              <>
                <div className="space-y-2">
                  <Label htmlFor="dest-username">Username</Label>
                  <Input
                    id="dest-username"
                    placeholder="backup_user"
                    value={formData.username}
                    onChange={(e) => handleInputChange('username', e.target.value)}
                  />
                </div>

                <div className="space-y-2">
                  <Label htmlFor="dest-password">Password</Label>
                  <Input
                    id="dest-password"
                    type="password"
                    placeholder="Enter password"
                    value={formData.password}
                    onChange={(e) => handleInputChange('password', e.target.value)}
                  />
                </div>
              </>
            )}

            {(formData.type === 's3' || formData.type === 'azure' || formData.type === 'gcs') && (
              <>
                <div className="space-y-2">
                  <Label htmlFor="dest-access-key">
                    {formData.type === 's3' ? 'Access Key' : formData.type === 'azure' ? 'Account Name' : 'Service Account'}
                  </Label>
                  <Input
                    id="dest-access-key"
                    placeholder={formData.type === 's3' ? 'AKIAIOSFODNN7EXAMPLE' : 'account-name'}
                    value={formData.accessKey}
                    onChange={(e) => handleInputChange('accessKey', e.target.value)}
                  />
                </div>

                <div className="space-y-2">
                  <Label htmlFor="dest-secret-key">
                    {formData.type === 's3' ? 'Secret Key' : formData.type === 'azure' ? 'Account Key' : 'Private Key'}
                  </Label>
                  <Input
                    id="dest-secret-key"
                    type="password"
                    placeholder="Enter secret key"
                    value={formData.secretKey}
                    onChange={(e) => handleInputChange('secretKey', e.target.value)}
                  />
                </div>
              </>
            )}
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setIsAddModalOpen(false)}>
              Cancel
            </Button>
            <Button
              type="button"
              onClick={handleAddDestination}
              disabled={!formData.name || !formData.path || (formData.type !== 'local' && !formData.host)}
            >
              Add Destination
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Destination Modal */}
      <Dialog open={!!editingDestination} onOpenChange={() => setEditingDestination(null)}>
        <DialogContent className="sm:max-w-[600px]">
          <DialogHeader>
            <DialogTitle>Edit Storage Destination</DialogTitle>
            <DialogDescription>
              Update configuration for this storage destination.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="edit-dest-name">Destination Name</Label>
              <Input
                id="edit-dest-name"
                value={formData.name}
                onChange={(e) => handleInputChange('name', e.target.value)}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="edit-dest-type">Storage Type</Label>
              <Select value={formData.type} onValueChange={(value) => handleInputChange('type', value)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="nfs">NFS</SelectItem>
                  <SelectItem value="cifs">CIFS/SMB</SelectItem>
                  <SelectItem value="local">Local Storage</SelectItem>
                  <SelectItem value="s3">Amazon S3</SelectItem>
                  <SelectItem value="azure">Azure Blob Storage</SelectItem>
                  <SelectItem value="gcs">Google Cloud Storage</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {(formData.type === 'nfs' || formData.type === 'cifs' || formData.type === 's3' || formData.type === 'azure' || formData.type === 'gcs') && (
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="edit-dest-host">Host</Label>
                  <Input
                    id="edit-dest-host"
                    value={formData.host}
                    onChange={(e) => handleInputChange('host', e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="edit-dest-port">Port</Label>
                  <Input
                    id="edit-dest-port"
                    type="number"
                    value={formData.port}
                    onChange={(e) => handleInputChange('port', e.target.value)}
                  />
                </div>
              </div>
            )}

            <div className="space-y-2">
              <Label htmlFor="edit-dest-path">Path</Label>
              <Input
                id="edit-dest-path"
                value={formData.path}
                onChange={(e) => handleInputChange('path', e.target.value)}
              />
            </div>

            {(formData.type === 'cifs' || formData.type === 'nfs') && (
              <>
                <div className="space-y-2">
                  <Label htmlFor="edit-dest-username">Username</Label>
                  <Input
                    id="edit-dest-username"
                    value={formData.username}
                    onChange={(e) => handleInputChange('username', e.target.value)}
                  />
                </div>

                <div className="space-y-2">
                  <Label htmlFor="edit-dest-password">Password</Label>
                  <Input
                    id="edit-dest-password"
                    type="password"
                    placeholder="Leave blank to keep current password"
                    value={formData.password}
                    onChange={(e) => handleInputChange('password', e.target.value)}
                  />
                </div>
              </>
            )}

            {(formData.type === 's3' || formData.type === 'azure' || formData.type === 'gcs') && (
              <>
                <div className="space-y-2">
                  <Label htmlFor="edit-dest-access-key">
                    {formData.type === 's3' ? 'Access Key' : formData.type === 'azure' ? 'Account Name' : 'Service Account'}
                  </Label>
                  <Input
                    id="edit-dest-access-key"
                    value={formData.accessKey}
                    onChange={(e) => handleInputChange('accessKey', e.target.value)}
                  />
                </div>

                <div className="space-y-2">
                  <Label htmlFor="edit-dest-secret-key">
                    {formData.type === 's3' ? 'Secret Key' : formData.type === 'azure' ? 'Account Key' : 'Private Key'}
                  </Label>
                  <Input
                    id="edit-dest-secret-key"
                    type="password"
                    placeholder="Leave blank to keep current secret"
                    value={formData.secretKey}
                    onChange={(e) => handleInputChange('secretKey', e.target.value)}
                  />
                </div>
              </>
            )}
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setEditingDestination(null)}>
              Cancel
            </Button>
            <Button
              type="button"
              onClick={handleUpdateDestination}
              disabled={!formData.name || !formData.path || (formData.type !== 'local' && !formData.host)}
            >
              Update Destination
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
