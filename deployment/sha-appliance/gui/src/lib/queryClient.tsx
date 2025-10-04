// React Query client configuration
// Following our best practices: proper error handling, performance optimization

'use client';

import React from 'react';
import { QueryClient, QueryClientProvider, Query } from '@tanstack/react-query';

// Create query client with optimized settings
const createQueryClient = () => {
  return new QueryClient({
    defaultOptions: {
      queries: {
        // Stale time - how long data stays fresh
        staleTime: 5 * 60 * 1000, // 5 minutes
        // Cache time - how long data stays in cache after component unmounts
        gcTime: 10 * 60 * 1000, // 10 minutes (renamed from cacheTime)
        // Retry configuration
        retry: (failureCount, error) => {
          // Don't retry on 404s or client errors
          if (error instanceof Error && error.message.includes('404')) {
            return false;
          }
          // Retry up to 3 times for other errors
          return failureCount < 3;
        },
        // Global error suppression for job transition periods
        throwOnError: (error, query) => {
          // Check if this is a job progress query during transition period
          if (query.queryKey[0] === 'jobProgress') {
            // Let the specific useJobProgress hook handle grace period logic
            return true;
          }
          
          // For other queries, use default error handling
          return true;
        },
        retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30000),
        // Refetch settings
        refetchOnWindowFocus: false, // Don't refetch on window focus
        refetchOnReconnect: true, // Refetch when network reconnects
        refetchOnMount: true, // Refetch when component mounts
      },
      mutations: {
        // Retry mutations once
        retry: 1,
        retryDelay: 1000,
      },
    },
  });
};

// Client-side query client
let browserQueryClient: QueryClient | undefined = undefined;

function getQueryClient() {
  if (typeof window === 'undefined') {
    // Server: always make a new query client
    return createQueryClient();
  } else {
    // Browser: make a new query client if we don't already have one
    if (!browserQueryClient) browserQueryClient = createQueryClient();
    return browserQueryClient;
  }
}

interface QueryProviderProps {
  children: React.ReactNode;
}

export function QueryProvider({ children }: QueryProviderProps) {
  const queryClient = getQueryClient();

  return (
    <QueryClientProvider client={queryClient}>
      {children}
    </QueryClientProvider>
  );
}
