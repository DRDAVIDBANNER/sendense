import { Badge } from "@/components/ui/badge";
import {
  CheckCircle,
  Clock,
  AlertTriangle,
  XCircle,
  PlayCircle
} from "lucide-react";
import { cn } from "@/lib/utils";

type StatusType = 'success' | 'warning' | 'error' | 'pending' | 'running';

interface StatusBadgeProps {
  status: StatusType;
  className?: string;
}

const statusConfig = {
  success: {
    label: 'Success',
    icon: CheckCircle,
    className: 'bg-green-500/10 text-green-400 border-green-500/20'
  },
  warning: {
    label: 'Warning',
    icon: AlertTriangle,
    className: 'bg-yellow-500/10 text-yellow-400 border-yellow-500/20'
  },
  error: {
    label: 'Error',
    icon: XCircle,
    className: 'bg-red-500/10 text-red-400 border-red-500/20'
  },
  pending: {
    label: 'Pending',
    icon: Clock,
    className: 'bg-gray-500/10 text-gray-400 border-gray-500/20'
  },
  running: {
    label: 'Running',
    icon: PlayCircle,
    className: 'bg-blue-500/10 text-blue-400 border-blue-500/20'
  }
};

export function StatusBadge({ status, className }: StatusBadgeProps) {
  const config = statusConfig[status];
  const Icon = config.icon;

  return (
    <Badge
      variant="outline"
      className={cn(
        'flex items-center gap-1.5 font-medium',
        config.className,
        className
      )}
    >
      <Icon className="h-3 w-3" />
      {config.label}
    </Badge>
  );
}
