"use client";

import { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Checkbox } from "@/components/ui/checkbox";
import { PageHeader } from "@/components/common/PageHeader";
import {
  Plus,
  Users,
  User,
  Shield,
  Crown,
  Settings,
  MoreHorizontal,
  Edit,
  Trash2,
  Mail,
  Calendar,
  Activity
} from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

interface User {
  id: string;
  name: string;
  email: string;
  role: 'admin' | 'operator' | 'viewer';
  status: 'active' | 'inactive' | 'pending';
  lastLogin: string;
  createdAt: string;
  permissions: string[];
}

interface Role {
  name: 'admin' | 'operator' | 'viewer';
  label: string;
  description: string;
  permissions: string[];
}

const roles: Role[] = [
  {
    name: 'admin',
    label: 'Administrator',
    description: 'Full system access and management',
    permissions: ['all']
  },
  {
    name: 'operator',
    label: 'Operator',
    description: 'Backup operations and monitoring',
    permissions: ['backup:create', 'backup:read', 'backup:delete', 'vm:read', 'reports:read']
  },
  {
    name: 'viewer',
    label: 'Viewer',
    description: 'Read-only access to system information',
    permissions: ['backup:read', 'vm:read', 'reports:read']
  }
];

const allPermissions = [
  { key: 'backup:create', label: 'Create Backups' },
  { key: 'backup:read', label: 'View Backups' },
  { key: 'backup:delete', label: 'Delete Backups' },
  { key: 'backup:restore', label: 'Restore Backups' },
  { key: 'vm:read', label: 'View Virtual Machines' },
  { key: 'vm:manage', label: 'Manage VMs' },
  { key: 'reports:read', label: 'View Reports' },
  { key: 'reports:export', label: 'Export Reports' },
  { key: 'users:read', label: 'View Users' },
  { key: 'users:manage', label: 'Manage Users' },
  { key: 'settings:read', label: 'View Settings' },
  { key: 'settings:manage', label: 'Manage Settings' },
  { key: 'system:admin', label: 'System Administration' }
];

const mockUsers: User[] = [
  {
    id: '1',
    name: 'John Administrator',
    email: 'john.admin@company.com',
    role: 'admin',
    status: 'active',
    lastLogin: '2025-10-06T08:30:00Z',
    createdAt: '2025-01-15T10:00:00Z',
    permissions: ['all']
  },
  {
    id: '2',
    name: 'Sarah Operator',
    email: 'sarah.ops@company.com',
    role: 'operator',
    status: 'active',
    lastLogin: '2025-10-06T07:45:00Z',
    createdAt: '2025-02-20T14:30:00Z',
    permissions: ['backup:create', 'backup:read', 'backup:delete', 'vm:read', 'reports:read']
  },
  {
    id: '3',
    name: 'Mike Viewer',
    email: 'mike.view@company.com',
    role: 'viewer',
    status: 'active',
    lastLogin: '2025-10-05T16:20:00Z',
    createdAt: '2025-03-10T09:15:00Z',
    permissions: ['backup:read', 'vm:read', 'reports:read']
  },
  {
    id: '4',
    name: 'Alice Pending',
    email: 'alice.new@company.com',
    role: 'viewer',
    status: 'pending',
    lastLogin: '',
    createdAt: '2025-10-01T11:00:00Z',
    permissions: ['backup:read', 'vm:read', 'reports:read']
  }
];

export default function UsersPage() {
  const [users, setUsers] = useState<User[]>(mockUsers);
  const [isAddModalOpen, setIsAddModalOpen] = useState(false);
  const [editingUser, setEditingUser] = useState<User | null>(null);
  const [showRoleModal, setShowRoleModal] = useState(false);
  const [selectedRole, setSelectedRole] = useState<Role | null>(null);

  const [formData, setFormData] = useState({
    name: '',
    email: '',
    role: 'viewer' as User['role'],
    customPermissions: [] as string[]
  });

  const handleInputChange = (field: string, value: string | string[]) => {
    setFormData(prev => ({ ...prev, [field]: value }));
  };

  const handlePermissionChange = (permission: string, checked: boolean) => {
    setFormData(prev => ({
      ...prev,
      customPermissions: checked
        ? [...prev.customPermissions, permission]
        : prev.customPermissions.filter(p => p !== permission)
    }));
  };

  const handleAddUser = () => {
    const roleData = roles.find(r => r.name === formData.role)!;
    const permissions = formData.role === 'admin' ? ['all'] : formData.customPermissions;

    const newUser: User = {
      id: Date.now().toString(),
      name: formData.name,
      email: formData.email,
      role: formData.role,
      status: 'pending',
      lastLogin: '',
      createdAt: new Date().toISOString(),
      permissions
    };

    setUsers(prev => [...prev, newUser]);
    setFormData({ name: '', email: '', role: 'viewer', customPermissions: [] });
    setIsAddModalOpen(false);
  };

  const handleEditUser = (user: User) => {
    setEditingUser(user);
    setFormData({
      name: user.name,
      email: user.email,
      role: user.role,
      customPermissions: user.permissions.includes('all') ? [] : user.permissions
    });
  };

  const handleUpdateUser = () => {
    if (!editingUser) return;

    const roleData = roles.find(r => r.name === formData.role)!;
    const permissions = formData.role === 'admin' ? ['all'] : formData.customPermissions;

    setUsers(prev => prev.map(user =>
      user.id === editingUser.id
        ? {
            ...user,
            name: formData.name,
            email: formData.email,
            role: formData.role,
            permissions
          }
        : user
    ));

    setEditingUser(null);
    setFormData({ name: '', email: '', role: 'viewer', customPermissions: [] });
  };

  const handleDeleteUser = (userId: string) => {
    setUsers(prev => prev.filter(user => user.id !== userId));
  };

  const handleActivateUser = (userId: string) => {
    setUsers(prev => prev.map(user =>
      user.id === userId
        ? { ...user, status: 'active' as const }
        : user
    ));
  };

  const getRoleIcon = (role: string) => {
    switch (role) {
      case 'admin':
        return <Crown className="h-4 w-4 text-yellow-500" />;
      case 'operator':
        return <Shield className="h-4 w-4 text-blue-500" />;
      case 'viewer':
        return <User className="h-4 w-4 text-gray-500" />;
      default:
        return <User className="h-4 w-4 text-gray-500" />;
    }
  };

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'active':
        return <Badge className="bg-green-500/10 text-green-400 border-green-500/20">Active</Badge>;
      case 'inactive':
        return <Badge variant="secondary">Inactive</Badge>;
      case 'pending':
        return <Badge className="bg-yellow-500/10 text-yellow-400 border-yellow-500/20">Pending</Badge>;
      default:
        return <Badge variant="secondary">Unknown</Badge>;
    }
  };

  const formatDate = (timestamp: string) => {
    if (!timestamp) return 'Never';

    const date = new Date(timestamp);
    return date.toLocaleDateString() + ' ' + date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  };

  const getRolePermissions = (roleName: string) => {
    const role = roles.find(r => r.name === roleName);
    return role ? role.permissions : [];
  };

  const getUserPermissions = (user: User) => {
    if (user.permissions.includes('all')) {
      return 'All Permissions';
    }
    return user.permissions.length + ' permissions';
  };

  const activeUsers = users.filter(u => u.status === 'active').length;
  const pendingUsers = users.filter(u => u.status === 'pending').length;
  const totalUsers = users.length;

  return (
    <div className="h-full flex flex-col">
      <PageHeader
        title="User Management"
        breadcrumbs={[
          { label: "Dashboard", href: "/dashboard" },
          { label: "Users" }
        ]}
        actions={
          <div className="flex gap-2">
            <Button
              variant="outline"
              onClick={() => setShowRoleModal(true)}
              className="gap-2"
            >
              <Shield className="h-4 w-4" />
              Manage Roles
            </Button>
            <Button onClick={() => setIsAddModalOpen(true)} className="gap-2">
              <Plus className="h-4 w-4" />
              Add User
            </Button>
          </div>
        }
      />

      <div className="flex-1 overflow-auto">
        <div className="p-6">
          <div className="mb-6">
            <h2 className="text-lg font-semibold text-foreground mb-2">
              User Accounts
            </h2>
            <p className="text-muted-foreground">
              Manage user access, roles, and permissions for your Sendense deployment.
            </p>
          </div>

          {/* Summary Cards */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Total Users</CardTitle>
                <Users className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{totalUsers}</div>
                <p className="text-xs text-muted-foreground">
                  Registered accounts
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Active Users</CardTitle>
                <Activity className="h-4 w-4 text-green-500" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold text-green-600">{activeUsers}</div>
                <p className="text-xs text-muted-foreground">
                  Currently active
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Pending Users</CardTitle>
                <Calendar className="h-4 w-4 text-yellow-500" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold text-yellow-600">{pendingUsers}</div>
                <p className="text-xs text-muted-foreground">
                  Awaiting activation
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Administrators</CardTitle>
                <Crown className="h-4 w-4 text-yellow-500" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold text-yellow-600">
                  {users.filter(u => u.role === 'admin').length}
                </div>
                <p className="text-xs text-muted-foreground">
                  Admin accounts
                </p>
              </CardContent>
            </Card>
          </div>

          {/* Users Table */}
          <Card>
            <CardHeader>
              <CardTitle>User Accounts</CardTitle>
            </CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>User</TableHead>
                    <TableHead>Role</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Permissions</TableHead>
                    <TableHead>Last Login</TableHead>
                    <TableHead>Created</TableHead>
                    <TableHead className="w-[50px]"></TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {users.map((user) => (
                    <TableRow key={user.id}>
                      <TableCell>
                        <div className="flex items-center gap-3">
                          <div className="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center">
                            <User className="h-4 w-4" />
                          </div>
                          <div>
                            <div className="font-medium">{user.name}</div>
                            <div className="text-sm text-muted-foreground flex items-center gap-1">
                              <Mail className="h-3 w-3" />
                              {user.email}
                            </div>
                          </div>
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          {getRoleIcon(user.role)}
                          <span className="capitalize">{user.role}</span>
                        </div>
                      </TableCell>
                      <TableCell>{getStatusBadge(user.status)}</TableCell>
                      <TableCell>
                        <span className="text-sm">{getUserPermissions(user)}</span>
                      </TableCell>
                      <TableCell>
                        <span className="text-sm">{formatDate(user.lastLogin)}</span>
                      </TableCell>
                      <TableCell>
                        <span className="text-sm">{formatDate(user.createdAt)}</span>
                      </TableCell>
                      <TableCell>
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
                            {user.status === 'pending' && (
                              <DropdownMenuItem onClick={() => handleActivateUser(user.id)}>
                                <Activity className="h-4 w-4 mr-2" />
                                Activate User
                              </DropdownMenuItem>
                            )}
                            <DropdownMenuItem onClick={() => handleEditUser(user)}>
                              <Edit className="h-4 w-4 mr-2" />
                              Edit User
                            </DropdownMenuItem>
                            <DropdownMenuSeparator />
                            <DropdownMenuItem
                              onClick={() => handleDeleteUser(user.id)}
                              className="text-destructive focus:text-destructive"
                            >
                              <Trash2 className="h-4 w-4 mr-2" />
                              Delete User
                            </DropdownMenuItem>
                          </DropdownMenuContent>
                        </DropdownMenu>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </div>
      </div>

      {/* Add User Modal */}
      <Dialog open={isAddModalOpen} onOpenChange={setIsAddModalOpen}>
        <DialogContent className="sm:max-w-[600px]">
          <DialogHeader>
            <DialogTitle>Add New User</DialogTitle>
            <DialogDescription>
              Create a new user account with appropriate role and permissions.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="user-name">Full Name</Label>
              <Input
                id="user-name"
                placeholder="John Doe"
                value={formData.name}
                onChange={(e) => handleInputChange('name', e.target.value)}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="user-email">Email Address</Label>
              <Input
                id="user-email"
                type="email"
                placeholder="john.doe@company.com"
                value={formData.email}
                onChange={(e) => handleInputChange('email', e.target.value)}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="user-role">Role</Label>
              <Select value={formData.role} onValueChange={(value) => handleInputChange('role', value)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {roles.map((role) => (
                    <SelectItem key={role.name} value={role.name}>
                      <div className="flex items-center gap-2">
                        {getRoleIcon(role.name)}
                        <span>{role.label}</span>
                      </div>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <p className="text-sm text-muted-foreground">
                {roles.find(r => r.name === formData.role)?.description}
              </p>
            </div>

            {formData.role !== 'admin' && (
              <div className="space-y-3">
                <Label>Custom Permissions</Label>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-3 max-h-48 overflow-y-auto">
                  {allPermissions.map((permission) => (
                    <div key={permission.key} className="flex items-center space-x-2">
                      <Checkbox
                        id={`perm-${permission.key}`}
                        checked={formData.customPermissions.includes(permission.key)}
                        onCheckedChange={(checked) =>
                          handlePermissionChange(permission.key, checked as boolean)
                        }
                      />
                      <Label
                        htmlFor={`perm-${permission.key}`}
                        className="text-sm font-normal cursor-pointer"
                      >
                        {permission.label}
                      </Label>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setIsAddModalOpen(false)}>
              Cancel
            </Button>
            <Button
              type="button"
              onClick={handleAddUser}
              disabled={!formData.name || !formData.email}
            >
              Add User
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit User Modal */}
      <Dialog open={!!editingUser} onOpenChange={() => setEditingUser(null)}>
        <DialogContent className="sm:max-w-[600px]">
          <DialogHeader>
            <DialogTitle>Edit User</DialogTitle>
            <DialogDescription>
              Update user information, role, and permissions.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="edit-user-name">Full Name</Label>
              <Input
                id="edit-user-name"
                value={formData.name}
                onChange={(e) => handleInputChange('name', e.target.value)}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="edit-user-email">Email Address</Label>
              <Input
                id="edit-user-email"
                type="email"
                value={formData.email}
                onChange={(e) => handleInputChange('email', e.target.value)}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="edit-user-role">Role</Label>
              <Select value={formData.role} onValueChange={(value) => handleInputChange('role', value)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {roles.map((role) => (
                    <SelectItem key={role.name} value={role.name}>
                      <div className="flex items-center gap-2">
                        {getRoleIcon(role.name)}
                        <span>{role.label}</span>
                      </div>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <p className="text-sm text-muted-foreground">
                {roles.find(r => r.name === formData.role)?.description}
              </p>
            </div>

            {formData.role !== 'admin' && (
              <div className="space-y-3">
                <Label>Custom Permissions</Label>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-3 max-h-48 overflow-y-auto">
                  {allPermissions.map((permission) => (
                    <div key={permission.key} className="flex items-center space-x-2">
                      <Checkbox
                        id={`edit-perm-${permission.key}`}
                        checked={formData.customPermissions.includes(permission.key)}
                        onCheckedChange={(checked) =>
                          handlePermissionChange(permission.key, checked as boolean)
                        }
                      />
                      <Label
                        htmlFor={`edit-perm-${permission.key}`}
                        className="text-sm font-normal cursor-pointer"
                      >
                        {permission.label}
                      </Label>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setEditingUser(null)}>
              Cancel
            </Button>
            <Button
              type="button"
              onClick={handleUpdateUser}
              disabled={!formData.name || !formData.email}
            >
              Update User
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Role Management Modal */}
      <Dialog open={showRoleModal} onOpenChange={setShowRoleModal}>
        <DialogContent className="sm:max-w-[700px]">
          <DialogHeader>
            <DialogTitle>Role Management</DialogTitle>
            <DialogDescription>
              View and understand the different user roles and their permissions.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-6">
            {roles.map((role) => (
              <Card key={role.name}>
                <CardHeader>
                  <div className="flex items-center gap-3">
                    {getRoleIcon(role.name)}
                    <div>
                      <CardTitle className="text-lg">{role.label}</CardTitle>
                      <p className="text-sm text-muted-foreground">{role.description}</p>
                    </div>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="space-y-2">
                    <Label className="text-sm font-medium">Permissions:</Label>
                    <div className="flex flex-wrap gap-2">
                      {role.permissions.includes('all') ? (
                        <Badge className="bg-primary/10 text-primary border-primary/20">
                          All Permissions
                        </Badge>
                      ) : (
                        role.permissions.map((permission) => {
                          const perm = allPermissions.find(p => p.key === permission);
                          return (
                            <Badge key={permission} variant="outline" className="text-xs">
                              {perm?.label || permission}
                            </Badge>
                          );
                        })
                      )}
                    </div>
                  </div>
                  <div className="mt-4 text-sm text-muted-foreground">
                    Users with this role: {users.filter(u => u.role === role.name).length}
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>

          <DialogFooter>
            <Button type="button" onClick={() => setShowRoleModal(false)}>
              Close
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
