'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { Card, Button, Badge, Alert, Spinner, Modal } from 'flowbite-react';
import { HiOutlineLightBulb, HiOutlineCheckCircle, HiOutlineXCircle, HiOutlineCog, HiOutlineRefresh } from 'react-icons/hi';

interface NetworkRecommendation {
  id: string;
  source_network_name: string;
  recommended_network_id: string;
  recommended_network_name: string;
  confidence_score: number;
  reasoning: string[];
  vm_count: number;
  is_test_recommendation: boolean;
  performance_impact: 'low' | 'medium' | 'high';
  compatibility_score: number;
}

interface RecommendationCriteria {
  vm_requirements: string[];
  network_performance: string[];
  security_requirements: string[];
  availability_requirements: string[];
}

interface BulkRecommendationResult {
  recommendations: NetworkRecommendation[];
  summary: {
    total_networks: number;
    high_confidence: number;
    medium_confidence: number;
    low_confidence: number;
  };
  criteria: RecommendationCriteria;
}

interface NetworkRecommendationEngineProps {
  vmId?: string;
  onRecommendationApply?: (recommendations: NetworkRecommendation[]) => void;
}

export default function NetworkRecommendationEngine({ vmId, onRecommendationApply }: NetworkRecommendationEngineProps) {
  const [recommendations, setRecommendations] = useState<BulkRecommendationResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [selectedRecommendations, setSelectedRecommendations] = useState<Set<string>>(new Set());
  const [applyingRecommendations, setApplyingRecommendations] = useState(false);
  const [showCriteriaModal, setShowCriteriaModal] = useState(false);
  const [customCriteria, setCustomCriteria] = useState<RecommendationCriteria>({
    vm_requirements: [],
    network_performance: [],
    security_requirements: [],
    availability_requirements: []
  });

  const fetchRecommendations = useCallback(async (criteria?: RecommendationCriteria) => {
    try {
      setLoading(true);
      setError('');

      const endpoint = vmId 
        ? `/api/networks/recommendations?vm_id=${vmId}`
        : '/api/networks/recommendations';

      const requestBody = criteria ? { criteria } : {};

      const response = await fetch(endpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(requestBody)
      });

      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.error || 'Failed to fetch network recommendations');
      }

      setRecommendations(data);
      setSelectedRecommendations(new Set());
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to generate recommendations');
    } finally {
      setLoading(false);
    }
  }, [vmId]);

  useEffect(() => {
    fetchRecommendations();
  }, [fetchRecommendations]);

  const getConfidenceColor = (score: number) => {
    if (score >= 80) return 'success';
    if (score >= 60) return 'warning';
    return 'failure';
  };

  const getConfidenceLabel = (score: number) => {
    if (score >= 80) return 'High Confidence';
    if (score >= 60) return 'Medium Confidence';
    return 'Low Confidence';
  };

  const getPerformanceImpactColor = (impact: string) => {
    switch (impact) {
      case 'low': return 'success';
      case 'medium': return 'warning';
      case 'high': return 'failure';
      default: return 'gray';
    }
  };

  const handleRecommendationToggle = (recommendationId: string) => {
    const newSelected = new Set(selectedRecommendations);
    if (newSelected.has(recommendationId)) {
      newSelected.delete(recommendationId);
    } else {
      newSelected.add(recommendationId);
    }
    setSelectedRecommendations(newSelected);
  };

  const handleSelectAll = () => {
    if (!recommendations) return;
    
    const highConfidenceIds = recommendations.recommendations
      .filter(r => r.confidence_score >= 80)
      .map(r => r.id);
    
    setSelectedRecommendations(new Set(highConfidenceIds));
  };

  const handleDeselectAll = () => {
    setSelectedRecommendations(new Set());
  };

  const handleApplyRecommendations = async () => {
    if (!recommendations || selectedRecommendations.size === 0) return;

    try {
      setApplyingRecommendations(true);
      
      const selectedRecs = recommendations.recommendations.filter(
        r => selectedRecommendations.has(r.id)
      );

      // Apply recommendations via API
      const response = await fetch('/api/networks/apply-recommendations', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          vm_id: vmId,
          recommendations: selectedRecs.map(r => ({
            source_network_name: r.source_network_name,
            destination_network_id: r.recommended_network_id,
            destination_network_name: r.recommended_network_name,
            is_test_network: r.is_test_recommendation
          }))
        })
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || 'Failed to apply recommendations');
      }

      // Notify parent component
      if (onRecommendationApply) {
        onRecommendationApply(selectedRecs);
      }

      // Refresh recommendations
      fetchRecommendations();
      
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to apply recommendations');
    } finally {
      setApplyingRecommendations(false);
    }
  };

  const handleCustomCriteriaSubmit = () => {
    fetchRecommendations(customCriteria);
    setShowCriteriaModal(false);
  };

  if (loading && !recommendations) {
    return (
      <Card>
        <div className="flex items-center justify-center py-12">
          <Spinner size="lg" />
          <span className="ml-3 text-gray-500">Generating network recommendations...</span>
        </div>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-gray-900 dark:text-white flex items-center">
            <HiOutlineLightBulb className="mr-2 h-6 w-6 text-yellow-500" />
            Smart Network Recommendations
          </h2>
          <p className="text-gray-500 dark:text-gray-400">
            AI-powered network mapping suggestions based on VM requirements and network performance
          </p>
        </div>
        
        <div className="flex items-center space-x-2">
          <Button
            size="sm"
            color="gray"
            onClick={() => setShowCriteriaModal(true)}
          >
            <HiOutlineCog className="mr-2 h-4 w-4" />
            Customize Criteria
          </Button>
          <Button size="sm" color="gray" onClick={() => fetchRecommendations()}>
            <HiOutlineRefresh className="mr-2 h-4 w-4" />
            Refresh
          </Button>
        </div>
      </div>

      {error && (
        <Alert color="failure" onDismiss={() => setError('')}>
          {error}
        </Alert>
      )}

      {recommendations && (
        <>
          {/* Summary */}
          <Card>
            <h3 className="text-lg font-semibold mb-4">Recommendation Summary</h3>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              <div className="text-center">
                <div className="text-2xl font-bold text-gray-900 dark:text-white">
                  {recommendations.summary.total_networks}
                </div>
                <div className="text-sm text-gray-500">Total Networks</div>
              </div>
              <div className="text-center">
                <div className="text-2xl font-bold text-green-600">
                  {recommendations.summary.high_confidence}
                </div>
                <div className="text-sm text-gray-500">High Confidence</div>
              </div>
              <div className="text-center">
                <div className="text-2xl font-bold text-yellow-600">
                  {recommendations.summary.medium_confidence}
                </div>
                <div className="text-sm text-gray-500">Medium Confidence</div>
              </div>
              <div className="text-center">
                <div className="text-2xl font-bold text-red-600">
                  {recommendations.summary.low_confidence}
                </div>
                <div className="text-sm text-gray-500">Low Confidence</div>
              </div>
            </div>
          </Card>

          {/* Bulk Actions */}
          {recommendations.recommendations.length > 0 && (
            <Card>
              <div className="flex items-center justify-between">
                <div className="flex items-center space-x-4">
                  <span className="text-sm text-gray-500">
                    {selectedRecommendations.size} of {recommendations.recommendations.length} selected
                  </span>
                  <div className="flex space-x-2">
                    <Button size="sm" color="gray" onClick={handleSelectAll}>
                      Select High Confidence
                    </Button>
                    <Button size="sm" color="gray" onClick={handleDeselectAll}>
                      Deselect All
                    </Button>
                  </div>
                </div>
                
                <Button
                  size="sm"
                  color="blue"
                  disabled={selectedRecommendations.size === 0 || applyingRecommendations}
                  onClick={handleApplyRecommendations}
                >
                  {applyingRecommendations ? (
                    <>
                      <Spinner size="sm" className="mr-2" />
                      Applying...
                    </>
                  ) : (
                    <>
                      <HiOutlineCheckCircle className="mr-2 h-4 w-4" />
                      Apply Selected ({selectedRecommendations.size})
                    </>
                  )}
                </Button>
              </div>
            </Card>
          )}

          {/* Recommendations List */}
          <div className="grid gap-4">
            {recommendations.recommendations.map((recommendation) => (
              <Card key={recommendation.id} className={`cursor-pointer transition-colors ${
                selectedRecommendations.has(recommendation.id)
                  ? 'ring-2 ring-blue-500 bg-blue-50 dark:bg-blue-900/20'
                  : ''
              }`}>
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <div className="flex items-center space-x-3 mb-3">
                      <input
                        type="checkbox"
                        checked={selectedRecommendations.has(recommendation.id)}
                        onChange={() => handleRecommendationToggle(recommendation.id)}
                        className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                      />
                      <div>
                        <h4 className="text-lg font-semibold text-gray-900 dark:text-white">
                          {recommendation.source_network_name}
                        </h4>
                        <p className="text-sm text-gray-500">
                          → {recommendation.recommended_network_name}
                        </p>
                      </div>
                    </div>

                    <div className="flex items-center space-x-4 mb-3">
                      <Badge color={getConfidenceColor(recommendation.confidence_score)} size="sm">
                        {getConfidenceLabel(recommendation.confidence_score)} ({recommendation.confidence_score}%)
                      </Badge>
                      <Badge color={getPerformanceImpactColor(recommendation.performance_impact)} size="sm">
                        {recommendation.performance_impact.charAt(0).toUpperCase() + recommendation.performance_impact.slice(1)} Impact
                      </Badge>
                      <Badge color={recommendation.is_test_recommendation ? 'purple' : 'blue'} size="sm">
                        {recommendation.is_test_recommendation ? 'Test' : 'Production'}
                      </Badge>
                      <span className="text-sm text-gray-500">
                        {recommendation.vm_count} VMs
                      </span>
                    </div>

                    <div className="mb-3">
                      <h5 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                        Reasoning:
                      </h5>
                      <ul className="list-disc list-inside space-y-1">
                        {recommendation.reasoning.map((reason, index) => (
                          <li key={index} className="text-sm text-gray-600 dark:text-gray-400">
                            {reason}
                          </li>
                        ))}
                      </ul>
                    </div>

                    <div className="flex items-center justify-between">
                      <div className="text-sm text-gray-500">
                        Compatibility Score: {recommendation.compatibility_score}%
                      </div>
                      <div className="flex space-x-2">
                        {recommendation.confidence_score >= 80 && (
                          <HiOutlineCheckCircle className="h-5 w-5 text-green-500" />
                        )}
                        {recommendation.confidence_score < 60 && (
                          <HiOutlineXCircle className="h-5 w-5 text-red-500" />
                        )}
                      </div>
                    </div>
                  </div>
                </div>
              </Card>
            ))}
          </div>

          {recommendations.recommendations.length === 0 && (
            <Card>
              <div className="text-center py-12">
                <HiOutlineLightBulb className="mx-auto h-12 w-12 text-gray-400 mb-4" />
                <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
                  No Recommendations Available
                </h3>
                <p className="text-gray-500 dark:text-gray-400">
                  All networks appear to be optimally configured, or there are no unmapped networks to recommend.
                </p>
              </div>
            </Card>
          )}
        </>
      )}

      {/* Criteria Customization Modal */}
      <Modal show={showCriteriaModal} onClose={() => setShowCriteriaModal(false)} size="lg">
        <Modal.Header>Customize Recommendation Criteria</Modal.Header>
        <Modal.Body>
          <div className="space-y-6">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                VM Requirements
              </label>
              <div className="grid grid-cols-2 gap-2">
                {['High Performance', 'Low Latency', 'High Bandwidth', 'Isolated Network'].map((req) => (
                  <label key={req} className="flex items-center">
                    <input
                      type="checkbox"
                      checked={customCriteria.vm_requirements.includes(req)}
                      onChange={(e) => {
                        const newReqs = e.target.checked
                          ? [...customCriteria.vm_requirements, req]
                          : customCriteria.vm_requirements.filter(r => r !== req);
                        setCustomCriteria({...customCriteria, vm_requirements: newReqs});
                      }}
                      className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded mr-2"
                    />
                    <span className="text-sm">{req}</span>
                  </label>
                ))}
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                Security Requirements
              </label>
              <div className="grid grid-cols-2 gap-2">
                {['DMZ Network', 'Internal Only', 'VPN Required', 'Firewall Rules'].map((req) => (
                  <label key={req} className="flex items-center">
                    <input
                      type="checkbox"
                      checked={customCriteria.security_requirements.includes(req)}
                      onChange={(e) => {
                        const newReqs = e.target.checked
                          ? [...customCriteria.security_requirements, req]
                          : customCriteria.security_requirements.filter(r => r !== req);
                        setCustomCriteria({...customCriteria, security_requirements: newReqs});
                      }}
                      className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded mr-2"
                    />
                    <span className="text-sm">{req}</span>
                  </label>
                ))}
              </div>
            </div>
          </div>
        </Modal.Body>
        <Modal.Footer>
          <div className="flex justify-between w-full">
            <Button color="gray" onClick={() => setShowCriteriaModal(false)}>
              Cancel
            </Button>
            <Button color="blue" onClick={handleCustomCriteriaSubmit}>
              Apply Criteria
            </Button>
          </div>
        </Modal.Footer>
      </Modal>
    </div>
  );
}
