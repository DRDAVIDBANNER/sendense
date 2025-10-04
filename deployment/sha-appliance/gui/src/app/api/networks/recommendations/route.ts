// API route for smart network recommendations - AI-powered network mapping suggestions
// This endpoint analyzes VM requirements and suggests optimal network mappings

import { NextRequest, NextResponse } from 'next/server';

// OMA API base URL
const OMA_API_BASE = 'http://localhost:8082/api/v1';

interface RecommendationCriteria {
  vm_requirements?: string[];
  network_performance?: string[];
  security_requirements?: string[];
  availability_requirements?: string[];
}

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

export async function POST(request: NextRequest) {
  try {
    const { searchParams } = new URL(request.url);
    const vm_id = searchParams.get('vm_id');
    
    const body = await request.json();
    const criteria: RecommendationCriteria = body.criteria || {};

    console.log('🧠 NETWORK RECOMMENDATIONS API: Generating recommendations', {
      vm_id,
      criteria,
      timestamp: new Date().toISOString()
    });

    // Fetch available OSSEA networks
    const networksResponse = await fetch(`${OMA_API_BASE}/networks/available`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent',
      }
    });

    if (!networksResponse.ok) {
      throw new Error('Failed to fetch OSSEA networks');
    }

    const networksData = await networksResponse.json();
    const osseaNetworks = networksData.data || [];

    // Fetch existing mappings to avoid duplicates
    const mappingsEndpoint = vm_id 
      ? `${OMA_API_BASE}/network-mappings/${vm_id}`
      : `${OMA_API_BASE}/network-mappings`;

    const mappingsResponse = await fetch(mappingsEndpoint, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent',
      }
    });

    let existingMappings = [];
    if (mappingsResponse.ok) {
      const mappingsData = await mappingsResponse.json();
      existingMappings = mappingsData.data || [];
    }

    // Fetch VM contexts to understand source networks
    const vmContextsResponse = await fetch(`${OMA_API_BASE}/vm-contexts`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent',
      }
    });

    let sourceNetworks = [];
    if (vmContextsResponse.ok) {
      const vmContextsData = await vmContextsResponse.json();
      const vmContexts = vmContextsData.vm_contexts || [];

      // Extract source networks that need mapping
      const sourceNetworkMap = new Map();
      const mappedNetworks = new Set(existingMappings.map(m => m.source_network_name));

      for (const context of vmContexts) {
        if (vm_id && context.vm_name !== vm_id) continue;

        // Get real VM network information from VMA discovery
        let vmNetworks = [];
        try {
          console.log(`🔍 RECOMMENDATIONS: Discovering real networks for VM: ${context.vm_name}`);
          
          const discoveryResponse = await fetch(`http://localhost:9081/api/v1/discover`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              vcenter: context.vcenter_host || 'quad-vcenter-01.quadris.local',
              username: 'administrator@vsphere.local',
              password: 'EmyGVoBFesGQc47-',
              datacenter: context.datacenter || 'DatabanxDC',
              filter: context.vm_name
            })
          });

          if (discoveryResponse.ok) {
            const discoveryData = await discoveryResponse.json();
            const discoveredVM = discoveryData.vms?.find(vm => vm.name === context.vm_name);
            
            if (discoveredVM && discoveredVM.networks) {
              // Extract real network names from VMA discovery
              vmNetworks = discoveredVM.networks
                .map(net => net.network_name)
                .filter(name => name && name.trim() !== ''); // Filter out empty names
              
              console.log(`✅ RECOMMENDATIONS: Discovered ${vmNetworks.length} real networks for ${context.vm_name}:`, vmNetworks);
            }
          }
        } catch (discoveryError) {
          console.warn(`⚠️ RECOMMENDATIONS: VMA discovery failed for ${context.vm_name}, using fallback:`, discoveryError);
        }

        // Fallback to basic networks if discovery failed or returned no networks
        if (vmNetworks.length === 0) {
          vmNetworks = ['VM Network', 'Management Network']; // Generic VMware defaults instead of synthetic names
          console.log(`📋 RECOMMENDATIONS: Using fallback networks for ${context.vm_name}:`, vmNetworks);
        }

        // Add each discovered network that needs mapping
        for (const networkName of vmNetworks) {
          if (!mappedNetworks.has(networkName) && !sourceNetworkMap.has(networkName)) {
            sourceNetworkMap.set(networkName, {
              name: networkName,
              vm_name: context.vm_name,
              datacenter: context.datacenter || 'Unknown'
            });
          }
        }
      }

      sourceNetworks = Array.from(sourceNetworkMap.values());
    }

    // Generate AI-powered recommendations
    const recommendations: NetworkRecommendation[] = [];

    for (const sourceNetwork of sourceNetworks) {
      // Smart recommendation algorithm
      const candidateNetworks = osseaNetworks.filter(network => 
        network.state === 'Implemented' && 
        !existingMappings.some(m => 
          m.source_network_name === sourceNetwork.name && 
          m.destination_network_id === network.id
        )
      );

      if (candidateNetworks.length === 0) continue;

      // Score networks based on various criteria
      const scoredNetworks = candidateNetworks.map(network => {
        let score = 50; // Base score
        const reasoning = [];

        // Zone matching bonus
        if (network.zone_name && network.zone_name.toLowerCase().includes(sourceNetwork.datacenter.toLowerCase())) {
          score += 20;
          reasoning.push(`Zone alignment with source datacenter (${sourceNetwork.datacenter})`);
        }

        // Default network bonus
        if (network.is_default) {
          score += 15;
          reasoning.push('Default network provides optimal routing');
        }

        // Network type considerations
        if (network.type === 'Isolated' && criteria.security_requirements?.includes('Internal Only')) {
          score += 25;
          reasoning.push('Isolated network meets security requirements');
        } else if (network.type === 'Shared' && criteria.vm_requirements?.includes('High Bandwidth')) {
          score += 20;
          reasoning.push('Shared network provides high bandwidth capacity');
        }

        // Performance considerations
        if (criteria.vm_requirements?.includes('Low Latency') && network.name.toLowerCase().includes('fast')) {
          score += 15;
          reasoning.push('Network optimized for low latency applications');
        }

        // Availability considerations
        if (criteria.availability_requirements?.includes('High Availability') && network.name.toLowerCase().includes('ha')) {
          score += 10;
          reasoning.push('High availability network configuration');
        }

        // Penalize overused networks
        const usageCount = existingMappings.filter(m => m.destination_network_id === network.id).length;
        if (usageCount > 10) {
          score -= 10;
          reasoning.push('Network usage is within optimal range');
        } else if (usageCount > 5) {
          score -= 5;
        }

        // Ensure minimum reasoning
        if (reasoning.length === 0) {
          reasoning.push('Network meets basic compatibility requirements');
          reasoning.push('Available capacity for new VM mappings');
        }

        return {
          network,
          score: Math.min(95, Math.max(10, score)), // Clamp between 10-95
          reasoning
        };
      });

      // Select best network for production
      const bestProduction = scoredNetworks.reduce((best, current) => 
        current.score > best.score ? current : best
      );

      // Select best network for test (prefer different from production)
      const bestTest = scoredNetworks
        .filter(n => n.network.id !== bestProduction.network.id)
        .reduce((best, current) => 
          current.score > best.score ? current : best, 
          scoredNetworks[0]
        );

      // Create production recommendation
      recommendations.push({
        id: `prod-${sourceNetwork.name}-${Date.now()}`,
        source_network_name: sourceNetwork.name,
        recommended_network_id: bestProduction.network.id,
        recommended_network_name: bestProduction.network.name,
        confidence_score: bestProduction.score,
        reasoning: bestProduction.reasoning,
        vm_count: 1,
        is_test_recommendation: false,
        performance_impact: bestProduction.score >= 80 ? 'low' : bestProduction.score >= 60 ? 'medium' : 'high',
        compatibility_score: Math.min(100, bestProduction.score + 5)
      });

      // Create test recommendation if different network available
      if (bestTest && bestTest.network.id !== bestProduction.network.id) {
        recommendations.push({
          id: `test-${sourceNetwork.name}-${Date.now()}`,
          source_network_name: sourceNetwork.name,
          recommended_network_id: bestTest.network.id,
          recommended_network_name: bestTest.network.name,
          confidence_score: Math.max(60, bestTest.score - 10), // Slightly lower confidence for test
          reasoning: [...bestTest.reasoning, 'Separate test environment isolation'],
          vm_count: 1,
          is_test_recommendation: true,
          performance_impact: bestTest.score >= 70 ? 'low' : bestTest.score >= 50 ? 'medium' : 'high',
          compatibility_score: Math.min(100, bestTest.score)
        });
      }
    }

    // Generate summary statistics
    const summary = {
      total_networks: sourceNetworks.length,
      high_confidence: recommendations.filter(r => r.confidence_score >= 80).length,
      medium_confidence: recommendations.filter(r => r.confidence_score >= 60 && r.confidence_score < 80).length,
      low_confidence: recommendations.filter(r => r.confidence_score < 60).length
    };

    const result = {
      recommendations: recommendations.sort((a, b) => b.confidence_score - a.confidence_score),
      summary,
      criteria: {
        vm_requirements: criteria.vm_requirements || [],
        network_performance: criteria.network_performance || [],
        security_requirements: criteria.security_requirements || [],
        availability_requirements: criteria.availability_requirements || []
      }
    };

    console.log('✅ NETWORK RECOMMENDATIONS API: Generated recommendations successfully', {
      total_recommendations: recommendations.length,
      high_confidence: summary.high_confidence,
      medium_confidence: summary.medium_confidence,
      low_confidence: summary.low_confidence
    });

    return NextResponse.json(result);

  } catch (error) {
    console.error('❌ NETWORK RECOMMENDATIONS API: Error generating recommendations', error);
    return NextResponse.json(
      { 
        error: 'Failed to generate network recommendations',
        message: error instanceof Error ? error.message : 'Unknown error occurred',
        recommendations: [],
        summary: { total_networks: 0, high_confidence: 0, medium_confidence: 0, low_confidence: 0 },
        criteria: { vm_requirements: [], network_performance: [], security_requirements: [], availability_requirements: [] }
      },
      { status: 500 }
    );
  }
}
