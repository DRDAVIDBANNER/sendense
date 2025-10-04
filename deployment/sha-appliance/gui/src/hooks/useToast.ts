// Simple toast notification hook
// Following our best practices: TypeScript, useCallback for performance

'use client';

import { useCallback } from 'react';

export type ToastType = 'success' | 'error' | 'warning' | 'info';

export interface ToastMessage {
  type: ToastType;
  title: string;
  message?: string;
  duration?: number;
}

// Simple toast implementation - can be enhanced with a toast library later
export function useToast() {
  const showToast = useCallback((toast: ToastMessage) => {
    // For now, using console and alert - will be replaced with proper toast UI
    const { type, title, message, duration = 5000 } = toast;
    
    console.log(`[${type.toUpperCase()}] ${title}${message ? `: ${message}` : ''}`);
    
    // Simple notification for critical errors
    if (type === 'error') {
      // In a real implementation, this would trigger a toast component
      console.error(`Error: ${title}${message ? ` - ${message}` : ''}`);
    }
  }, []);

  const success = useCallback((title: string, message?: string) => {
    showToast({ type: 'success', title, message });
  }, [showToast]);

  const error = useCallback((title: string, message?: string) => {
    showToast({ type: 'error', title, message });
  }, [showToast]);

  const warning = useCallback((title: string, message?: string) => {
    showToast({ type: 'warning', title, message });
  }, [showToast]);

  const info = useCallback((title: string, message?: string) => {
    showToast({ type: 'info', title, message });
  }, [showToast]);

  return {
    showToast,
    success,
    error,
    warning,
    info,
  };
}
