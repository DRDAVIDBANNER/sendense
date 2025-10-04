'use client';

import React, { createContext, useContext, useState, useCallback } from 'react';

interface Notification {
  id: string;
  type: 'success' | 'error' | 'warning' | 'info';
  title: string;
  message?: string;
  duration?: number;
}

interface NotificationContextType {
  notifications: Notification[];
  addNotification: (notification: Omit<Notification, 'id'>) => void;
  removeNotification: (id: string) => void;
  success: (title: string, message?: string) => void;
  error: (title: string, message?: string) => void;
  warning: (title: string, message?: string) => void;
  info: (title: string, message?: string) => void;
}

const NotificationContext = createContext<NotificationContextType | null>(null);

export const useNotifications = () => {
  const context = useContext(NotificationContext);
  if (!context) {
    throw new Error('useNotifications must be used within NotificationProvider');
  }
  return context;
};

export interface NotificationProviderProps {
  children: React.ReactNode;
}

export const NotificationProvider = React.memo(({ children }: NotificationProviderProps) => {
  const [notifications, setNotifications] = useState<Notification[]>([]);

  const addNotification = useCallback((notification: Omit<Notification, 'id'>) => {
    const id = Math.random().toString(36).substring(2);
    const newNotification = { ...notification, id };
    
    setNotifications(prev => [...prev, newNotification]);
    
    // Auto-remove after duration
    const duration = notification.duration ?? 5000;
    if (duration > 0) {
      setTimeout(() => {
        removeNotification(id);
      }, duration);
    }
  }, []);

  const removeNotification = useCallback((id: string) => {
    setNotifications(prev => prev.filter(n => n.id !== id));
  }, []);

  const success = useCallback((title: string, message?: string) => {
    addNotification({ type: 'success', title, message });
  }, [addNotification]);

  const error = useCallback((title: string, message?: string) => {
    addNotification({ type: 'error', title, message, duration: 8000 });
  }, [addNotification]);

  const warning = useCallback((title: string, message?: string) => {
    addNotification({ type: 'warning', title, message, duration: 6000 });
  }, [addNotification]);

  const info = useCallback((title: string, message?: string) => {
    addNotification({ type: 'info', title, message });
  }, [addNotification]);

  const contextValue = React.useMemo(() => ({
    notifications,
    addNotification,
    removeNotification,
    success,
    error,
    warning,
    info
  }), [notifications, addNotification, removeNotification, success, error, warning, info]);

  return (
    <NotificationContext.Provider value={contextValue}>
      {children}
      <SimpleNotificationContainer notifications={notifications} onRemove={removeNotification} />
    </NotificationContext.Provider>
  );
});

NotificationProvider.displayName = 'NotificationProvider';

// Simplified notification container using only HTML/CSS, no Flowbite components
interface SimpleNotificationContainerProps {
  notifications: Notification[];
  onRemove: (id: string) => void;
}

const SimpleNotificationContainer = React.memo(({ notifications, onRemove }: SimpleNotificationContainerProps) => {
  const getIcon = (type: Notification['type']) => {
    switch (type) {
      case 'success':
        return '✅';
      case 'error':
        return '❌';
      case 'warning':
        return '⚠️';
      case 'info':
        return 'ℹ️';
    }
  };

  const getColors = (type: Notification['type']) => {
    switch (type) {
      case 'success':
        return 'bg-green-50 border-green-200 text-green-800 dark:bg-green-900 dark:border-green-700 dark:text-green-200';
      case 'error':
        return 'bg-red-50 border-red-200 text-red-800 dark:bg-red-900 dark:border-red-700 dark:text-red-200';
      case 'warning':
        return 'bg-yellow-50 border-yellow-200 text-yellow-800 dark:bg-yellow-900 dark:border-yellow-700 dark:text-yellow-200';
      case 'info':
        return 'bg-blue-50 border-blue-200 text-blue-800 dark:bg-blue-900 dark:border-blue-700 dark:text-blue-200';
    }
  };

  if (notifications.length === 0) return null;

  return (
    <div className="fixed top-4 right-4 z-50 space-y-2 max-w-sm">
      {notifications.map((notification) => {
        const icon = getIcon(notification.type);
        const colors = getColors(notification.type);
        
        return (
          <div
            key={notification.id}
            className={`p-4 border rounded-lg shadow-lg ${colors}`}
          >
            <div className="flex items-start">
              <div className="flex-shrink-0">
                <span className="text-xl">{icon}</span>
              </div>
              <div className="ml-3 flex-1">
                <div className="font-semibold text-sm">{notification.title}</div>
                {notification.message && (
                  <div className="text-sm opacity-90 mt-1">
                    {notification.message}
                  </div>
                )}
              </div>
              <button
                onClick={() => onRemove(notification.id)}
                className="ml-2 flex-shrink-0 text-lg opacity-70 hover:opacity-100 transition-opacity"
                aria-label="Close notification"
              >
                ×
              </button>
            </div>
          </div>
        );
      })}
    </div>
  );
});

SimpleNotificationContainer.displayName = 'SimpleNotificationContainer';