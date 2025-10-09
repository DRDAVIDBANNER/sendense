"use client";

import { Button } from "@/components/ui/button";
import { Home, ChevronRight } from "lucide-react";

interface BreadcrumbNavProps {
  path: string;
  onNavigate: (path: string) => void;
}

export function BreadcrumbNav({ path, onNavigate }: BreadcrumbNavProps) {
  const pathParts = path.split('/').filter(Boolean);

  const buildPath = (index: number) => {
    if (index === -1) return '/';
    return '/' + pathParts.slice(0, index + 1).join('/');
  };

  return (
    <nav className="flex items-center space-x-1 text-sm">
      <Button
        variant="ghost"
        size="sm"
        onClick={() => onNavigate('/')}
        className="h-8 px-2 text-muted-foreground hover:text-foreground"
      >
        <Home className="h-4 w-4" />
      </Button>

      {pathParts.map((part, index) => (
        <div key={index} className="flex items-center">
          <ChevronRight className="h-4 w-4 text-muted-foreground mx-1" />
          <Button
            variant="ghost"
            size="sm"
            onClick={() => onNavigate(buildPath(index))}
            className="h-8 px-2 text-muted-foreground hover:text-foreground"
          >
            {part}
          </Button>
        </div>
      ))}
    </nav>
  );
}
