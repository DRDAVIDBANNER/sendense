'use client';

import { Card } from 'flowbite-react';
import { VMCentricLayout } from '@/components/layout/VMCentricLayout';
import { VMwareCredentialsManager } from '@/components/settings/VMwareCredentialsManager';
import { UnifiedOSSEAConfiguration } from '@/components/settings/UnifiedOSSEAConfiguration';

export default function OSSEASettings() {
  return (
    <VMCentricLayout>
      <div className="p-4 bg-gray-50 dark:bg-gray-900 min-h-screen space-y-6">
        {/* Unified OSSEA Configuration - Single component for everything */}
        <UnifiedOSSEAConfiguration />

        {/* VMware Credentials Management */}
        <Card>
          <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
            🔐 VMware vCenter Credentials
          </h3>
          <p className="text-gray-600 dark:text-gray-300 mb-4">
            Manage VMware vCenter credentials for migration operations. Credentials are encrypted and stored securely.
          </p>
          <VMwareCredentialsManager />
        </Card>
      </div>
    </VMCentricLayout>
  );
}