'use client';

import React, { useState, useEffect } from 'react';
import { Card, Label, TextInput, Button, Alert, Spinner, Select, Badge } from 'flowbite-react';
import { HiCheckCircle, HiXCircle, HiExclamation, HiRefresh, HiServer, HiCloudDownload } from 'react-icons/hi';
import { api } from '@/lib/api';

interface ValidationCheck {
  status: 'pass' | 'warning' | 'fail' | 'skipped';
  message: string;
  details?: any;
}

interface ValidationResult {
  oma_vm_detection: ValidationCheck;
  compute_offering: ValidationCheck;
  account_match: ValidationCheck;
  network_selection: ValidationCheck;
  overall_status: 'pass' | 'warning' | 'fail';
}

interface OMAVMInfo {
  vm_id: string;
  vm_name: string;
  mac_address: string;
  ip_address: string;
  account: string;
}

interface NetworkInfo {
  id: string;
  name: string;
  zone_id: string;
  zone_name: string;
  state: string;
}

export const CloudStackValidation: React.FC = () => {
  // Form state
  const [apiUrl, setApiUrl] = useState('');
  const [apiKey, setApiKey] = useState('');
  const [secretKey, setSecretKey] = useState('');
  const [omaVmId, setOmaVmId] = useState('');
  const [serviceOfferingId, setServiceOfferingId] = useState('');
  const [networkId, setNetworkId] = useState('');

  // UI state
  const [testing, setTesting] = useState(false);
  const [detecting, setDetecting] = useState(false);
  const [loadingNetworks, setLoadingNetworks] = useState(false);
  const [validating, setValidating] = useState(false);
  
  // Data state
  const [connectionSuccess, setConnectionSuccess] = useState(false);
  const [omaInfo, setOmaInfo] = useState<OMAVMInfo | null>(null);
  const [networks, setNetworks] = useState<NetworkInfo[]>([]);
  const [validationResult, setValidationResult] = useState<ValidationResult | null>(null);
  
  // Message state
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  // Load existing configuration on mount
  useEffect(() => {
    loadExistingConfig();
  }, []);

  const loadExistingConfig = async () => {
    try {
      // Load from database via API
      const response = await fetch('/api/settings/ossea');
      if (response.ok) {
        const data = await response.json();
        if (data) {
          setApiUrl(data.api_url || '');
          // Note: API keys are masked in the response for security
          // User needs to re-enter if they want to test/change
          setApiKey(data.api_key || '');
          setSecretKey(''); // Always require re-entry for security
          setOmaVmId(data.oma_vm_id || '');
          setServiceOfferingId(data.service_offering_id || '');
          setNetworkId(data.network_id || '');
        }
      }
    } catch (err) {
      console.error('Failed to load existing config:', err);
    }
  };

  const handleTestConnection = async () => {
    setError(null);
    setSuccess(null);
    setTesting(true);
    setConnectionSuccess(false);

    try {
      const result = await api.testCloudStackConnection({
        api_url: apiUrl,
        api_key: apiKey,
        secret_key: secretKey
      });

      if (result.success) {
        setSuccess(result.message);
        setConnectionSuccess(true);
      } else {
        setError(result.error || result.message);
        setConnectionSuccess(false);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to test connection');
      setConnectionSuccess(false);
    } finally {
      setTesting(false);
    }
  };

  const handleDetectOMAVM = async () => {
    setError(null);
    setSuccess(null);
    setDetecting(true);

    try {
      const result = await api.detectOMAVM({
        api_url: apiUrl,
        api_key: apiKey,
        secret_key: secretKey
      });

      if (result.success && result.oma_info) {
        setOmaInfo(result.oma_info);
        setOmaVmId(result.oma_info.vm_id);
        setSuccess(result.message);
      } else {
        setError(result.error || result.message);
        setOmaInfo(null);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to detect OMA VM');
      setOmaInfo(null);
    } finally {
      setDetecting(false);
    }
  };

  const handleLoadNetworks = async () => {
    setError(null);
    setLoadingNetworks(true);

    try {
      const result = await api.getCloudStackNetworks();

      if (result.success) {
        setNetworks(result.networks);
        if (result.networks.length > 0) {
          setSuccess(`Found ${result.count} network(s)`);
        } else {
          setError('No networks found. Please check your CloudStack configuration.');
        }
      } else {
        setError(result.error || 'Failed to load networks');
        setNetworks([]);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load networks');
      setNetworks([]);
    } finally {
      setLoadingNetworks(false);
    }
  };

  const handleValidate = async () => {
    setError(null);
    setSuccess(null);
    setValidating(true);

    try {
      const result = await api.validateCloudStackSettings({
        api_url: apiUrl,
        api_key: apiKey,
        secret_key: secretKey,
        oma_vm_id: omaVmId || undefined,
        service_offering_id: serviceOfferingId || undefined,
        network_id: networkId || undefined
      });

      if (result.result) {
        setValidationResult(result.result);
        
        if (result.result.overall_status === 'pass') {
          setSuccess(result.message);
        } else if (result.result.overall_status === 'warning') {
          setError(result.message);
        } else {
          setError(result.message);
        }
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to validate settings');
    } finally {
      setValidating(false);
    }
  };

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'pass':
        return <Badge color="success" icon={HiCheckCircle}>Pass</Badge>;
      case 'warning':
        return <Badge color="warning" icon={HiExclamation}>Warning</Badge>;
      case 'fail':
        return <Badge color="failure" icon={HiXCircle}>Fail</Badge>;
      case 'skipped':
        return <Badge color="gray">Skipped</Badge>;
      default:
        return <Badge color="gray">{status}</Badge>;
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'pass':
        return <HiCheckCircle className="h-5 w-5 text-green-500" />;
      case 'warning':
        return <HiExclamation className="h-5 w-5 text-yellow-500" />;
      case 'fail':
        return <HiXCircle className="h-5 w-5 text-red-500" />;
      default:
        return <HiExclamation className="h-5 w-5 text-gray-400" />;
    }
  };

  return (
    <div className="space-y-6">
      {/* Messages */}
      {error && (
        <Alert color="failure" onDismiss={() => setError(null)}>
          <span className="font-medium">Error:</span> {error}
        </Alert>
      )}
      
      {success && (
        <Alert color="success" onDismiss={() => setSuccess(null)}>
          <span className="font-medium">Success:</span> {success}
        </Alert>
      )}

      {/* Connection Test Section */}
      <Card>
        <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
          CloudStack Connection
        </h3>
        
        <div className="space-y-4">
          <div>
            <Label htmlFor="apiUrl" value="CloudStack API URL" />
            <TextInput
              id="apiUrl"
              type="text"
              placeholder="http://cloudstack.example.com:8080/client/api"
              value={apiUrl}
              onChange={(e) => setApiUrl(e.target.value)}
              disabled={testing}
            />
          </div>

          <div>
            <Label htmlFor="apiKey" value="API Key" />
            <TextInput
              id="apiKey"
              type="text"
              placeholder="Your CloudStack API Key"
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              disabled={testing}
            />
          </div>

          <div>
            <Label htmlFor="secretKey" value="Secret Key" />
            <TextInput
              id="secretKey"
              type="password"
              placeholder="Your CloudStack Secret Key"
              value={secretKey}
              onChange={(e) => setSecretKey(e.target.value)}
              disabled={testing}
            />
          </div>

          <Button
            onClick={handleTestConnection}
            disabled={testing || !apiUrl || !apiKey || !secretKey}
            color={connectionSuccess ? "success" : "blue"}
          >
            {testing ? (
              <>
                <Spinner size="sm" className="mr-2" />
                Testing Connection...
              </>
            ) : connectionSuccess ? (
              <>
                <HiCheckCircle className="mr-2 h-5 w-5" />
                Connected
              </>
            ) : (
              <>
                <HiServer className="mr-2 h-5 w-5" />
                Test Connection
              </>
            )}
          </Button>
        </div>
      </Card>

      {/* OMA VM Detection Section */}
      {connectionSuccess && (
        <Card>
          <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
            OMA VM Detection
          </h3>
          
          <div className="space-y-4">
            <Button
              onClick={handleDetectOMAVM}
              disabled={detecting}
              color="blue"
            >
              {detecting ? (
                <>
                  <Spinner size="sm" className="mr-2" />
                  Detecting...
                </>
              ) : (
                <>
                  <HiRefresh className="mr-2 h-5 w-5" />
                  Auto-Detect OMA VM
                </>
              )}
            </Button>

            {omaInfo && (
              <div className="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg p-4">
                <div className="flex items-start">
                  <HiCheckCircle className="h-5 w-5 text-green-500 mt-0.5 mr-3" />
                  <div className="flex-1">
                    <h4 className="font-medium text-green-900 dark:text-green-100">
                      OMA VM Detected
                    </h4>
                    <div className="mt-2 text-sm text-green-800 dark:text-green-200 space-y-1">
                      <p><span className="font-medium">VM Name:</span> {omaInfo.vm_name}</p>
                      <p><span className="font-medium">VM ID:</span> {omaInfo.vm_id}</p>
                      <p><span className="font-medium">MAC Address:</span> {omaInfo.mac_address}</p>
                      <p><span className="font-medium">IP Address:</span> {omaInfo.ip_address}</p>
                      <p><span className="font-medium">Account:</span> {omaInfo.account}</p>
                    </div>
                  </div>
                </div>
              </div>
            )}

            <div>
              <Label htmlFor="omaVmId" value="OMA VM ID" />
              <TextInput
                id="omaVmId"
                type="text"
                placeholder="Enter OMA VM ID manually if auto-detect fails"
                value={omaVmId}
                onChange={(e) => setOmaVmId(e.target.value)}
              />
              <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {omaInfo ? '✓ Auto-detected' : 'Manual entry or auto-detect above'}
              </p>
            </div>
          </div>
        </Card>
      )}

      {/* Network Selection Section */}
      {connectionSuccess && (
        <Card>
          <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
            Network Selection
          </h3>
          
          <div className="space-y-4">
            <Button
              onClick={handleLoadNetworks}
              disabled={loadingNetworks}
              color="blue"
            >
              {loadingNetworks ? (
                <>
                  <Spinner size="sm" className="mr-2" />
                  Loading Networks...
                </>
              ) : (
                <>
                  <HiCloudDownload className="mr-2 h-5 w-5" />
                  Load Available Networks
                </>
              )}
            </Button>

            {networks.length > 0 && (
              <div>
                <Label htmlFor="networkId" value="Select Network" />
                <Select
                  id="networkId"
                  value={networkId}
                  onChange={(e) => setNetworkId(e.target.value)}
                  required
                >
                  <option value="">Choose a network...</option>
                  {networks.map((network) => (
                    <option key={network.id} value={network.id}>
                      {network.name} ({network.zone_name} - {network.state})
                    </option>
                  ))}
                </Select>
                <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                  Found {networks.length} network(s)
                </p>
              </div>
            )}
          </div>
        </Card>
      )}

      {/* Service Offering (Hidden Field) */}
      <input type="hidden" value={serviceOfferingId} />

      {/* Validation Section */}
      {connectionSuccess && (
        <Card>
          <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
            CloudStack Validation
          </h3>
          
          <div className="space-y-4">
            <Button
              onClick={handleValidate}
              disabled={validating}
              color="blue"
              size="lg"
            >
              {validating ? (
                <>
                  <Spinner size="sm" className="mr-2" />
                  Validating...
                </>
              ) : (
                <>
                  <HiCheckCircle className="mr-2 h-5 w-5" />
                  Test and Discover Resources
                </>
              )}
            </Button>

            {validationResult && (
              <div className="space-y-3">
                {/* Overall Status */}
                <div className={`p-4 rounded-lg border-2 ${
                  validationResult.overall_status === 'pass'
                    ? 'bg-green-50 dark:bg-green-900/20 border-green-300 dark:border-green-700'
                    : validationResult.overall_status === 'warning'
                    ? 'bg-yellow-50 dark:bg-yellow-900/20 border-yellow-300 dark:border-yellow-700'
                    : 'bg-red-50 dark:bg-red-900/20 border-red-300 dark:border-red-700'
                }`}>
                  <div className="flex items-center justify-between">
                    <span className="font-semibold text-lg">Overall Status:</span>
                    {getStatusBadge(validationResult.overall_status)}
                  </div>
                </div>

                {/* Individual Checks */}
                <div className="space-y-2">
                  {/* OMA VM Detection */}
                  <div className="flex items-start space-x-3 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
                    {getStatusIcon(validationResult.oma_vm_detection.status)}
                    <div className="flex-1">
                      <div className="flex items-center justify-between">
                        <span className="font-medium">OMA VM Detection</span>
                        {getStatusBadge(validationResult.oma_vm_detection.status)}
                      </div>
                      <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                        {validationResult.oma_vm_detection.message}
                      </p>
                    </div>
                  </div>

                  {/* Compute Offering */}
                  <div className="flex items-start space-x-3 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
                    {getStatusIcon(validationResult.compute_offering.status)}
                    <div className="flex-1">
                      <div className="flex items-center justify-between">
                        <span className="font-medium">Compute Offering</span>
                        {getStatusBadge(validationResult.compute_offering.status)}
                      </div>
                      <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                        {validationResult.compute_offering.message}
                      </p>
                    </div>
                  </div>

                  {/* Account Match */}
                  <div className="flex items-start space-x-3 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
                    {getStatusIcon(validationResult.account_match.status)}
                    <div className="flex-1">
                      <div className="flex items-center justify-between">
                        <span className="font-medium">Account Match</span>
                        {getStatusBadge(validationResult.account_match.status)}
                      </div>
                      <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                        {validationResult.account_match.message}
                      </p>
                    </div>
                  </div>

                  {/* Network Selection */}
                  <div className="flex items-start space-x-3 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
                    {getStatusIcon(validationResult.network_selection.status)}
                    <div className="flex-1">
                      <div className="flex items-center justify-between">
                        <span className="font-medium">Network Selection</span>
                        {getStatusBadge(validationResult.network_selection.status)}
                      </div>
                      <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                        {validationResult.network_selection.message}
                      </p>
                    </div>
                  </div>
                </div>
              </div>
            )}
          </div>
        </Card>
      )}
    </div>
  );
};

