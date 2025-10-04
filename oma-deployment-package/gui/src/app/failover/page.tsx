'use client';

import { useState, useEffect } from 'react';
import { Card, Badge, Button, Alert, Modal, TextInput, Label, Select, ToggleSwitch } from 'flowbite-react';
import { 
  HiPlay, 
  HiStop, 
  HiRefresh, 
  HiExclamationCircle, 
  HiCheckCircle, 
  HiClock,
  HiLightningBolt,
  HiBeaker,
  HiTrash
} from 'react-icons/hi';

import { VMCentricLayout } from '@/components/layout/VMCentricLayout';
import { ClientOnly } from '@/components/ClientOnly';

// Interfaces for failover operations
interface VM {
  id: string;
  name: string;
  path: string;
  datacenter: string;
  power_state: string;
  guest_os: string;
  memory_mb: number;
  num_cpu: number;
  vmx_version?: string;
  disks?: DiskInfo[];
  networks?: NetworkInfo[];
}

interface DiskInfo {
  id: string;
  label: string;
  path: string;
  vmdk_path: string;
  size_gb: number;
  capacity_bytes: number;
  datastore: string;
  provisioning_type: string;
  unit_number: number;
}

interface NetworkInfo {
  label: string;
  network_name: string;
  adapter_type: string;
  mac_address: string;
  connected: boolean;
}

interface FailoverJob {
  job_id: string;
  vm_id: string;
  vm_name: string;
  job_type: string;
  status: string;
  progress: number;
  destination_vm_id?: string;
  snapshot_id?: string;
  created_at: string;
  started_at?: string;
  completed_at?: string;
  duration?: string;
  error_message?: string;
}

interface FailoverRequest {
  vm_id: string;
  vm_name: string;
  failover_type: 'live' | 'test';
  skip_validation?: boolean;
  test_duration?: string;
  auto_cleanup?: boolean;
  network_mappings?: Record<string, string>;
  custom_config?: Record<string, any>;
}

export default function FailoverManagement() {
  const [vms, setVMs] = useState<VM[]>([]);
  const [failoverJobs, setFailoverJobs] = useState<FailoverJob[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  
  // Modal states
  const [showFailoverModal, setShowFailoverModal] = useState(false);
  const [selectedVM, setSelectedVM] = useState<VM | null>(null);
  
  // Failover configuration
  const [failoverType, setFailoverType] = useState<'live' | 'test'>('test');
  const [skipValidation, setSkipValidation] = useState(false);
  const [testDuration, setTestDuration] = useState('2h');
  const [autoCleanup, setAutoCleanup] = useState(true);

  // Fetch data on component mount
  useEffect(() => {
    fetchFailoverJobs();
    fetchVMs();
    
    // Set up auto-refresh for job status
    const interval = setInterval(fetchFailoverJobs, 10000); // Refresh every 10 seconds
    return () => clearInterval(interval);
  }, []);

  const fetchVMs = async () => {
    try {
      // Use Enhanced Discovery API with default credential
      const response = await fetch('/api/discovery/discover-vms', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          credential_id: 2, // Default VMware credential
          create_context: false
        })
      });
      
      if (response.ok) {
        const data = await response.json();
        setVMs(data.discovered_vms || []);
      }
    } catch (err) {
      console.error('Failed to fetch VMs:', err);
    }
  };

  const fetchFailoverJobs = async () => {
    try {
      console.log('🔄 Fetching failover jobs...');
      const response = await fetch('/api/failover');
      if (response.ok) {
        const data = await response.json();
        console.log('✅ Failover jobs fetched:', data);
        setFailoverJobs(data.jobs || []);
      } else {
        console.warn('⚠️ Failed to fetch failover jobs:', response.status);
      }
    } catch (err) {
      console.error('❌ Error fetching failover jobs:', err);
    }
  };

  const initiateFailover = async (request: FailoverRequest) => {
    try {
      setLoading(true);
      setError('');
      setSuccess('');

      console.log('🚀 Initiating failover:', request);

      const response = await fetch('/api/failover', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(request)
      });

      const data = await response.json();

      if (response.ok && data.success) {
        // Show success message and close modal after brief delay
        setSuccess(`${request.failover_type === 'live' ? 'Live' : 'Test'} failover initiated successfully! Job ID: ${data.job_id}`);
        
        console.log('✅ Failover initiated:', {
          job_id: data.job_id,
          estimated_duration: data.estimated_duration
        });

        // Auto-close modal after 2 seconds to let user see the success message
        setTimeout(() => {
          setShowFailoverModal(false);
          fetchFailoverJobs(); // Refresh job list
        }, 2000);
        
      } else {
        setError(data.error || 'Failed to initiate failover');
        console.error('❌ Failover failed:', data);
      }
    } catch (err) {
      setError('Network error initiating failover');
      console.error('❌ Failover error:', err);
    } finally {
      setLoading(false);
    }
  };

  const endTestFailover = async (jobId: string) => {
    try {
      setLoading(true);
      console.log('🧹 Ending test failover:', jobId);

      const response = await fetch(`/api/failover/${jobId}`, {
        method: 'DELETE'
      });

      const data = await response.json();

      if (response.ok && data.success) {
        setSuccess('Test failover cleanup completed successfully!');
        fetchFailoverJobs(); // Refresh job list
        console.log('✅ Test failover cleanup completed');
      } else {
        setError(data.error || 'Failed to cleanup test failover');
        console.error('❌ Cleanup failed:', data);
      }
    } catch (err) {
      setError('Network error during cleanup');
      console.error('❌ Cleanup error:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleFailoverSubmit = () => {
    if (!selectedVM) return;

    const request: FailoverRequest = {
      vm_id: selectedVM.id,
      vm_name: selectedVM.name,
      failover_type: failoverType,
      skip_validation: skipValidation,
      ...(failoverType === 'test' && {
        test_duration: testDuration,
        auto_cleanup: autoCleanup
      })
    };

    initiateFailover(request);
  };

  const openFailoverModal = (vm: VM, type: 'live' | 'test') => {
    setSelectedVM(vm);
    setFailoverType(type);
    setShowFailoverModal(true);
  };

  const getStatusBadge = (status: string | null | undefined) => {
    if (!status) {
      return <Badge color="gray">Unknown</Badge>;
    }
    switch (status.toLowerCase()) {
      case 'completed':
        return <Badge color="success" icon={HiCheckCircle}>{status}</Badge>;
      case 'failed':
        return <Badge color="failure" icon={HiExclamationCircle}>{status}</Badge>;
      case 'in_progress':
      case 'pending':
        return <Badge color="warning" icon={HiClock}>{status}</Badge>;
      default:
        return <Badge color="gray">{status}</Badge>;
    }
  };

  const getProgressWidth = (progress: number) => {
    return Math.min(Math.max(progress, 0), 100);
  };

  return (
    <VMCentricLayout>
      <div className="p-4 bg-gray-50 dark:bg-gray-900 min-h-screen">
        <div className="mb-6">
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white mb-2">
            🔄 VM Failover Management
          </h1>
          <p className="text-gray-600 dark:text-gray-300">
            Live and Test VM Failover Operations to OSSEA
          </p>
        </div>

        {error && (
          <Alert color="failure" className="mb-4" onDismiss={() => setError('')}>
            <span>{error}</span>
          </Alert>
        )}

        {success && (
          <Alert color="success" className="mb-4" onDismiss={() => setSuccess('')}>
            <span>{success}</span>
          </Alert>
        )}

        {/* Status Cards */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
          <Card>
            <div className="flex items-center">
              <div className="p-3 rounded-full bg-blue-100 dark:bg-blue-900 mr-4">
                <ClientOnly>
                  <HiLightningBolt className="w-6 h-6 text-blue-600 dark:text-blue-300" />
                </ClientOnly>
              </div>
              <div>
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
                  Available VMs
                </h3>
                <p className="text-sm text-gray-600 dark:text-gray-300">
                  {vms.length} VMs ready
                </p>
              </div>
            </div>
          </Card>

          <Card>
            <div className="flex items-center">
              <div className="p-3 rounded-full bg-green-100 dark:bg-green-900 mr-4">
                <ClientOnly>
                  <HiPlay className="w-6 h-6 text-green-600 dark:text-green-300" />
                </ClientOnly>
              </div>
              <div>
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
                  Live Failovers
                </h3>
                <p className="text-sm text-gray-600 dark:text-gray-300">
                  {failoverJobs.filter(j => j.job_type === 'live').length} jobs
                </p>
              </div>
            </div>
          </Card>

          <Card>
            <div className="flex items-center">
              <div className="p-3 rounded-full bg-purple-100 dark:bg-purple-900 mr-4">
                <ClientOnly>
                  <HiBeaker className="w-6 h-6 text-purple-600 dark:text-purple-300" />
                </ClientOnly>
              </div>
              <div>
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
                  Test Failovers
                </h3>
                <p className="text-sm text-gray-600 dark:text-gray-300">
                  {failoverJobs.filter(j => j.job_type === 'test').length} jobs
                </p>
              </div>
            </div>
          </Card>

          <Card>
            <div className="flex items-center">
              <div className="p-3 rounded-full bg-orange-100 dark:bg-orange-900 mr-4">
                <ClientOnly>
                  <HiClock className="w-6 h-6 text-orange-600 dark:text-orange-300" />
                </ClientOnly>
              </div>
              <div>
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
                  Active Jobs
                </h3>
                <p className="text-sm text-gray-600 dark:text-gray-300">
                  {failoverJobs.filter(j => j.status === 'in_progress' || j.status === 'pending').length} running
                </p>
              </div>
            </div>
          </Card>
        </div>

        {/* Available VMs for Failover */}
        {vms.length > 0 && (
          <Card className="mb-6">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-xl font-bold text-gray-900 dark:text-white">
                Available VMs for Failover
              </h2>
              <Button onClick={fetchVMs} color="gray" size="sm">
                <ClientOnly>
                  <HiRefresh className="mr-2 h-4 w-4" />
                </ClientOnly>
                Refresh
              </Button>
            </div>
            <div className="overflow-x-auto">
              <table className="w-full text-sm text-left text-gray-500 dark:text-gray-400">
                <thead className="text-xs text-gray-700 uppercase bg-gray-50 dark:bg-gray-700 dark:text-gray-400">
                  <tr>
                    <th scope="col" className="px-6 py-3">VM Name</th>
                    <th scope="col" className="px-6 py-3">Power State</th>
                    <th scope="col" className="px-6 py-3">OS Type</th>
                    <th scope="col" className="px-6 py-3">Resources</th>
                    <th scope="col" className="px-6 py-3">Failover Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {vms.map((vm) => (
                    <tr key={vm.id} className="bg-white border-b dark:bg-gray-800 dark:border-gray-700">
                      <td className="px-6 py-4 font-medium text-gray-900 whitespace-nowrap dark:text-white">
                        {vm.name}
                      </td>
                      <td className="px-6 py-4">
                        <Badge color={vm.power_state === 'poweredOn' ? 'success' : 'gray'}>
                          {vm.power_state}
                        </Badge>
                      </td>
                      <td className="px-6 py-4">{vm.guest_os}</td>
                      <td className="px-6 py-4">
                        {vm.num_cpu} CPU / {Math.round(vm.memory_mb / 1024)} GB RAM
                      </td>
                      <td className="px-6 py-4">
                        <div className="flex space-x-2">
                          <Button
                            size="xs"
                            color="success"
                            onClick={() => openFailoverModal(vm, 'live')}
                            disabled={loading}
                          >
                            <ClientOnly>
                              <HiLightningBolt className="mr-1 h-3 w-3" />
                            </ClientOnly>
                            Live Failover
                          </Button>
                          <Button
                            size="xs"
                            color="purple"
                            onClick={() => openFailoverModal(vm, 'test')}
                            disabled={loading}
                          >
                            <ClientOnly>
                              <HiBeaker className="mr-1 h-3 w-3" />
                            </ClientOnly>
                            Test Failover
                          </Button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </Card>
        )}

        {/* Active Failover Jobs */}
        {failoverJobs.length > 0 && (
          <Card>
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-xl font-bold text-gray-900 dark:text-white">
                Failover Jobs
              </h2>
              <Button onClick={fetchFailoverJobs} color="gray" size="sm">
                <ClientOnly>
                  <HiRefresh className="mr-2 h-4 w-4" />
                </ClientOnly>
                Refresh
              </Button>
            </div>
            <div className="overflow-x-auto">
              <table className="w-full text-sm text-left text-gray-500 dark:text-gray-400">
                <thead className="text-xs text-gray-700 uppercase bg-gray-50 dark:bg-gray-700 dark:text-gray-400">
                  <tr>
                    <th scope="col" className="px-6 py-3">Job ID</th>
                    <th scope="col" className="px-6 py-3">VM Name</th>
                    <th scope="col" className="px-6 py-3">Type</th>
                    <th scope="col" className="px-6 py-3">Status</th>
                    <th scope="col" className="px-6 py-3">Progress</th>
                    <th scope="col" className="px-6 py-3">Duration</th>
                    <th scope="col" className="px-6 py-3">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {failoverJobs.map((job, index) => (
                    <tr key={`${job.job_id}-${index}`} className="bg-white border-b dark:bg-gray-800 dark:border-gray-700">
                      <td className="px-6 py-4 font-mono text-xs">
                        {job.job_id.substring(0, 16)}...
                      </td>
                      <td className="px-6 py-4 font-medium">
                        {job.vm_name}
                      </td>
                      <td className="px-6 py-4">
                        <Badge color={job.job_type === 'live' ? 'success' : 'purple'}>
                          {job.job_type}
                        </Badge>
                      </td>
                      <td className="px-6 py-4">
                        {getStatusBadge(job.status)}
                      </td>
                      <td className="px-6 py-4">
                        <div className="w-full bg-gray-200 rounded-full h-2 dark:bg-gray-700">
                          <div 
                            className="bg-blue-600 h-2 rounded-full transition-all duration-300" 
                            style={{ width: `${getProgressWidth(job.progress)}%` }}
                          ></div>
                        </div>
                        <span className="text-xs text-gray-500">{Math.round(job.progress)}%</span>
                      </td>
                      <td className="px-6 py-4 text-xs">
                        {job.duration || 'N/A'}
                      </td>
                      <td className="px-6 py-4">
                        {job.job_type === 'test' && job.status === 'completed' && (
                          <Button
                            size="xs"
                            color="failure"
                            onClick={() => endTestFailover(job.job_id)}
                            disabled={loading}
                          >
                            <ClientOnly>
                              <HiTrash className="mr-1 h-3 w-3" />
                            </ClientOnly>
                            Cleanup
                          </Button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </Card>
        )}

        {/* Failover Configuration Modal */}
        <Modal show={showFailoverModal} onClose={() => !loading && setShowFailoverModal(false)} size="md">
          <Modal.Header>
            Configure {failoverType === 'live' ? 'Live' : 'Test'} Failover
          </Modal.Header>
          <Modal.Body>
            {/* Show success/error messages in modal */}
            {success && (
              <Alert color="success" className="mb-4">
                <span>{success}</span>
              </Alert>
            )}
            
            {error && (
              <Alert color="failure" className="mb-4">
                <span>{error}</span>
              </Alert>
            )}

            {selectedVM && (
              <div className="space-y-4">
                <div>
                  <Label htmlFor="vm-name">VM Name</Label>
                  <TextInput
                    id="vm-name"
                    value={selectedVM.name}
                    disabled
                    className="mt-1"
                  />
                </div>

                <div>
                  <Label htmlFor="failover-type">Failover Type</Label>
                  <Select
                    id="failover-type"
                    value={failoverType}
                    onChange={(e) => setFailoverType(e.target.value as 'live' | 'test')}
                    className="mt-1"
                  >
                    <option value="test">Test Failover (Safe, Reversible)</option>
                    <option value="live">Live Failover (Production)</option>
                  </Select>
                </div>

                {failoverType === 'test' && (
                  <>
                    <div>
                      <Label htmlFor="test-duration">Test Duration</Label>
                      <Select
                        id="test-duration"
                        value={testDuration}
                        onChange={(e) => setTestDuration(e.target.value)}
                        className="mt-1"
                      >
                        <option value="30m">30 minutes</option>
                        <option value="1h">1 hour</option>
                        <option value="2h">2 hours</option>
                        <option value="4h">4 hours</option>
                        <option value="8h">8 hours</option>
                      </Select>
                    </div>

                    <div className="flex items-center">
                      <ToggleSwitch
                        checked={autoCleanup}
                        onChange={setAutoCleanup}
                        label="Auto-cleanup after test duration"
                      />
                    </div>
                  </>
                )}

                <div className="flex items-center">
                  <ToggleSwitch
                    checked={skipValidation}
                    onChange={setSkipValidation}
                    label="Skip pre-failover validation (Advanced)"
                  />
                </div>

                {failoverType === 'live' && (
                  <Alert color="warning">
                    <HiExclamationCircle className="h-4 w-4" />
                    <span className="ml-2">
                      <strong>Warning:</strong> Live failover will permanently move the VM to OSSEA. 
                      This operation cannot be easily reversed.
                    </span>
                  </Alert>
                )}
              </div>
            )}
          </Modal.Body>
          <Modal.Footer>
            <Button 
              onClick={handleFailoverSubmit}
              disabled={loading}
              color={failoverType === 'live' ? 'failure' : 'purple'}
            >
              {loading ? (
                <>
                  <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                  </svg>
                  Initiating...
                </>
              ) : (
                `Start ${failoverType === 'live' ? 'Live' : 'Test'} Failover`
              )}
            </Button>
            <Button 
              color="gray" 
              onClick={() => setShowFailoverModal(false)}
              disabled={loading}
            >
              Cancel
            </Button>
          </Modal.Footer>
        </Modal>
      </div>
    </VMCentricLayout>
  );
}




