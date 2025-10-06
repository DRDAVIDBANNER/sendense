"use client";

import { useState } from "react";
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
  datacenterCount: number;
  vmCount: number;
}

const mockSources: VCenterSource[] = [
  {
    id: '1',
    name: 'Production vCenter',
    host: 'vc-prod.company.com',
    port: 443,
    username: 'administrator@vsphere.local',
    status: 'connected',
    lastConnected: '2025-10-06T08:00:00Z',
    version: '8.0 Update 2',
    datacenterCount: 2,
    vmCount: 47
  },
  {
    id: '2',
    name: 'Development vCenter',
    host: 'vc-dev.company.com',
    port: 443,
    username: 'administrator@vsphere.local',
    status: 'connected',
    lastConnected: '2025-10-06T07:30:00Z',
    version: '8.0 Update 1',
    datacenterCount: 1,
    vmCount: 23
  },
  {
    id: '3',
    name: 'Legacy vCenter',
    host: 'vc-legacy.company.com',
    port: 443,
    username: 'administrator@vsphere.local',
    status: 'error',
    lastConnected: '2025-10-05T15:00:00Z',
    version: '6.7 Update 3',
    datacenterCount: 1,
    vmCount: 12
  }
];

export default function SourcesPage() {
  const [sources, setSources] = useState<VCenterSource[]>(mockSources);
  const [isAddModalOpen, setIsAddModalOpen] = useState(false);
  const [editingSource, setEditingSource] = useState<VCenterSource | null>(null);

  const [formData, setFormData] = useState({
    name: '',
    host: '',
    port: '443',
    username: '',
    password: ''
  });

  const handleInputChange = (field: string, value: string) => {
    setFormData(prev => ({ ...prev, [field]: value }));
  };

  const handleTestConnection = async (sourceId: string) => {
    // Simulate connection test
    setSources(prev => prev.map(source =>
      source.id === sourceId
        ? { ...source, status: 'connected' as const, lastConnected: new Date().toISOString() }
        : source
    ));
  };

  const handleAddSource = () => {
    const newSource: VCenterSource = {
      id: Date.now().toString(),
      name: formData.name,
      host: formData.host,
      port: parseInt(formData.port),
      username: formData.username,
      status: 'disconnected',
      lastConnected: '',
      datacenterCount: 0,
      vmCount: 0
    };

    setSources(prev => [...prev, newSource]);
    setFormData({ name: '', host: '', port: '443', username: '', password: '' });
    setIsAddModalOpen(false);
  };

  const handleEditSource = (source: VCenterSource) => {
    setEditingSource(source);
    setFormData({
      name: source.name,
      host: source.host,
      port: source.port.toString(),
      username: source.username,
      password: ''
    });
  };

  const handleUpdateSource = () => {
    if (!editingSource) return;

    setSources(prev => prev.map(source =>
      source.id === editingSource.id
        ? {
            ...source,
            name: formData.name,
            host: formData.host,
            port: parseInt(formData.port),
            username: formData.username
          }
        : source
    ));

    setEditingSource(null);
    setFormData({ name: '', host: '', port: '443', username: '', password: '' });
  };

  const handleDeleteSource = (sourceId: string) => {
    setSources(prev => prev.filter(source => source.id !== sourceId));
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
            {sources.map((source) => (
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
            ))}

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
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setIsAddModalOpen(false)}>
              Cancel
            </Button>
            <Button
              type="button"
              onClick={handleAddSource}
              disabled={!formData.name || !formData.host || !formData.username || !formData.password}
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
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setEditingSource(null)}>
              Cancel
            </Button>
            <Button
              type="button"
              onClick={handleUpdateSource}
              disabled={!formData.name || !formData.host || !formData.username}
            >
              Update Source
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
