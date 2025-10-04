'use client';

import React from 'react';
import Link from 'next/link';
import { 
  HiHome, 
  HiDocumentSearch, 
  HiServer, 
  HiClipboardList, 
  HiLightningBolt, 
  HiGlobeAlt, 
  HiDocumentText, 
  HiCog,
  HiMenuAlt3,
  HiX,
  HiClock,
  HiCollection,
  HiUsers
} from 'react-icons/hi';
import { ClientIcon } from '../ClientIcon';

export interface LeftNavigationProps {
  activeSection: string;
  onSectionChange: (section: string) => void;
  collapsed: boolean;
  onToggle: () => void;
}

interface NavigationItem {
  id: string;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
  href: string;
  description: string;
}

const navigationItems: NavigationItem[] = [
  {
    id: 'dashboard',
    label: 'Dashboard',
    icon: HiHome,
    href: '/dashboard',
    description: 'System overview and summaries'
  },
  {
    id: 'discovery',
    label: 'Discovery',
    icon: HiDocumentSearch,
    href: '/discovery',
    description: 'VM discovery from vCenter'
  },
  {
    id: 'virtual-machines',
    label: 'Virtual Machines',
    icon: HiServer,
    href: '/virtual-machines',
    description: 'VM management (primary)'
  },
  {
    id: 'replication-jobs',
    label: 'Replication Jobs',
    icon: HiClipboardList,
    href: '/replication-jobs',
    description: 'Job-centric view'
  },
  {
    id: 'failover',
    label: 'Failover',
    icon: HiLightningBolt,
    href: '/failover',
    description: 'Failover management'
  },
  {
    id: 'network-mapping',
    label: 'Network Mapping',
    icon: HiGlobeAlt,
    href: '/network-mapping',
    description: 'Network configuration'
  },
  {
    id: 'schedules',
    label: 'Schedules',
    icon: HiClock,
    href: '/schedules',
    description: 'Automated replication schedules'
  },
  {
    id: 'machine-groups',
    label: 'Machine Groups',
    icon: HiCollection,
    href: '/machine-groups',
    description: 'VM group management'
  },
  {
    id: 'vm-assignment',
    label: 'VM Assignment',
    icon: HiUsers,
    href: '/vm-assignment',
    description: 'Assign VMs to groups'
  },
  {
    id: 'logs',
    label: 'Logs',
    icon: HiDocumentText,
    href: '/logs',
    description: 'System logs and troubleshooting'
  },
  {
    id: 'monitoring',
    label: 'Real-Time Monitoring',
    icon: HiLightningBolt,
    href: '/monitoring',
    description: 'Live system and migration monitoring'
  },
  {
    id: 'settings',
    label: 'Settings',
    icon: HiCog,
    href: '/settings',
    description: 'Configuration'
  }
];

export const LeftNavigation = React.memo(({ 
  activeSection, 
  onSectionChange, 
  collapsed, 
  onToggle 
}: LeftNavigationProps) => {
  const isActive = React.useCallback((itemId: string) => activeSection === itemId, [activeSection]);

  const handleItemClick = React.useCallback((item: NavigationItem) => {
    onSectionChange(item.id);
  }, [onSectionChange]);

  return (
    <div className="h-full bg-white dark:bg-gray-800 border-r border-gray-200 dark:border-gray-700 shadow-sm">
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-700">
        {!collapsed && (
          <div className="text-center">
            <h1 className="text-xl font-bold text-gray-900 dark:text-white tracking-wide">
              OSSEA-Migrate
            </h1>
            <p className="text-sm text-gray-600 dark:text-gray-300 mt-1 font-medium">
              Migration Platform
            </p>
          </div>
        )}
        
        <button
          onClick={onToggle}
          className="p-2 rounded-lg text-gray-500 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-gray-700 transition-colors"
          aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
        >
          <ClientIcon className="w-5 h-5">
            {collapsed ? <HiMenuAlt3 /> : <HiX />}
          </ClientIcon>
        </button>
      </div>

      {/* Navigation Items */}
      <nav className="h-full px-3 py-4 overflow-y-auto">
        <ul className="space-y-1 font-medium">
          {navigationItems.map((item) => {
            const active = isActive(item.id);
            
            return (
              <li key={item.id}>
                <Link
                  href={item.href}
                  onClick={() => handleItemClick(item)}
                  className={`
                    flex items-center p-3 rounded-lg transition-colors group
                    ${active 
                      ? 'bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300' 
                      : 'text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-700'
                    }
                  `}
                  title={collapsed ? `${item.label} - ${item.description}` : undefined}
                >
                  <ClientIcon className={`
                    w-5 h-5 transition-colors
                    ${active 
                      ? 'text-blue-600 dark:text-blue-400' 
                      : 'text-gray-500 group-hover:text-gray-700 dark:text-gray-400 dark:group-hover:text-gray-300'
                    }
                  `}>
                    <item.icon />
                  </ClientIcon>
                  
                  {!collapsed && (
                    <div className="ml-3 flex-1">
                      <span className="text-sm font-medium">
                        {item.label}
                      </span>
                      <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">
                        {item.description}
                      </p>
                    </div>
                  )}
                </Link>
              </li>
            );
          })}
        </ul>
      </nav>
    </div>
  );
});

LeftNavigation.displayName = 'LeftNavigation';