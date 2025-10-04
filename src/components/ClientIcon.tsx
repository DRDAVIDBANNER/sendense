'use client';

import { useEffect, useState } from 'react';

interface ClientIconProps {
  children: React.ReactNode;
  className?: string;
}

// Client-side wrapper to prevent hydration issues with browser extensions
export function ClientIcon({ children, className }: ClientIconProps) {
  const [isClient, setIsClient] = useState(false);

  useEffect(() => {
    setIsClient(true);
  }, []);

  if (!isClient) {
    // Return a placeholder during SSR to match client
    return <div className={className} style={{ width: '1.25rem', height: '1.25rem' }} />;
  }

  return <div className={className}>{children}</div>;
}

