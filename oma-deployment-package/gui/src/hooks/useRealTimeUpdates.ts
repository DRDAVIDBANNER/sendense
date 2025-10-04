import { useState, useEffect, useRef, useCallback } from 'react';

interface ActiveJob {
  id: string;
  vm_name: string;
  status: string;
  progress_percent: number;
  current_operation: string;
  bytes_transferred: number;
  total_bytes: number;
  throughput_mbps: number;
  updated_at: string;
}

interface SystemInfo {
  uptime: string;
  memory_info: string;
  timestamp: string;
  active_job_count: number;
  error?: string;
}

interface RealTimeData {
  active_jobs: ActiveJob[];
  system_info: SystemInfo;
}

interface UseRealTimeUpdatesReturn {
  data: RealTimeData | null;
  isConnected: boolean;
  error: string | null;
  connect: () => void;
  disconnect: () => void;
}

export function useRealTimeUpdates(): UseRealTimeUpdatesReturn {
  const [data, setData] = useState<RealTimeData | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const eventSourceRef = useRef<EventSource | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout>();

  const connect = useCallback(() => {
    if (eventSourceRef.current) {
      return; // Already connected
    }

    try {
      // Use Server-Sent Events for real-time updates
      const eventSource = new EventSource('/api/websocket');
      eventSourceRef.current = eventSource;

      eventSource.onopen = () => {
        console.log('🔗 Real-time connection established');
        setIsConnected(true);
        setError(null);
      };

      eventSource.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data);
          
          if (message.type === 'system_update') {
            setData(message.data);
          } else if (message.type === 'connection') {
            console.log('📡 Real-time monitoring:', message.message);
          }
        } catch (parseError) {
          console.error('Failed to parse real-time message:', parseError);
        }
      };

      eventSource.onerror = (event) => {
        console.error('❌ Real-time connection error:', event);
        setError('Connection lost. Attempting to reconnect...');
        setIsConnected(false);
        
        // Auto-reconnect after 5 seconds
        reconnectTimeoutRef.current = setTimeout(() => {
          console.log('🔄 Attempting to reconnect...');
          disconnect();
          connect();
        }, 5000);
      };

    } catch (err) {
      console.error('Failed to establish real-time connection:', err);
      setError('Failed to connect to real-time updates');
    }
  }, []);

  const disconnect = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
    }
    
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
    }
    
    setIsConnected(false);
    console.log('🔌 Real-time connection closed');
  }, []);

  // Auto-connect on mount
  useEffect(() => {
    connect();
    
    return () => {
      disconnect();
    };
  }, [connect, disconnect]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      disconnect();
    };
  }, [disconnect]);

  return {
    data,
    isConnected,
    error,
    connect,
    disconnect
  };
}

// Specific hook for just active job progress
export function useActiveJobProgress(): {
  activeJobs: ActiveJob[];
  isConnected: boolean;
  error: string | null;
} {
  const { data, isConnected, error } = useRealTimeUpdates();
  
  return {
    activeJobs: data?.active_jobs || [],
    isConnected,
    error
  };
}

// Specific hook for system health
export function useSystemHealth(): {
  systemInfo: SystemInfo | null;
  isConnected: boolean;
  error: string | null;
} {
  const { data, isConnected, error } = useRealTimeUpdates();
  
  return {
    systemInfo: data?.system_info || null,
    isConnected,
    error
  };
}
