'use client';

import Link from 'next/link';
import { HiHome, HiCog, HiDatabase, HiCloud, HiGlobe, HiChartBar, HiServer, HiLightningBolt, HiDocumentSearch, HiClock, HiCollection, HiArchive } from 'react-icons/hi';
import { ClientIcon } from './ClientIcon';

interface SidebarProps {
  currentPage: string;
}

export function AppSidebar({ currentPage }: SidebarProps) {
  const isActive = (page: string) => currentPage === page;
  
  return (
    <div className="h-full bg-gray-50 dark:bg-gray-800 border-r border-gray-200 dark:border-gray-700">
      <div className="h-full px-3 py-4 overflow-y-auto">
        <ul className="space-y-2 font-medium">
          <li>
            <Link
              href="/"
              className={`flex items-center p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 ${
                isActive('dashboard') ? 'bg-gray-100 dark:bg-gray-700' : ''
              }`}
            >
              <ClientIcon className="w-5 h-5 text-gray-500 dark:text-gray-400">
                <HiHome />
              </ClientIcon>
              <span className="ml-3">Dashboard</span>
            </Link>
          </li>
          <li>
            <div className="flex items-center p-2 text-gray-900 dark:text-white">
              <ClientIcon className="w-5 h-5 text-gray-500 dark:text-gray-400">
                <HiCloud />
              </ClientIcon>
              <span className="flex-1 ml-3 whitespace-nowrap font-semibold">OSSEA Config</span>
            </div>
            <ul className="pl-8 space-y-1">
              <li>
                <Link
                  href="/settings/ossea"
                  className={`flex items-center p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 ${
                    isActive('ossea-settings') ? 'bg-gray-100 dark:bg-gray-700' : ''
                  }`}
                >
                  <ClientIcon className="w-4 h-4 text-gray-500 dark:text-gray-400">
                    <HiDatabase />
                  </ClientIcon>
                  <span className="ml-3">Connection</span>
                </Link>
              </li>
            </ul>
          </li>
          <li>
            <Link
              href="/discovery"
              className={`flex items-center p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 ${
                isActive('discovery') ? 'bg-gray-100 dark:bg-gray-700' : ''
              }`}
            >
              <ClientIcon className="w-5 h-5 text-gray-500 dark:text-gray-400">
                <HiDocumentSearch />
              </ClientIcon>
              <span className="ml-3">Discovery</span>
            </Link>
          </li>
          <li>
            <Link
              href="/virtual-machines"
              className={`flex items-center p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 ${
                isActive('virtual-machines') ? 'bg-gray-100 dark:bg-gray-700' : ''
              }`}
            >
              <ClientIcon className="w-5 h-5 text-gray-500 dark:text-gray-400">
                <HiServer />
              </ClientIcon>
              <span className="ml-3">Virtual Machines</span>
            </Link>
          </li>
          <li>
            <Link
              href="/backups"
              className={`flex items-center p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 ${
                isActive('backups') ? 'bg-gray-100 dark:bg-gray-700' : ''
              }`}
            >
              <ClientIcon className="w-5 h-5 text-gray-500 dark:text-gray-400">
                <HiArchive />
              </ClientIcon>
              <span className="ml-3">Backups</span>
            </Link>
          </li>
          <li>
            <Link
              href="/network-mapping"
              className={`flex items-center p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 ${
                isActive('network-mapping') ? 'bg-gray-100 dark:bg-gray-700' : ''
              }`}
            >
              <ClientIcon className="w-5 h-5 text-gray-500 dark:text-gray-400">
                <HiGlobe />
              </ClientIcon>
              <span className="ml-3">Network Mapping</span>
            </Link>
          </li>
          <li>
            <Link
              href="/schedules"
              className={`flex items-center p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 ${
                isActive('schedules') ? 'bg-gray-100 dark:bg-gray-700' : ''
              }`}
            >
              <ClientIcon className="w-5 h-5 text-gray-500 dark:text-gray-400">
                <HiClock />
              </ClientIcon>
              <span className="ml-3">Schedules</span>
            </Link>
          </li>
          <li>
            <Link
              href="/machine-groups"
              className={`flex items-center p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 ${
                isActive('machine-groups') ? 'bg-gray-100 dark:bg-gray-700' : ''
              }`}
            >
              <ClientIcon className="w-5 h-5 text-gray-500 dark:text-gray-400">
                <HiCollection />
              </ClientIcon>
              <span className="ml-3">Machine Groups</span>
            </Link>
          </li>
          <li>
            <Link
              href="/analytics"
              className={`flex items-center p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 ${
                isActive('analytics') ? 'bg-gray-100 dark:bg-gray-700' : ''
              }`}
            >
              <ClientIcon className="w-5 h-5 text-gray-500 dark:text-gray-400">
                <HiChartBar />
              </ClientIcon>
              <span className="ml-3">Analytics</span>
            </Link>
          </li>
          <li>
            <Link
              href="/monitoring"
              className={`flex items-center p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 ${
                isActive('monitoring') ? 'bg-gray-100 dark:bg-gray-700' : ''
              }`}
            >
              <ClientIcon className="w-5 h-5 text-gray-500 dark:text-gray-400">
                <HiLightningBolt />
              </ClientIcon>
              <span className="ml-3">Real-Time Monitoring</span>
            </Link>
          </li>
          <li>
            <Link
              href="/settings"
              className={`flex items-center p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 ${
                isActive('settings') ? 'bg-gray-100 dark:bg-gray-700' : ''
              }`}
            >
              <ClientIcon className="w-5 h-5 text-gray-500 dark:text-gray-400">
                <HiCog />
              </ClientIcon>
              <span className="ml-3">Settings</span>
            </Link>
          </li>
        </ul>
      </div>
    </div>
  );
}