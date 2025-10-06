"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState } from "react";
import {
  BarChart3,
  Database,
  FileText,
  Home,
  Moon,
  Server,
  Settings,
  Shield,
  Sun,
  Users,
  Wrench
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

const menuItems = [
  { name: "Dashboard", href: "/dashboard", icon: Home },
  { name: "Protection Flows", href: "/protection-flows", icon: Shield },
  { name: "Protection Groups", href: "/protection-groups", icon: Database },
  { name: "Appliances", href: "/appliances", icon: Server },
  { name: "Report Center", href: "/report-center", icon: BarChart3 },
  { name: "Settings", href: "/settings/sources", icon: Settings },
  { name: "Users", href: "/users", icon: Users },
  { name: "Support", href: "/support", icon: Wrench },
];

export function Sidebar() {
  const pathname = usePathname();
  const [isDark, setIsDark] = useState(true);

  const toggleTheme = () => {
    setIsDark(!isDark);
    document.documentElement.classList.toggle("dark", !isDark);
  };

  return (
    <div className="w-64 bg-sidebar border-r border-sidebar-border flex flex-col h-screen">
      {/* Logo */}
      <div className="p-6 border-b border-sidebar-border">
        <h1 className="text-xl font-bold text-sidebar-foreground">
          Sendense
        </h1>
        <p className="text-sm text-sidebar-foreground/70">
          Universal Backup
        </p>
      </div>

      {/* Navigation */}
      <nav className="flex-1 p-4">
        <ul className="space-y-2">
          {menuItems.map((item) => {
            const Icon = item.icon;
            const isActive = pathname === item.href ||
              (item.href !== "/dashboard" && pathname.startsWith(item.href));

            return (
              <li key={item.name}>
                <Link href={item.href}>
                  <Button
                    variant={isActive ? "secondary" : "ghost"}
                    className={cn(
                      "w-full justify-start gap-3 h-10",
                      isActive && "bg-sidebar-accent text-sidebar-accent-foreground"
                    )}
                  >
                    <Icon className="h-4 w-4" />
                    {item.name}
                  </Button>
                </Link>
              </li>
            );
          })}
        </ul>
      </nav>

      {/* Theme Toggle */}
      <div className="p-4 border-t border-sidebar-border">
        <Button
          variant="ghost"
          onClick={toggleTheme}
          className="w-full justify-start gap-3 h-10"
        >
          {isDark ? (
            <>
              <Sun className="h-4 w-4" />
              Light Mode
            </>
          ) : (
            <>
              <Moon className="h-4 w-4" />
              Dark Mode
            </>
          )}
        </Button>
      </div>
    </div>
  );
}
