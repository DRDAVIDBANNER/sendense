// API route for applying network mapping recommendations
// This endpoint takes recommended mappings and creates them via the OMA API

import { NextRequest, NextResponse } from 'next/server';

// OMA API base URL
const OMA_API_BASE = 'http://localhost:8082/api/v1';

interface ApplyRecommendationRequest {
  vm_id?: string;
  recommendations: {
    source_network_name: string;
    destination_network_id: string;
    destination_network_name: string;
    is_test_network: boolean;
  }[];
}

export async function POST(request: NextRequest) {
  try {
    const body: ApplyRecommendationRequest = await request.json();
    const { vm_id, recommendations } = body;

    console.log('🎯 APPLY RECOMMENDATIONS API: Applying network recommendations', {
      vm_id,
      recommendation_count: recommendations.length,
      timestamp: new Date().toISOString()
    });

    if (!recommendations.length) {
      return NextResponse.json(
        { 
          error: 'No recommendations provided',
          message: 'At least one recommendation is required'
        },
        { status: 400 }
      );
    }

    const results = [];
    const errors = [];

    // If vm_id is provided, apply to specific VM
    if (vm_id) {
      for (const recommendation of recommendations) {
        try {
          const mappingPayload = {
            vm_id: vm_id,
            source_network_name: recommendation.source_network_name,
            destination_network_id: recommendation.destination_network_id,
            destination_network_name: recommendation.destination_network_name,
            is_test_network: recommendation.is_test_network
          };

          console.log(`📤 Creating recommended mapping: ${recommendation.source_network_name} → ${recommendation.destination_network_name}`);

          const response = await fetch(`${OMA_API_BASE}/network-mappings`, {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
              'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent',
            },
            body: JSON.stringify(mappingPayload)
          });

          if (response.ok) {
            const data = await response.json();
            results.push({
              source_network_name: recommendation.source_network_name,
              destination_network_name: recommendation.destination_network_name,
              is_test_network: recommendation.is_test_network,
              mapping_id: data.data?.id,
              status: 'created'
            });
            console.log(`✅ Recommendation applied: ${recommendation.source_network_name}`);
          } else {
            const errorData = await response.json();
            console.error(`❌ Failed to apply recommendation for ${recommendation.source_network_name}:`, errorData);
            errors.push({
              source_network_name: recommendation.source_network_name,
              error: errorData.error || 'Failed to create mapping'
            });
          }

        } catch (recError) {
          console.error(`❌ Error applying recommendation for ${recommendation.source_network_name}:`, recError);
          errors.push({
            source_network_name: recommendation.source_network_name,
            error: recError instanceof Error ? recError.message : 'Unknown error'
          });
        }
      }
    } else {
      // Apply recommendations across all VMs that have the source networks
      // First, fetch all VM contexts to find VMs with matching networks
      const vmContextsResponse = await fetch(`${OMA_API_BASE}/vm-contexts`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent',
        }
      });

      if (!vmContextsResponse.ok) {
        throw new Error('Failed to fetch VM contexts');
      }

      const vmContextsData = await vmContextsResponse.json();
      const vmContexts = vmContextsData.vm_contexts || [];

      // For each recommendation, find VMs that might have the source network
      for (const recommendation of recommendations) {
        const sourceNetworkName = recommendation.source_network_name;
        
        // Find VMs that might have this network (simplified logic)
        const candidateVMs = vmContexts.filter((context: {vm_name: string}) => {
          // In a real implementation, we would check actual VM network details
          // For now, use a simple pattern matching approach
          return sourceNetworkName.includes(context.vm_name) || 
                 sourceNetworkName === 'VM Network' || 
                 sourceNetworkName === 'Production Network';
        });

        for (const vmContext of candidateVMs) {
          try {
            const mappingPayload = {
              vm_id: vmContext.vm_name,
              source_network_name: recommendation.source_network_name,
              destination_network_id: recommendation.destination_network_id,
              destination_network_name: recommendation.destination_network_name,
              is_test_network: recommendation.is_test_network
            };

            console.log(`📤 Creating mapping for ${vmContext.vm_name}: ${recommendation.source_network_name} → ${recommendation.destination_network_name}`);

            const response = await fetch(`${OMA_API_BASE}/network-mappings`, {
              method: 'POST',
              headers: {
                'Content-Type': 'application/json',
                'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent',
              },
              body: JSON.stringify(mappingPayload)
            });

            if (response.ok) {
              const data = await response.json();
              results.push({
                vm_id: vmContext.vm_name,
                source_network_name: recommendation.source_network_name,
                destination_network_name: recommendation.destination_network_name,
                is_test_network: recommendation.is_test_network,
                mapping_id: data.data?.id,
                status: 'created'
              });
              console.log(`✅ Mapping created for ${vmContext.vm_name}`);
            } else {
              const errorData = await response.json();
              // Don't treat "already exists" as an error
              if (errorData.error?.includes('already exists') || errorData.error?.includes('duplicate')) {
                console.log(`ℹ️ Mapping already exists for ${vmContext.vm_name}: ${recommendation.source_network_name}`);
                results.push({
                  vm_id: vmContext.vm_name,
                  source_network_name: recommendation.source_network_name,
                  destination_network_name: recommendation.destination_network_name,
                  is_test_network: recommendation.is_test_network,
                  status: 'already_exists'
                });
              } else {
                console.error(`❌ Failed to create mapping for ${vmContext.vm_name}:`, errorData);
                errors.push({
                  vm_id: vmContext.vm_name,
                  source_network_name: recommendation.source_network_name,
                  error: errorData.error || 'Failed to create mapping'
                });
              }
            }

          } catch (vmError) {
            console.error(`❌ Error creating mapping for ${vmContext.vm_name}:`, vmError);
            errors.push({
              vm_id: vmContext.vm_name,
              source_network_name: recommendation.source_network_name,
              error: vmError instanceof Error ? vmError.message : 'Unknown error'
            });
          }
        }
      }
    }

    // Calculate summary
    const createdMappings = results.filter(r => r.status === 'created').length;
    const existingMappings = results.filter(r => r.status === 'already_exists').length;

    const response = {
      success: errors.length === 0,
      message: errors.length === 0 
        ? `Successfully applied ${createdMappings} network mapping recommendations`
        : `Applied ${createdMappings} recommendations with ${errors.length} errors`,
      summary: {
        total_recommendations: recommendations.length,
        mappings_created: createdMappings,
        mappings_existing: existingMappings,
        errors_count: errors.length,
        success_rate: ((createdMappings + existingMappings) / (results.length + errors.length) * 100).toFixed(1)
      },
      results,
      errors
    };

    console.log('✅ APPLY RECOMMENDATIONS API: Recommendations applied', {
      created: createdMappings,
      existing: existingMappings,
      errors: errors.length
    });

    return NextResponse.json(response);

  } catch (error) {
    console.error('❌ APPLY RECOMMENDATIONS API: Error applying recommendations', error);
    return NextResponse.json(
      { 
        error: 'Failed to apply network mapping recommendations',
        message: error instanceof Error ? error.message : 'Unknown error occurred',
        success: false,
        results: [],
        errors: []
      },
      { status: 500 }
    );
  }
}
