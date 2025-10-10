import { format } from 'date-fns';

// Format bytes (supports up to PB)
export const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 B';
  if (bytes < 0) return 'N/A';

  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));

  return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`;
};

// Format duration (seconds to human-readable)
export const formatDuration = (seconds: number): string => {
  if (seconds < 0 || isNaN(seconds)) return 'N/A';
  if (seconds < 60) return `${Math.floor(seconds)}s`;

  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);

  if (hours > 0) return `${hours}h ${minutes}m`;
  return `${minutes}m`;
};

// Calculate duration from timestamps
export const getDuration = (startedAt: string | null, completedAt: string | null): number => {
  if (!startedAt || !completedAt) return 0;

  const start = new Date(startedAt).getTime();
  const end = new Date(completedAt).getTime();

  if (isNaN(start) || isNaN(end)) return 0;

  return (end - start) / 1000; // Convert to seconds
};

// Format timestamp for display
export const formatTimestamp = (timestamp: string): string => {
  try {
    return format(new Date(timestamp), 'MMM dd, HH:mm');
  } catch {
    return 'Invalid date';
  }
};
