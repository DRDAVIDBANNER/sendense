"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { PageHeader } from "@/components/common/PageHeader";
import {
  CheckCircle,
  Clock,
  MoreHorizontal,
  Server,
  AlertTriangle,
  XCircle,
  HardDrive
} from "lucide-react";
import { format } from "date-fns";

interface Appliance {
  id: string;
  name: string;
  type: 'SNA' | 'SHA';
  status: 'pending' | 'approved' | 'online' | 'offline' | 'degraded';
  site_id: string;
  site_name: string;
  ip_address: string;
  last_seen: string;
  performance: {
    throughput: number;
    cpu_usage: number;
    memory_usage: number;
    disk_usage: number;
  };
}

interface Site {
  id: string;
  name: string;
  description: string;
  location: string;
  appliance_count: number;
  status: 'healthy' | 'degraded' | 'offline';
}

const mockSites: Site[] = [
  {
    id: 'site-1',
    name: 'OMA Primary',
    description: 'Primary Sendense Hub Appliance location',
    location: 'OMA Appliance Server Room',
    appliance_count: 1,
    status: 'healthy'
  },
  {
    id: 'site-2',
    name: 'VCenter Cluster',
    description: 'vCenter-integrated node appliances',
    location: 'Virtual Infrastructure',
    appliance_count: 1,
    status: 'healthy'
  },
  {
    id: 'site-3',
    name: 'DR Site',
    description: 'Disaster recovery backup location',
    location: 'Secondary Data Center',
    appliance_count: 1,
    status: 'degraded'
  }
];

const mockAppliances: Appliance[] = [
  {
    id: '1',
    name: 'oma-appliance-01',
    type: 'SHA',
    status: 'online',
    site_id: 'site-1',
    site_name: 'OMA Primary',
    ip_address: '10.245.246.125',
    last_seen: '2025-10-06T14:30:00Z',
    performance: {
      throughput: 850,
      cpu_usage: 45,
      memory_usage: 62,
      disk_usage: 78
    }
  },
  {
    id: '2',
    name: 'vcenter-node-01',
    type: 'SNA',
    status: 'online',
    site_id: 'site-2',
    site_name: 'VCenter Cluster',
    ip_address: '10.0.100.231',
    last_seen: '2025-10-06T14:25:00Z',
    performance: {
      throughput: 620,
      cpu_usage: 38,
      memory_usage: 45,
      disk_usage: 52
    }
  },
  {
    id: '3',
    name: 'pending-appliance-01',
    type: 'SNA',
    status: 'pending',
    site_id: '',
    site_name: 'Unassigned',
    ip_address: '192.168.1.100',
    last_seen: '2025-10-06T12:00:00Z',
    performance: {
      throughput: 0,
      cpu_usage: 0,
      memory_usage: 0,
      disk_usage: 0
    }
  },
  {
    id: '4',
    name: 'backup-site-01',
    type: 'SHA',
    status: 'degraded',
    site_id: 'site-3',
    site_name: 'DR Site',
    ip_address: '10.10.10.50',
    last_seen: '2025-10-06T13:45:00Z',
    performance: {
      throughput: 120,
      cpu_usage: 85,
      memory_usage: 92,
      disk_usage: 95
    }
  }
];

export default function AppliancesPage() {
  const [appliances, setAppliances] = useState<Appliance[]>(mockAppliances);
  const [sites, setSites] = useState<Site[]>(mockSites);
  const [selectedSiteFilter, setSelectedSiteFilter] = useState<string>('all');

  const getStatusIcon = (status: Appliance['status']) => {
    switch (status) {
      case 'online':
        return <CheckCircle className="h-4 w-4 text-green-500" />;
      case 'offline':
        return <XCircle className="h-4 w-4 text-red-500" />;
      case 'degraded':
        return <AlertTriangle className="h-4 w-4 text-yellow-500" />;
      case 'pending':
        return <Clock className="h-4 w-4 text-blue-500" />;
      default:
        return <Clock className="h-4 w-4 text-gray-500" />;
    }
  };

  const getStatusBadge = (status: Appliance['status']) => {
    const variants = {
      online: 'bg-green-500/10 text-green-400 border-green-500/20',
      offline: 'bg-red-500/10 text-red-400 border-red-500/20',
      degraded: 'bg-yellow-500/10 text-yellow-400 border-yellow-500/20',
      pending: 'bg-blue-500/10 text-blue-400 border-blue-500/20',
      approved: 'bg-purple-500/10 text-purple-400 border-purple-500/20'
    };

    return (
      <Badge className={variants[status]}>
        {status.charAt(0).toUpperCase() + status.slice(1)}
      </Badge>
    );
  };

  const getTypeBadge = (type: Appliance['type']) => {
    return (
      <Badge variant="outline" className="gap-1">
        {type === 'SHA' ? <HardDrive className="h-3 w-3" /> : <Server className="h-3 w-3" />}
        {type}
      </Badge>
    );
  };

  const handleApproveAppliance = (id: string) => {
    setAppliances(prev =>
      prev.map(appliance =>
        appliance.id === id
          ? { ...appliance, status: 'approved' as const }
          : appliance
      )
    );
  };

  const handleRejectAppliance = (id: string) => {
    setAppliances(prev => prev.filter(appliance => appliance.id !== id));
  };

  // Filter appliances by selected site
  const filteredAppliances = selectedSiteFilter === 'all'
    ? appliances
    : appliances.filter(a => a.site_id === selectedSiteFilter);

  // Calculate statistics
  const stats = {
    total: filteredAppliances.length,
    online: filteredAppliances.filter(a => a.status === 'online').length,
    pending: filteredAppliances.filter(a => a.status === 'pending').length,
    degraded: filteredAppliances.filter(a => a.status === 'degraded').length
  };

  // Site statistics
  const siteStats = {
    total: sites.length,
    healthy: sites.filter(s => s.status === 'healthy').length,
    degraded: sites.filter(s => s.status === 'degraded').length,
    offline: sites.filter(s => s.status === 'offline').length
  };

  return (
    <div className="h-full flex flex-col">
      <PageHeader
        title="Appliances"
        breadcrumbs={[
          { label: "Dashboard", href: "/dashboard" },
          { label: "Appliances" }
        ]}
      />

      <div className="flex-1 overflow-auto">
        <div className="p-6 space-y-6">
          <Tabs defaultValue="appliances" className="space-y-6">
            <TabsList>
              <TabsTrigger value="appliances">Appliances</TabsTrigger>
              <TabsTrigger value="sites">Sites</TabsTrigger>
            </TabsList>

            <TabsContent value="appliances" className="space-y-6">
              {/* Site Filter */}
              <div className="flex items-center gap-4">
                <div className="flex items-center gap-2">
                  <label className="text-sm font-medium">Filter by Site:</label>
                  <Select value={selectedSiteFilter} onValueChange={setSelectedSiteFilter}>
                    <SelectTrigger className="w-48">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">All Sites</SelectItem>
                      {sites.map(site => (
                        <SelectItem key={site.id} value={site.id}>
                          {site.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              </div>

              {/* Statistics Cards */}
              <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Total Appliances</CardTitle>
                <Server className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{stats.total}</div>
                <p className="text-xs text-muted-foreground">
                  {stats.online} online, {stats.pending} pending
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Online</CardTitle>
                <CheckCircle className="h-4 w-4 text-green-500" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold text-green-600">{stats.online}</div>
                <p className="text-xs text-muted-foreground">
                  Fully operational
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Pending Approval</CardTitle>
                <Clock className="h-4 w-4 text-blue-500" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold text-blue-600">{stats.pending}</div>
                <p className="text-xs text-muted-foreground">
                  Awaiting approval
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Degraded</CardTitle>
                <AlertTriangle className="h-4 w-4 text-yellow-500" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold text-yellow-600">{stats.degraded}</div>
                <p className="text-xs text-muted-foreground">
                  Needs attention
                </p>
              </CardContent>
            </Card>
          </div>

          {/* Appliances Table */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Server className="h-5 w-5" />
                Appliance Management
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="border border-border rounded-lg overflow-hidden">
                <Table>
                  <TableHeader>
                    <TableRow className="bg-muted/50">
                      <TableHead className="w-[200px]">Name</TableHead>
                      <TableHead className="w-[100px]">Type</TableHead>
                      <TableHead className="w-[120px]">Status</TableHead>
                      <TableHead className="w-[150px]">Site</TableHead>
                      <TableHead className="w-[140px]">IP Address</TableHead>
                      <TableHead className="w-[160px]">Last Seen</TableHead>
                      <TableHead className="w-[120px]">Throughput</TableHead>
                      <TableHead className="w-[100px]">Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {appliances.length === 0 ? (
                      <TableRow>
                        <TableCell colSpan={8} className="text-center py-8 text-muted-foreground">
                          No appliances found
                        </TableCell>
                      </TableRow>
                    ) : (
                      filteredAppliances.map((appliance) => (
                        <TableRow key={appliance.id}>
                          <TableCell className="font-medium">
                            <div className="flex items-center gap-2">
                              {getStatusIcon(appliance.status)}
                              {appliance.name}
                            </div>
                          </TableCell>
                          <TableCell>
                            {getTypeBadge(appliance.type)}
                          </TableCell>
                          <TableCell>
                            {getStatusBadge(appliance.status)}
                          </TableCell>
                          <TableCell>{appliance.site_name}</TableCell>
                          <TableCell className="font-mono text-sm">
                            {appliance.ip_address}
                          </TableCell>
                          <TableCell className="text-sm text-muted-foreground">
                            {format(new Date(appliance.last_seen), 'MMM dd, HH:mm')}
                          </TableCell>
                          <TableCell>
                            {appliance.status === 'online' || appliance.status === 'degraded' ? (
                              <span className="font-medium">
                                {appliance.performance.throughput} MB/s
                              </span>
                            ) : (
                              <span className="text-muted-foreground">-</span>
                            )}
                          </TableCell>
                          <TableCell>
                            {appliance.status === 'pending' ? (
                              <div className="flex items-center gap-2">
                                <Button
                                  size="sm"
                                  onClick={() => handleApproveAppliance(appliance.id)}
                                  className="h-8 px-2"
                                >
                                  Approve
                                </Button>
                                <Button
                                  size="sm"
                                  variant="outline"
                                  onClick={() => handleRejectAppliance(appliance.id)}
                                  className="h-8 px-2"
                                >
                                  Reject
                                </Button>
                              </div>
                            ) : (
                              <DropdownMenu>
                                <DropdownMenuTrigger asChild>
                                  <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
                                    <MoreHorizontal className="h-4 w-4" />
                                  </Button>
                                </DropdownMenuTrigger>
                                <DropdownMenuContent align="end">
                                  <DropdownMenuItem>View Details</DropdownMenuItem>
                                  <DropdownMenuItem>Edit Configuration</DropdownMenuItem>
                                  <DropdownMenuItem>View Logs</DropdownMenuItem>
                                </DropdownMenuContent>
                              </DropdownMenu>
                            )}
                          </TableCell>
                        </TableRow>
                      ))
                    )}
                  </TableBody>
                </Table>
              </div>
            </CardContent>
          </Card>
            </TabsContent>

            <TabsContent value="sites" className="space-y-6">
              {/* Site Statistics */}
              <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
                <Card>
                  <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                    <CardTitle className="text-sm font-medium">Total Sites</CardTitle>
                    <Server className="h-4 w-4 text-muted-foreground" />
                  </CardHeader>
                  <CardContent>
                    <div className="text-2xl font-bold">{siteStats.total}</div>
                    <p className="text-xs text-muted-foreground">
                      {siteStats.healthy} healthy, {siteStats.degraded} degraded
                    </p>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                    <CardTitle className="text-sm font-medium">Healthy Sites</CardTitle>
                    <CheckCircle className="h-4 w-4 text-green-500" />
                  </CardHeader>
                  <CardContent>
                    <div className="text-2xl font-bold text-green-600">{siteStats.healthy}</div>
                    <p className="text-xs text-muted-foreground">
                      All appliances operational
                    </p>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                    <CardTitle className="text-sm font-medium">Degraded Sites</CardTitle>
                    <AlertTriangle className="h-4 w-4 text-yellow-500" />
                  </CardHeader>
                  <CardContent>
                    <div className="text-2xl font-bold text-yellow-600">{siteStats.degraded}</div>
                    <p className="text-xs text-muted-foreground">
                      Some appliances need attention
                    </p>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                    <CardTitle className="text-sm font-medium">Offline Sites</CardTitle>
                    <XCircle className="h-4 w-4 text-red-500" />
                  </CardHeader>
                  <CardContent>
                    <div className="text-2xl font-bold text-red-600">{siteStats.offline}</div>
                    <p className="text-xs text-muted-foreground">
                      No connectivity
                    </p>
                  </CardContent>
                </Card>
              </div>

              {/* Sites Management */}
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Server className="h-5 w-5" />
                    Site Management
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="border border-border rounded-lg overflow-hidden">
                    <Table>
                      <TableHeader>
                        <TableRow className="bg-muted/50">
                          <TableHead className="w-[200px]">Site Name</TableHead>
                          <TableHead className="w-[300px]">Description</TableHead>
                          <TableHead className="w-[200px]">Location</TableHead>
                          <TableHead className="w-[120px]">Appliances</TableHead>
                          <TableHead className="w-[120px]">Status</TableHead>
                          <TableHead className="w-[100px]">Actions</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {sites.length === 0 ? (
                          <TableRow>
                            <TableCell colSpan={6} className="text-center py-8 text-muted-foreground">
                              No sites configured
                            </TableCell>
                          </TableRow>
                        ) : (
                          sites.map((site) => {
                            const getSiteStatusIcon = (status: Site['status']) => {
                              switch (status) {
                                case 'healthy':
                                  return <CheckCircle className="h-4 w-4 text-green-500" />;
                                case 'degraded':
                                  return <AlertTriangle className="h-4 w-4 text-yellow-500" />;
                                case 'offline':
                                  return <XCircle className="h-4 w-4 text-red-500" />;
                                default:
                                  return <Clock className="h-4 w-4 text-gray-500" />;
                              }
                            };

                            const getSiteStatusBadge = (status: Site['status']) => {
                              const variants = {
                                healthy: 'bg-green-500/10 text-green-400 border-green-500/20',
                                degraded: 'bg-yellow-500/10 text-yellow-400 border-yellow-500/20',
                                offline: 'bg-red-500/10 text-red-400 border-red-500/20'
                              };

                              return (
                                <Badge className={variants[status]}>
                                  {status.charAt(0).toUpperCase() + status.slice(1)}
                                </Badge>
                              );
                            };

                            return (
                              <TableRow key={site.id}>
                                <TableCell className="font-medium">
                                  <div className="flex items-center gap-2">
                                    {getSiteStatusIcon(site.status)}
                                    {site.name}
                                  </div>
                                </TableCell>
                                <TableCell className="text-sm text-muted-foreground">
                                  {site.description}
                                </TableCell>
                                <TableCell>{site.location}</TableCell>
                                <TableCell>
                                  <Badge variant="secondary">
                                    {site.appliance_count} appliances
                                  </Badge>
                                </TableCell>
                                <TableCell>
                                  {getSiteStatusBadge(site.status)}
                                </TableCell>
                                <TableCell>
                                  <DropdownMenu>
                                    <DropdownMenuTrigger asChild>
                                      <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
                                        <MoreHorizontal className="h-4 w-4" />
                                      </Button>
                                    </DropdownMenuTrigger>
                                    <DropdownMenuContent align="end">
                                      <DropdownMenuItem>View Appliances</DropdownMenuItem>
                                      <DropdownMenuItem>Edit Site</DropdownMenuItem>
                                      <DropdownMenuItem>Site Health Report</DropdownMenuItem>
                                    </DropdownMenuContent>
                                  </DropdownMenu>
                                </TableCell>
                              </TableRow>
                            );
                          })
                        )}
                      </TableBody>
                    </Table>
                  </div>
                </CardContent>
              </Card>
            </TabsContent>
          </Tabs>
        </div>
      </div>
    </div>
  );
}
