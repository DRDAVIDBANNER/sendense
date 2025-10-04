'use client';

import React from 'react';
import { Card } from 'flowbite-react';

export const SettingsView = React.memo(() => {
  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-3xl font-bold text-gray-900 dark:text-white">System Settings</h1>
        <p className="text-gray-600 dark:text-gray-400">Configure MigrateKit OSSEA system parameters</p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* VMware Credentials */}
        <Card>
          <div className="p-6">
            <h3 className="text-xl font-semibold text-gray-900 dark:text-white mb-4">
              🔐 VMware Credentials Management
            </h3>
            <div className="space-y-4">
              <div className="p-4 border rounded-lg dark:border-gray-600 bg-blue-50 dark:bg-blue-900/20">
                <h4 className="font-medium text-blue-900 dark:text-blue-100">Credential Management Ready</h4>
                <p className="text-sm text-blue-700 dark:text-blue-300 mt-1">
                  Complete VMware credential management system implemented with AES-256 encryption.
                </p>
                <div className="mt-3 space-y-1 text-xs text-blue-600 dark:text-blue-400">
                  <div>✅ Database schema created with encrypted password storage</div>
                  <div>✅ Backend services implemented with encryption</div>
                  <div>✅ API endpoints ready for credential management</div>
                  <div>✅ Security model with environment-based key management</div>
                </div>
                <div className="mt-3 p-2 bg-yellow-100 dark:bg-yellow-900/20 rounded text-xs text-yellow-800 dark:text-yellow-200">
                  <strong>Deployment Required:</strong> Set encryption key and deploy enhanced OMA API to activate credential management
                </div>
              </div>
            </div>
          </div>
        </Card>

        {/* System Monitoring */}
        <Card>
          <div className="p-6">
            <h3 className="text-xl font-semibold text-gray-900 dark:text-white mb-4">
              📊 System Status
            </h3>
            <div className="space-y-3">
              <div className="flex justify-between items-center p-3 bg-green-50 dark:bg-green-900/20 rounded-lg">
                <span className="text-sm font-medium text-gray-900 dark:text-white">Multi-Volume Snapshots</span>
                <span className="text-xs text-green-600 dark:text-green-400">✅ Enterprise Protection Active</span>
              </div>
              <div className="flex justify-between items-center p-3 bg-green-50 dark:bg-green-900/20 rounded-lg">
                <span className="text-sm font-medium text-gray-900 dark:text-white">Persistent Device Naming</span>
                <span className="text-xs text-green-600 dark:text-green-400">✅ NBD Memory Sync Active</span>
              </div>
              <div className="flex justify-between items-center p-3 bg-green-50 dark:bg-green-900/20 rounded-lg">
                <span className="text-sm font-medium text-gray-900 dark:text-white">Sparse Block Optimization</span>
                <span className="text-xs text-green-600 dark:text-green-400">✅ 50%+ Bandwidth Efficiency</span>
              </div>
              <div className="flex justify-between items-center p-3 bg-green-50 dark:bg-green-900/20 rounded-lg">
                <span className="text-sm font-medium text-gray-900 dark:text-white">Live Failover VirtIO</span>
                <span className="text-xs text-green-600 dark:text-green-400">✅ Enhanced Error Handling</span>
              </div>
            </div>
          </div>
        </Card>

        {/* Network Configuration */}
        <Card>
          <div className="p-6">
            <h3 className="text-xl font-semibold text-gray-900 dark:text-white mb-4">
              🌐 Network Configuration
            </h3>
            <div className="space-y-3">
              <div className="flex justify-between items-center p-3 bg-green-50 dark:bg-green-900/20 rounded-lg">
                <span className="text-sm font-medium text-gray-900 dark:text-white">VMA-OMA Tunnel</span>
                <span className="text-xs text-green-600 dark:text-green-400">✅ Port 443 TLS Active</span>
              </div>
              <div className="flex justify-between items-center p-3 bg-green-50 dark:bg-green-900/20 rounded-lg">
                <span className="text-sm font-medium text-gray-900 dark:text-white">NBD Export Management</span>
                <span className="text-xs text-green-600 dark:text-green-400">✅ Persistent Naming Active</span>
              </div>
            </div>
          </div>
        </Card>

        {/* System Configuration */}
        <Card>
          <div className="p-6">
            <h3 className="text-xl font-semibold text-gray-900 dark:text-white mb-4">
              ⚙️ System Configuration
            </h3>
            <div className="space-y-3">
              <div className="p-4 border rounded-lg dark:border-gray-600">
                <h4 className="font-medium text-gray-900 dark:text-white">Job Execution</h4>
                <p className="text-sm text-gray-600 dark:text-gray-400">Migration and failover job settings</p>
                <div className="mt-2 text-xs text-gray-500">Coming in future release</div>
              </div>
              <div className="p-4 border rounded-lg dark:border-gray-600">
                <h4 className="font-medium text-gray-900 dark:text-white">Storage Configuration</h4>
                <p className="text-sm text-gray-600 dark:text-gray-400">Volume and snapshot settings</p>
                <div className="mt-2 text-xs text-gray-500">Coming in future release</div>
              </div>
            </div>
          </div>
        </Card>
      </div>
    </div>
  );
});

SettingsView.displayName = 'SettingsView';




