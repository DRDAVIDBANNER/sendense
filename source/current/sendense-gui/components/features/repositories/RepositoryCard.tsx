"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import { Button } from "@/components/ui/button";
import {
  Database,
  Server,
  HardDrive,
  Cloud,
  FolderOpen,
  MoreVertical,
  CheckCircle,
  XCircle,
  AlertTriangle,
  Clock,
  Trash2,
  Edit
} from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

export interface RepositoryCapacity {
  total: number; // Total capacity in bytes
  used: number; // Used capacity in bytes
  available: number; // Available capacity in bytes
  unit: 'B' | 'KB' | 'MB' | 'GB' | 'TB';
}

export interface Repository {
  id: string;
  name: string;
  type: 'local' | 's3' | 'nfs' | 'cifs' | 'azure';
  status: 'online' | 'offline' | 'warning';
  capacity: RepositoryCapacity;
  description?: string;
  lastTested?: string;
  location?: string;
}

interface RepositoryCardProps {
  repository: Repository;
  onEdit?: (repository: Repository) => void;
  onDelete?: (repository: Repository) => void;
  onTest?: (repository: Repository) => void;
}

export function RepositoryCard({ repository, onEdit, onDelete, onTest }: RepositoryCardProps) {
  const getTypeIcon = (type: string) => {
    switch (type) {
      case 'local': return <HardDrive className="h-4 w-4" />;
      case 's3': return <Cloud className="h-4 w-4" />;
      case 'nfs': return <Server className="h-4 w-4" />;
      case 'cifs': return <FolderOpen className="h-4 w-4" />;
      case 'azure': return <Cloud className="h-4 w-4" />;
      default: return <Database className="h-4 w-4" />;
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'online': return <CheckCircle className="h-4 w-4 text-green-500" />;
      case 'offline': return <XCircle className="h-4 w-4 text-red-500" />;
      case 'warning': return <AlertTriangle className="h-4 w-4 text-yellow-500" />;
      default: return <Clock className="h-4 w-4 text-gray-500" />;
    }
  };

  const getStatusBadge = (status: string) => {
    const variants = {
      online: 'bg-green-500/10 text-green-400 border-green-500/20',
      offline: 'bg-red-500/10 text-red-400 border-red-500/20',
      warning: 'bg-yellow-500/10 text-yellow-400 border-yellow-500/20'
    };

    return (
      <Badge className={variants[status as keyof typeof variants] || variants.offline}>
        {status.charAt(0).toUpperCase() + status.slice(1)}
      </Badge>
    );
  };

  const formatCapacity = (bytes: number, unit: string) => {
    return `${bytes} ${unit}`;
  };

  const getUsagePercentage = () => {
    const { total, used } = repository.capacity;
    if (total === 0) return 0;
    return Math.round((used / total) * 100);
  };

  const getUsageColor = () => {
    const percentage = getUsagePercentage();
    if (percentage >= 90) return 'bg-red-500';
    if (percentage >= 75) return 'bg-yellow-500';
    return 'bg-green-500';
  };

  return (
    <Card className="relative hover:shadow-md transition-shadow">
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="text-base flex items-center gap-2">
            {getTypeIcon(repository.type)}
            {repository.name}
          </CardTitle>
          <div className="flex items-center gap-2">
            {getStatusBadge(repository.status)}
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
                  <MoreVertical className="h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => onTest?.(repository)}>
                  Test Connection
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => onEdit?.(repository)}>
                  <Edit className="h-4 w-4 mr-2" />
                  Edit
                </DropdownMenuItem>
                <DropdownMenuItem
                  onClick={() => onDelete?.(repository)}
                  className="text-red-600 focus:text-red-600"
                >
                  <Trash2 className="h-4 w-4 mr-2" />
                  Delete
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
        {repository.description && (
          <p className="text-sm text-muted-foreground">{repository.description}</p>
        )}
      </CardHeader>

      <CardContent className="space-y-4">
        <div className="flex items-center justify-between text-sm">
          <span className="text-muted-foreground">Type:</span>
          <Badge variant="outline" className="capitalize">
            {repository.type.toUpperCase()}
          </Badge>
        </div>

        {repository.location && (
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">Location:</span>
            <span className="font-mono text-xs">{repository.location}</span>
          </div>
        )}

        <div className="space-y-2">
          <div className="flex justify-between text-sm">
            <span>Storage Usage</span>
            <span>{getUsagePercentage()}%</span>
          </div>
          <Progress
            value={getUsagePercentage()}
            className="h-2"
          />
          <div className="flex justify-between text-xs text-muted-foreground">
            <span>Used: {formatCapacity(repository.capacity.used, repository.capacity.unit)}</span>
            <span>Total: {formatCapacity(repository.capacity.total, repository.capacity.unit)}</span>
          </div>
        </div>

        {repository.lastTested && (
          <div className="flex items-center justify-between text-xs text-muted-foreground">
            <span>Last tested:</span>
            <span>{new Date(repository.lastTested).toLocaleString()}</span>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
