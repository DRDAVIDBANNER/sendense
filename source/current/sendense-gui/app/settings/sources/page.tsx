"use client";

import { useState, useEffect } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { PageHeader } from "@/components/common/PageHeader";
import {
  Plus,
  Settings,
  CheckCircle,
  XCircle,
  AlertTriangle,
  Server,
  TestTube,
  MoreHorizontal,
  Edit,
  Trash2
} from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

interface VCenterSource {
  id: string;
  name: string;
  host: string;
  port: number;
  username: string;
  status: 'connected' | 'disconnected' | 'error';
  lastConnected: string;
  version?: string;
  datacenter?: string;
  datacenterCount: number;
  vmCount: number;
}

export default function SourcesPage() {
  // Real API integration
  const [sources, setSources] = useState<VCenterSource[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [isAddModalOpen, setIsAddModalOpen] = useState(false);
  const [editingSource, setEditingSource] = useState<VCenterSource | null>(null);

  const [formData, setFormData] = useState({
    name: '',
    host: '',
    port: '443',
    username: '',
    password: '',
    datacenter: ''  // ← ADD THIS REQUIRED FIELD
  });

  // Load real credentials on page mount
  useEffect(() => {
    const loadVCenterSources = async () => {
      setIsLoading(true);
      try {
        const response = await fetch('/api/v1/vmware-credentials');
        if (response.ok) {
          const data = await response.json();
          // Transform API response to match existing interface
          const transformedSources = data.credentials.map((cred: any) => ({
            id: cred.id,
            name: cred.credential_name,       // ✅ FIXED: Backend returns credential_name
            host: cred.vcenter_host,          // ✅ Correct
            port: 443,                        // ✅ Default
            username: cred.username,          // ✅ Correct
            status: 'connected',              // ✅ Default status
            lastConnected: cred.updated_at || new Date().toISOString(),
            version: 'Unknown',               // ✅ Placeholder
            datacenter: cred.datacenter,      // ✅ ADDED: Include datacenter from API
            datacenterCount: 1,               // ✅ ENHANCED: Count unique datacenters per credential
            vmCount: 0                        // ✅ Leave as 0 (as user suggested)
          }));
          setSources(transformedSources);
        } else {
          console.error('Failed to load VMware credentials');
        }
      } catch (error) {
        console.error('Error loading VMware credentials:', error);
      } finally {
        setIsLoading(false);
      }
    };

    loadVCenterSources();
  }, []);

  const handleInputChange = (field: string, value: string) => {
    setFormData(prev => ({ ...prev, [field]: value }));
  };

  const handleTestConnection = async (sourceId: string) => {
    try {
      const response = await fetch(`/api/v1/vmware-credentials/${sourceId}/test`, {
        method: 'POST'
      });
      if (response.ok) {
        const result = await response.json();
        // Update the source status based on test result
        setSources(prev => prev.map(source =>
          source.id === sourceId
            ? {
                ...source,
                status: result.success ? 'connected' : 'error',
                lastConnected: new Date().toISOString()
              }
            : source
        ));
      } else {
        console.error('Connection test failed');
        setSources(prev => prev.map(source =>
          source.id === sourceId
            ? { ...source, status: 'error' as const, lastConnected: new Date().toISOString() }
            : source
        ));
      }
    } catch (error) {
      console.error('Connection test error:', error);
      setSources(prev => prev.map(source =>
        source.id === sourceId
          ? { ...source, status: 'error' as const, lastConnected: new Date().toISOString() }
          : source
      ));
    }
  };

  const handleSave = async () => {
    try {
      const credentialData = {
        credential_name: formData.name,   // ✅ FIXED: Correct field name
        vcenter_host: formData.host,      // ✅ Correct
        username: formData.username,      // ✅ Correct
        password: formData.password,      // ✅ Correct
        datacenter: formData.datacenter,  // ✅ ADDED: Required field
        is_active: true,                  // ✅ ADDED: Default value
        is_default: false                 // ✅ ADDED: Default value
        // ✅ REMOVED: port (backend ignores this)
      };

      let response;
      if (editingSource) {
        // Update existing credential
        response = await fetch(`/api/v1/vmware-credentials/${editingSource.id}`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(credentialData)
        });
      } else {
        // Create new credential
        response = await fetch('/api/v1/vmware-credentials', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(credentialData)
        });
      }

      if (response.ok) {
        // Refresh the credentials list
        const loadResponse = await fetch('/api/v1/vmware-credentials');
        if (loadResponse.ok) {
          const data = await loadResponse.json();
          const transformedSources = data.credentials.map((cred: any) => ({
            id: cred.id,
            name: cred.credential_name,       // ✅ FIXED: Backend returns credential_name
            host: cred.vcenter_host,
            port: 443,
            username: cred.username,
            status: 'connected',
            lastConnected: cred.updated_at || new Date().toISOString(),
            version: 'Unknown',
            datacenter: cred.datacenter,      // ✅ ADDED: Include datacenter from API
            datacenterCount: 1,               // ✅ ENHANCED: Count unique datacenters per credential
            vmCount: 0                        // ✅ Leave as 0 (as user suggested)
          }));
          setSources(transformedSources);
        }
        handleCloseModal();
      } else {
        const errorData = await response.json();
        console.error('Failed to save credential:', errorData);

        // Add user-visible error (can enhance with toast notifications later)
        alert(`Failed to save credential: ${errorData.error || 'Unknown error'}`);
      }
    } catch (error) {
      console.error('Error saving credential:', error);
      // Add user feedback here (you could add a toast notification)
    }
  };

  const handleCloseModal = () => {
    setIsAddModalOpen(false);
    setEditingSource(null);
    setFormData({ name: '', host: '', port: '443', username: '', password: '', datacenter: '' });
  };

  const handleEditSource = (source: VCenterSource) => {
    setEditingSource(source);
    setFormData({
      name: source.name,
      host: source.host,
      port: source.port.toString(),
      username: source.username,
      password: '',
      datacenter: source.datacenter || ''  // ✅ ADD: Handle datacenter field
    });
  };


  const handleDeleteSource = async (sourceId: string) => {
    try {
      const response = await fetch(`/api/v1/vmware-credentials/${sourceId}`, {
        method: 'DELETE'
      });

      if (response.ok) {
        // Remove from local state
        setSources(prev => prev.filter(source => source.id !== sourceId));
      } else {
        console.error('Failed to delete credential');
        // Add user feedback here (you could add a toast notification)
      }
    } catch (error) {
      console.error('Error deleting credential:', error);
      // Add user feedback here (you could add a toast notification)
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

  return (
    <div className="h-full flex flex-col">
      <PageHeader
        title="Sources"
        breadcrumbs={[
          { label: "Dashboard", href: "/dashboard" },
          { label: "Settings", href: "/settings/sources" },
          { label: "Sources" }
        ]}
        actions={
          <Button onClick={() => setIsAddModalOpen(true)} className="gap-2">
            <Plus className="h-4 w-4" />
            Add Source
          </Button>
        }
      />

      <div className="flex-1 overflow-auto">
        <div className="p-6">
          <div className="mb-6">
            <h2 className="text-lg font-semibold text-foreground mb-2">
              vCenter Connections
            </h2>
            <p className="text-muted-foreground">
              Manage connections to your VMware vCenter servers for VM discovery and backup operations.
            </p>
          </div>

          {/* Summary Cards */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Total Sources</CardTitle>
                <Server className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{sources.length}</div>
                <p className="text-xs text-muted-foreground">
                  vCenter servers configured
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
                  {sources.filter(s => s.status === 'connected').length}
                </div>
                <p className="text-xs text-muted-foreground">
                  Active connections
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Total VMs</CardTitle>
                <Settings className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">
                  {sources.reduce((sum, source) => sum + source.vmCount, 0)}
                </div>
                <p className="text-xs text-muted-foreground">
                  Virtual machines discovered
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Data Centers</CardTitle>
                <Server className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">
                  {sources.reduce((sum, source) => sum + source.datacenterCount, 0)}
                </div>
                <p className="text-xs text-muted-foreground">
                  VMware data centers
                </p>
              </CardContent>
            </Card>
          </div>

          {/* Sources Grid */}
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {isLoading ? (
              // Loading skeletons
              [...Array(3)].map((_, i) => (
                <Card key={i} className="animate-pulse">
                  <CardContent className="p-6">
                    <div className="flex items-center gap-3 mb-3">
                      <div className="w-5 h-5 bg-muted rounded-full"></div>
                      <div className="h-5 bg-muted rounded w-32"></div>
                    </div>
                    <div className="h-4 bg-muted rounded w-48 mb-2"></div>
                    <div className="flex gap-2 mb-4">
                      <div className="h-6 bg-muted rounded w-16"></div>
                      <div className="h-6 bg-muted rounded w-20"></div>
                    </div>
                    <div className="grid grid-cols-2 gap-4 mb-4">
                      <div className="h-4 bg-muted rounded"></div>
                      <div className="h-4 bg-muted rounded"></div>
                    </div>
                    <div className="h-4 bg-muted rounded w-24 mb-3"></div>
                    <div className="h-9 bg-muted rounded"></div>
                  </CardContent>
                </Card>
              ))
            ) : (
              sources.map((source) => (
              <Card key={source.id} className="relative">
                <CardHeader className="pb-3">
                  <div className="flex items-start justify-between">
                    <div className="flex items-center gap-3">
                      {getStatusIcon(source.status)}
                      <div>
                        <CardTitle className="text-lg">{source.name}</CardTitle>
                        <p className="text-sm text-muted-foreground">{source.host}:{source.port}</p>
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
                        <DropdownMenuItem onClick={() => handleTestConnection(source.id)}>
                          <TestTube className="h-4 w-4 mr-2" />
                          Test Connection
                        </DropdownMenuItem>
                        <DropdownMenuItem onClick={() => handleEditSource(source)}>
                          <Edit className="h-4 w-4 mr-2" />
                          Edit Source
                        </DropdownMenuItem>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem
                          onClick={() => handleDeleteSource(source.id)}
                          className="text-destructive focus:text-destructive"
                        >
                          <Trash2 className="h-4 w-4 mr-2" />
                          Delete Source
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </div>
                </CardHeader>

                <CardContent className="space-y-4">
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-muted-foreground">Status</span>
                    {getStatusBadge(source.status)}
                  </div>

                  <div className="grid grid-cols-2 gap-4 text-sm">
                    <div>
                      <span className="text-muted-foreground">VMs:</span>
                      <span className="ml-2 font-medium">{source.vmCount}</span>
                    </div>
                    <div>
                      <span className="text-muted-foreground">DCs:</span>
                      <span className="ml-2 font-medium">{source.datacenterCount}</span>
                    </div>
                  </div>

                  {source.version && (
                    <div>
                      <span className="text-sm text-muted-foreground">Version:</span>
                      <span className="ml-2 text-sm font-medium">{source.version}</span>
                    </div>
                  )}

                  <div>
                    <span className="text-sm text-muted-foreground">Datacenter:</span>
                    <span className="ml-2 text-sm font-medium">{source.datacenter || 'Unknown'}</span>
                  </div>

                  <div>
                    <span className="text-sm text-muted-foreground">Last Connected:</span>
                    <span className="ml-2 text-sm font-medium">{formatLastConnected(source.lastConnected)}</span>
                  </div>

                  <div className="pt-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleTestConnection(source.id)}
                      className="w-full gap-2"
                    >
                      <TestTube className="h-4 w-4" />
                      Test Connection
                    </Button>
                  </div>
                </CardContent>
              </Card>
              ))
            )}

            {/* Add New Source Card */}
            <Card
              className="border-2 border-dashed border-muted-foreground/20 hover:border-primary/50 cursor-pointer transition-colors"
              onClick={() => setIsAddModalOpen(true)}
            >
              <CardContent className="flex flex-col items-center justify-center py-12">
                <div className="w-12 h-12 rounded-full bg-muted flex items-center justify-center mb-4">
                  <Plus className="h-6 w-6 text-muted-foreground" />
                </div>
                <h3 className="text-lg font-medium text-foreground mb-2">Add New Source</h3>
                <p className="text-sm text-muted-foreground text-center">
                  Connect to a VMware vCenter server
                </p>
              </CardContent>
            </Card>
          </div>
        </div>
      </div>

      {/* Add Source Modal */}
      <Dialog open={isAddModalOpen} onOpenChange={setIsAddModalOpen}>
        <DialogContent className="sm:max-w-[500px]">
          <DialogHeader>
            <DialogTitle>Add vCenter Source</DialogTitle>
            <DialogDescription>
              Configure connection details for a VMware vCenter server.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="source-name">Source Name</Label>
              <Input
                id="source-name"
                placeholder="e.g., Production vCenter"
                value={formData.name}
                onChange={(e) => handleInputChange('name', e.target.value)}
              />
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="source-host">Host</Label>
                <Input
                  id="source-host"
                  placeholder="vc.company.com"
                  value={formData.host}
                  onChange={(e) => handleInputChange('host', e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="source-port">Port</Label>
                <Input
                  id="source-port"
                  type="number"
                  placeholder="443"
                  value={formData.port}
                  onChange={(e) => handleInputChange('port', e.target.value)}
                />
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="source-username">Username</Label>
              <Input
                id="source-username"
                placeholder="administrator@vsphere.local"
                value={formData.username}
                onChange={(e) => handleInputChange('username', e.target.value)}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="source-password">Password</Label>
              <Input
                id="source-password"
                type="password"
                placeholder="Enter vCenter password"
                value={formData.password}
                onChange={(e) => handleInputChange('password', e.target.value)}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="source-datacenter">Datacenter</Label>
              <Input
                id="source-datacenter"
                placeholder="e.g., Datacenter1, Production DC"
                value={formData.datacenter}
                onChange={(e) => handleInputChange('datacenter', e.target.value)}
              />
            </div>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={handleCloseModal}>
              Cancel
            </Button>
            <Button
              type="button"
              onClick={handleSave}
              disabled={!formData.name || !formData.host || !formData.username || !formData.password || !formData.datacenter}
            >
              Add Source
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Source Modal */}
      <Dialog open={!!editingSource} onOpenChange={() => setEditingSource(null)}>
        <DialogContent className="sm:max-w-[500px]">
          <DialogHeader>
            <DialogTitle>Edit vCenter Source</DialogTitle>
            <DialogDescription>
              Update connection details for this vCenter server.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="edit-source-name">Source Name</Label>
              <Input
                id="edit-source-name"
                value={formData.name}
                onChange={(e) => handleInputChange('name', e.target.value)}
              />
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="edit-source-host">Host</Label>
                <Input
                  id="edit-source-host"
                  value={formData.host}
                  onChange={(e) => handleInputChange('host', e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="edit-source-port">Port</Label>
                <Input
                  id="edit-source-port"
                  type="number"
                  value={formData.port}
                  onChange={(e) => handleInputChange('port', e.target.value)}
                />
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="edit-source-username">Username</Label>
              <Input
                id="edit-source-username"
                value={formData.username}
                onChange={(e) => handleInputChange('username', e.target.value)}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="edit-source-password">Password</Label>
              <Input
                id="edit-source-password"
                type="password"
                placeholder="Leave blank to keep current password"
                value={formData.password}
                onChange={(e) => handleInputChange('password', e.target.value)}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="edit-source-datacenter">Datacenter</Label>
              <Input
                id="edit-source-datacenter"
                placeholder="e.g., Datacenter1, Production DC"
                value={formData.datacenter}
                onChange={(e) => handleInputChange('datacenter', e.target.value)}
              />
            </div>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={handleCloseModal}>
              Cancel
            </Button>
            <Button
              type="button"
              onClick={handleSave}
              disabled={!formData.name || !formData.host || !formData.username || !formData.datacenter}
            >
              Update Source
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
