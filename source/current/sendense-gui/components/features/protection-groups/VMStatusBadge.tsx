"use client";

import { Badge } from "@/components/ui/badge";

interface VMStatusBadgeProps {
  status: 'discovered' | 'replicating' | 'ready_for_failover' | 'poweredOn' | 'poweredOff' | 'suspended';
  className?: string;
}

export function VMStatusBadge({ status, className = "" }: VMStatusBadgeProps) {
  const getStatusConfig = (status: string) => {
    switch (status) {
      case 'poweredOn':
      case 'running':
        return {
          label: 'Running',
          className: 'bg-green-500/10 text-green-400 border-green-500/20'
        };
      case 'poweredOff':
      case 'stopped':
        return {
          label: 'Stopped',
          className: 'bg-gray-500/10 text-gray-400 border-gray-500/20'
        };
      case 'suspended':
        return {
          label: 'Suspended',
          className: 'bg-yellow-500/10 text-yellow-400 border-yellow-500/20'
        };
      case 'discovered':
        return {
          label: 'Discovered',
          className: 'bg-blue-500/10 text-blue-400 border-blue-500/20'
        };
      case 'replicating':
        return {
          label: 'Replicating',
          className: 'bg-purple-500/10 text-purple-400 border-purple-500/20'
        };
      case 'ready_for_failover':
        return {
          label: 'Ready for Failover',
          className: 'bg-orange-500/10 text-orange-400 border-orange-500/20'
        };
      default:
        return {
          label: 'Unknown',
          className: 'bg-muted text-muted-foreground'
        };
    }
  };

  const config = getStatusConfig(status);

  return (
    <Badge variant="secondary" className={`${config.className} ${className}`}>
      {config.label}
    </Badge>
  );
}
