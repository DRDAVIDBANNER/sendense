'use client';

import React, { useState, useEffect } from 'react';
import { Card, Label, TextInput, Button, Alert, Spinner, Select, Badge } from 'flowbite-react';
import { HiCheckCircle, HiXCircle, HiExclamation, HiRefresh, HiServer, HiCloudDownload, HiSave } from 'react-icons/hi';

// ============================================================================
// TYPES
// ============================================================================

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

interface DiscoveredResources {
  oma_vm_id: string;
  oma_vm_name?: string;
  zones: Array<{ id: string; name: string; description?: string }>;
  domains: Array<{ id: string; name: string; path?: string }>;
  templates: Array<{ id: string; name: string; description: string; os_type: string }>;
  service_offerings: Array<{ id: string; name: string; description: string; cpu: number; memory: number }>;
  disk_offerings: Array<{ id: string; name: string; description: string; disk_size_gb?: number }>;
  networks: Array<{ id: string; name: string; zone_id?: string; zone_name?: string; state?: string }>;
}

type Step = 'connection' | 'selection' | 'complete';

// ============================================================================
// UNIFIED OSSEA CONFIGURATION COMPONENT
// ============================================================================

export const UnifiedOSSEAConfiguration: React.FC = () => {
  // ===========================
  // STATE MANAGEMENT
  // ===========================
  
  // Current step
  const [step, setStep] = useState<Step>('connection');
  
  // Connection credentials (entered once)
  const [hostname, setHostname] = useState('');
  const [apiKey, setApiKey] = useState('');
  const [secretKey, setSecretKey] = useState('');
  const [domain, setDomain] = useState('');
  
  // Discovered resources
  const [discovered, setDiscovered] = useState<DiscoveredResources | null>(null);
  
  // User selections
  const [selectedZone, setSelectedZone] = useState('');
  const [selectedTemplate, setSelectedTemplate] = useState('');
  const [selectedServiceOffering, setSelectedServiceOffering] = useState('');
  const [selectedDiskOffering, setSelectedDiskOffering] = useState('');
  const [selectedNetwork, setSelectedNetwork] = useState('');
  
  // Validation results
  const [validation, setValidation] = useState<ValidationResult | null>(null);
  
  // UI state
  const [testing, setTesting] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  
  // ===========================
  // LOAD EXISTING CONFIG
  // ===========================
  
  useEffect(() => {
    loadExistingConfig();
  }, []);
  
  const loadExistingConfig = async () => {
    try {
      const response = await fetch('/api/settings/ossea');
      if (response.ok) {
        const data = await response.json();
        if (data) {
          // Extract hostname from api_url
          const urlMatch = data.api_url?.match(/https?:\/\/([^\/]+)/);
          if (urlMatch) {
            setHostname(urlMatch[1]);
          }
          
          // Pre-fill domain
          setDomain(data.domain || '');
          
          // If we have a saved config, we can go straight to selection
          // (but user needs to re-enter credentials for security)
          if (data.zone && data.oma_vm_id) {
            // Show that config exists
            setSuccess('Configuration found. Enter credentials to modify or validate.');
          }
        }
      }
    } catch (err) {
      console.error('Failed to load existing config:', err);
    }
  };
  
  // ===========================
  // STEP 1: TEST & DISCOVER
  // ===========================
  
  const handleTestAndDiscover = async () => {
    setError(null);
    setSuccess(null);
    setTesting(true);
    
    try {
      // Build API URL from hostname
      const apiUrl = `http://${hostname}/client/api`;
      
      // Call combined discovery endpoint
      const response = await fetch('/api/cloudstack/discover-all', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          api_url: apiUrl,
          api_key: apiKey,
          secret_key: secretKey,
          domain: domain || 'ROOT'
        })
      });
      
      const data = await response.json();
      
      if (!response.ok) {
        throw new Error(data.error || 'Discovery failed');
      }
      
      // Store discovered resources
      setDiscovered(data);
      
      // Auto-select if only one option
      if (data.zones?.length === 1) {
        setSelectedZone(data.zones[0].id);
      }
      
      // Auto-select default network if available
      const defaultNetwork = data.networks?.find((n: any) => n.is_default);
      if (defaultNetwork) {
        setSelectedNetwork(defaultNetwork.id);
      }
      
      setSuccess(`✅ Connected successfully! Found OMA VM: ${data.oma_vm_name || 'Detected'}`);
      setStep('selection');
      
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to connect to CloudStack');
    } finally {
      setTesting(false);
    }
  };
  
  // ===========================
  // STEP 2: VALIDATE & SAVE
  // ===========================
  
  const handleValidateAndSave = async () => {
    setError(null);
    setSuccess(null);
    setSaving(true);
    
    try {
      // Build complete configuration
      const apiUrl = `http://${hostname}/client/api`;
      const config = {
        name: 'Production CloudStack',  // Required: configuration name
        api_url: apiUrl,
        api_key: apiKey,
        secret_key: secretKey,
        domain: domain || 'ROOT',
        zone: selectedZone,
        oma_vm_id: discovered?.oma_vm_id || '',
        template_id: selectedTemplate,
        service_offering_id: selectedServiceOffering,
        disk_offering_id: selectedDiskOffering,
        network_id: selectedNetwork
      };
      
      // First, validate the configuration
      const validateResponse = await fetch('/api/cloudstack/validate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(config)
      });
      
      const validationData = await validateResponse.json();
      console.log('Validation response:', validationData);
      
      // Extract the actual validation result from the response
      // Backend wraps it in { success, result, message }
      const result = validationData.result || validationData;
      console.log('Extracted validation result:', result);
      setValidation(result);
      
      // Check if validation failed critically
      if (result?.overall_status === 'fail') {
        setError('Validation failed. Please fix critical issues before saving.');
        return;
      }
      
      // Save the configuration (credentials will be encrypted by backend)
      const saveResponse = await fetch('/api/settings/ossea', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(config)
      });
      
      if (!saveResponse.ok) {
        const saveError = await saveResponse.json();
        throw new Error(saveError.error || 'Failed to save configuration');
      }
      
      setSuccess('✅ Configuration saved and encrypted successfully!');
      setStep('complete');
      
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save configuration');
    } finally {
      setSaving(false);
    }
  };
  
  // ===========================
  // STEP 3: RESET
  // ===========================
  
  const handleReset = () => {
    setStep('connection');
    setDiscovered(null);
    setValidation(null);
    setError(null);
    setSuccess(null);
  };
  
  // ===========================
  // RENDER HELPERS
  // ===========================
  
  const renderValidationBadge = (check: ValidationCheck | undefined) => {
    // Handle undefined check
    if (!check) {
      return (
        <div className="flex items-center space-x-2">
          <HiExclamation className="h-5 w-5 text-gray-500" />
          <span className="text-sm text-gray-500">Not checked</span>
        </div>
      );
    }
    
    const colors = {
      pass: 'success',
      warning: 'warning',
      fail: 'failure',
      skipped: 'gray'
    };
    
    const icons = {
      pass: HiCheckCircle,
      warning: HiExclamation,
      fail: HiXCircle,
      skipped: HiRefresh
    };
    
    const Icon = icons[check.status];
    
    return (
      <div className="flex items-center space-x-2">
        <Icon className={`h-5 w-5 ${
          check.status === 'pass' ? 'text-green-500' :
          check.status === 'warning' ? 'text-yellow-500' :
          check.status === 'fail' ? 'text-red-500' :
          'text-gray-500'
        }`} />
        <span className="text-sm">{check.message}</span>
      </div>
    );
  };
  
  // ===========================
  // RENDER: STEP 1 - CONNECTION
  // ===========================
  
  const renderConnectionStep = () => (
    <Card>
      <div className="space-y-4">
        <div className="flex items-center space-x-2">
          <HiServer className="h-6 w-6 text-blue-500" />
          <h3 className="text-xl font-semibold text-gray-900 dark:text-white">
            Step 1: CloudStack Connection
          </h3>
        </div>
        
        <p className="text-sm text-gray-600 dark:text-gray-400">
          Enter your CloudStack credentials. We'll automatically discover available resources and detect your OMA VM.
        </p>
        
        {error && (
          <Alert color="failure" icon={HiXCircle}>
            {error}
          </Alert>
        )}
        
        {success && (
          <Alert color="success" icon={HiCheckCircle}>
            {success}
          </Alert>
        )}
        
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div>
            <Label htmlFor="hostname" value="CloudStack Hostname:Port" />
            <TextInput
              id="hostname"
              type="text"
              placeholder="10.246.2.11:8080"
              value={hostname}
              onChange={(e) => setHostname(e.target.value)}
              disabled={testing}
            />
            <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
              Enter hostname:port only (we'll add /client/api automatically)
            </p>
          </div>
          
          <div>
            <Label htmlFor="domain" value="CloudStack Domain" />
            <TextInput
              id="domain"
              type="text"
              placeholder="151"
              value={domain}
              onChange={(e) => setDomain(e.target.value)}
              disabled={testing}
            />
            <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
              Enter your CloudStack domain name (e.g., '151'). Leave empty for ROOT domain.
            </p>
          </div>
        </div>
        
        <div>
          <Label htmlFor="apiKey" value="API Key" />
          <TextInput
            id="apiKey"
            type="text"
            placeholder="Your CloudStack API key"
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
            placeholder="Your CloudStack secret key"
            value={secretKey}
            onChange={(e) => setSecretKey(e.target.value)}
            disabled={testing}
          />
        </div>
        
        <Button
          color="blue"
          size="lg"
          onClick={handleTestAndDiscover}
          disabled={!hostname || !apiKey || !secretKey || testing}
          className="w-full"
        >
          {testing ? (
            <>
              <Spinner size="sm" className="mr-2" />
              Testing Connection & Discovering Resources...
            </>
          ) : (
            <>
              <HiCloudDownload className="mr-2 h-5 w-5" />
              Test Connection & Discover Resources
            </>
          )}
        </Button>
      </div>
    </Card>
  );
  
  // ===========================
  // RENDER: STEP 2 - SELECTION
  // ===========================
  
  const renderSelectionStep = () => (
    <div className="space-y-6">
      {/* Auto-Discovery Results */}
      <Card>
        <div className="space-y-4">
          <div className="flex items-center space-x-2">
            <HiCheckCircle className="h-6 w-6 text-green-500" />
            <h3 className="text-xl font-semibold text-gray-900 dark:text-white">
              Step 2: Auto-Discovery Results
            </h3>
          </div>
          
          <div className="grid grid-cols-1 gap-3 rounded-lg bg-green-50 p-4 dark:bg-green-900/20 md:grid-cols-2">
            <div className="flex items-center space-x-2">
              <HiCheckCircle className="h-5 w-5 text-green-600" />
              <span className="text-sm text-gray-700 dark:text-gray-300">
                Connected to CloudStack successfully
              </span>
            </div>
            <div className="flex items-center space-x-2">
              <HiCheckCircle className="h-5 w-5 text-green-600" />
              <span className="text-sm text-gray-700 dark:text-gray-300">
                OMA VM: {discovered?.oma_vm_name || 'Detected'}
              </span>
            </div>
            <div className="flex items-center space-x-2">
              <HiCheckCircle className="h-5 w-5 text-green-600" />
              <span className="text-sm text-gray-700 dark:text-gray-300">
                Found {discovered?.zones?.length || 0} zone(s)
              </span>
            </div>
            <div className="flex items-center space-x-2">
              <HiCheckCircle className="h-5 w-5 text-green-600" />
              <span className="text-sm text-gray-700 dark:text-gray-300">
                Found {discovered?.templates?.length || 0} template(s)
              </span>
            </div>
            <div className="flex items-center space-x-2">
              <HiCheckCircle className="h-5 w-5 text-green-600" />
              <span className="text-sm text-gray-700 dark:text-gray-300">
                Found {discovered?.service_offerings?.length || 0} service offering(s)
              </span>
            </div>
            <div className="flex items-center space-x-2">
              <HiCheckCircle className="h-5 w-5 text-green-600" />
              <span className="text-sm text-gray-700 dark:text-gray-300">
                Found {discovered?.networks?.length || 0} network(s)
              </span>
            </div>
          </div>
        </div>
      </Card>
      
      {/* Resource Selection */}
      <Card>
        <div className="space-y-4">
          <h3 className="text-xl font-semibold text-gray-900 dark:text-white">
            Step 3: Resource Selection
          </h3>
          
          <p className="text-sm text-gray-600 dark:text-gray-400">
            Select the CloudStack resources to use for VM migrations and failover operations.
          </p>
          
          {error && (
            <Alert color="failure" icon={HiXCircle}>
              {error}
            </Alert>
          )}
          
          {/* OMA VM ID (read-only, auto-detected) */}
          <div>
            <Label htmlFor="omaVmId" value="OMA VM ID (Auto-Detected)" />
            <TextInput
              id="omaVmId"
              type="text"
              value={discovered?.oma_vm_id || ''}
              disabled
            />
            <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
              Automatically detected by MAC address matching
            </p>
          </div>
          
          {/* Zone Selection */}
          <div>
            <Label htmlFor="zone" value="CloudStack Zone *" />
            <Select
              id="zone"
              value={selectedZone}
              onChange={(e) => setSelectedZone(e.target.value)}
              required
            >
              <option value="">Select a zone...</option>
              {discovered?.zones?.map((zone) => (
                <option key={zone.id} value={zone.id}>
                  {zone.name} {zone.description ? `(${zone.description})` : ''}
                </option>
              ))}
            </Select>
          </div>
          
          {/* Template Selection */}
          <div>
            <Label htmlFor="template" value="Default Template" />
            <Select
              id="template"
              value={selectedTemplate}
              onChange={(e) => setSelectedTemplate(e.target.value)}
            >
              <option value="">Select a template (optional)...</option>
              {discovered?.templates?.map((template) => (
                <option key={template.id} value={template.id}>
                  {template.name} - {template.os_type}
                </option>
              ))}
            </Select>
            <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
              Optional: Default template for VM creation
            </p>
          </div>
          
          {/* Service Offering Selection */}
          <div>
            <Label htmlFor="serviceOffering" value="Default Service Offering" />
            <Select
              id="serviceOffering"
              value={selectedServiceOffering}
              onChange={(e) => setSelectedServiceOffering(e.target.value)}
            >
              <option value="">Select a service offering (optional)...</option>
              {discovered?.service_offerings?.map((offering) => (
                <option key={offering.id} value={offering.id}>
                  {offering.name} ({offering.cpu} CPU, {offering.memory}MB RAM)
                </option>
              ))}
            </Select>
            <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
              Optional: Default compute resources for VMs
            </p>
          </div>
          
          {/* Disk Offering Selection */}
          <div>
            <Label htmlFor="diskOffering" value="Disk Offering *" />
            <Select
              id="diskOffering"
              value={selectedDiskOffering}
              onChange={(e) => setSelectedDiskOffering(e.target.value)}
              required
            >
              <option value="">Select a disk offering...</option>
              {discovered?.disk_offerings?.map((offering) => (
                <option key={offering.id} value={offering.id}>
                  {offering.name} {offering.disk_size_gb ? `(${offering.disk_size_gb}GB)` : ''}
                </option>
              ))}
            </Select>
            <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
              Required: Used for volume provisioning during replication
            </p>
          </div>
          
          {/* Network Selection */}
          <div>
            <Label htmlFor="network" value="Default Network" />
            <Select
              id="network"
              value={selectedNetwork}
              onChange={(e) => setSelectedNetwork(e.target.value)}
            >
              <option value="">Select a network (optional)...</option>
              {discovered?.networks?.map((network) => (
                <option key={network.id} value={network.id}>
                  {network.name} {network.zone_name ? `- ${network.zone_name}` : ''} ({network.state})
                </option>
              ))}
            </Select>
            <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
              Optional: Default network for VM failover
            </p>
          </div>
          
          <div className="flex space-x-4">
            <Button
              color="gray"
              onClick={handleReset}
              disabled={saving}
            >
              <HiRefresh className="mr-2 h-4 w-4" />
              Start Over
            </Button>
            
            <Button
              color="blue"
              size="lg"
              onClick={handleValidateAndSave}
              disabled={!selectedZone || !discovered?.oma_vm_id || !selectedDiskOffering || saving}
              className="flex-1"
            >
              {saving ? (
                <>
                  <Spinner size="sm" className="mr-2" />
                  Validating & Saving...
                </>
              ) : (
                <>
                  <HiSave className="mr-2 h-5 w-5" />
                  Validate & Save Configuration
                </>
              )}
            </Button>
          </div>
        </div>
      </Card>
    </div>
  );
  
  // ===========================
  // RENDER: STEP 3 - COMPLETE
  // ===========================
  
  const renderCompleteStep = () => (
    <div className="space-y-6">
      {/* Success Message */}
      <Alert color="success" icon={HiCheckCircle}>
        <span className="font-semibold">Configuration saved successfully!</span>
        <p className="mt-1 text-sm">
          Your CloudStack credentials have been encrypted and stored securely.
        </p>
      </Alert>
      
      {/* Validation Results */}
      {validation && (
        <Card>
          <div className="space-y-4">
            <h3 className="text-xl font-semibold text-gray-900 dark:text-white">
              Validation Results
            </h3>
            
            <div className="space-y-3">
              <div className="flex items-center justify-between rounded-lg border border-gray-200 p-3 dark:border-gray-700">
                <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
                  OMA VM Detection
                </span>
                {renderValidationBadge(validation?.oma_vm_detection)}
              </div>
              
              <div className="flex items-center justify-between rounded-lg border border-gray-200 p-3 dark:border-gray-700">
                <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
                  Compute Offering
                </span>
                {renderValidationBadge(validation?.compute_offering)}
              </div>
              
              <div className="flex items-center justify-between rounded-lg border border-gray-200 p-3 dark:border-gray-700">
                <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
                  Account Match
                </span>
                {renderValidationBadge(validation?.account_match)}
              </div>
              
              <div className="flex items-center justify-between rounded-lg border border-gray-200 p-3 dark:border-gray-700">
                <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
                  Network Selection
                </span>
                {renderValidationBadge(validation?.network_selection)}
              </div>
            </div>
            
            <div className="mt-4 rounded-lg bg-blue-50 p-4 dark:bg-blue-900/20">
              <p className="text-sm text-blue-800 dark:text-blue-300">
                <strong>Overall Status:</strong>{' '}
                {validation?.overall_status === 'pass' && '✅ All prerequisites met'}
                {validation?.overall_status === 'warning' && '⚠️ Configuration complete with warnings'}
                {validation?.overall_status === 'fail' && '❌ Critical issues found'}
                {!validation?.overall_status && '⏳ Validation pending...'}
              </p>
            </div>
          </div>
        </Card>
      )}
      
      {/* Actions */}
      <div className="flex justify-end">
        <Button color="gray" onClick={handleReset}>
          <HiRefresh className="mr-2 h-4 w-4" />
          Modify Configuration
        </Button>
      </div>
    </div>
  );
  
  // ===========================
  // MAIN RENDER
  // ===========================
  
  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-gray-900 dark:text-white">
          🔧 OSSEA Configuration
        </h2>
        <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
          Simplified CloudStack configuration with auto-discovery and human-readable options
        </p>
      </div>
      
      {/* Progress Indicator */}
      <div className="flex items-center space-x-4">
        <div className={`flex items-center space-x-2 ${step === 'connection' ? 'text-blue-600' : 'text-gray-400'}`}>
          <div className={`flex h-8 w-8 items-center justify-center rounded-full ${
            step === 'connection' ? 'bg-blue-600 text-white' : 
            step === 'selection' || step === 'complete' ? 'bg-green-600 text-white' : 'bg-gray-300'
          }`}>
            1
          </div>
          <span className="text-sm font-medium">Connection</span>
        </div>
        
        <div className="h-0.5 w-16 bg-gray-300"></div>
        
        <div className={`flex items-center space-x-2 ${step === 'selection' ? 'text-blue-600' : 'text-gray-400'}`}>
          <div className={`flex h-8 w-8 items-center justify-center rounded-full ${
            step === 'selection' ? 'bg-blue-600 text-white' :
            step === 'complete' ? 'bg-green-600 text-white' : 'bg-gray-300'
          }`}>
            2
          </div>
          <span className="text-sm font-medium">Selection</span>
        </div>
        
        <div className="h-0.5 w-16 bg-gray-300"></div>
        
        <div className={`flex items-center space-x-2 ${step === 'complete' ? 'text-green-600' : 'text-gray-400'}`}>
          <div className={`flex h-8 w-8 items-center justify-center rounded-full ${
            step === 'complete' ? 'bg-green-600 text-white' : 'bg-gray-300'
          }`}>
            3
          </div>
          <span className="text-sm font-medium">Complete</span>
        </div>
      </div>
      
      {/* Step Content */}
      {step === 'connection' && renderConnectionStep()}
      {step === 'selection' && renderSelectionStep()}
      {step === 'complete' && renderCompleteStep()}
    </div>
  );
};

