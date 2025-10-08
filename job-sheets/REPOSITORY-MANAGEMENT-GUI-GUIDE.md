# Repository Management GUI Guide

## 📚 Overview

The Repository Management interface provides complete control over backup storage repositories in the Sendense Professional platform. This feature enables administrators to configure, monitor, and manage all storage destinations used by protection flows.

## 🎯 Key Features

### Multi-Type Repository Support
- **Local Storage**: Direct attached storage or local filesystem paths
- **Amazon S3**: Cloud storage with bucket configuration
- **NFS Shares**: Network File System mount points
- **CIFS/SMB Shares**: Windows file server connections
- **Azure Blob Storage**: Microsoft cloud storage integration

### Comprehensive Management
- Real-time health monitoring and status tracking
- Capacity usage visualization and alerts
- Connection testing and validation
- Full CRUD operations (Create, Read, Update, Delete)
- Professional enterprise-grade interface

## 🖥️ User Interface

### Navigation Access
1. Open Sendense Professional GUI
2. Navigate to sidebar menu
3. Click **"Repositories"** (positioned between Appliances and Report Center)

### Main Dashboard

#### Summary Cards
- **Total Repositories**: Count of all configured repositories
- **Online**: Number of healthy, accessible repositories
- **Warning**: Repositories with issues or high usage
- **Offline**: Inaccessible or failed repositories

#### Capacity Overview
- **Total Capacity**: Aggregate storage across all repositories
- **Usage Visualization**: Progress bar showing overall utilization
- **Distribution**: Breakdown by repository type and location

#### Repository Grid
- **Card-based Layout**: Professional repository cards with status indicators
- **Health Status**: Visual indicators for online/warning/offline states
- **Capacity Metrics**: Storage usage with progress bars and percentages
- **Quick Actions**: Edit, test connection, and delete options

## 🔧 Repository Configuration

### Adding New Repositories

#### Step 1: Select Repository Type
1. Click **"Add Repository"** button
2. Choose from 5 repository types:
   - Local Storage
   - Amazon S3
   - NFS Share
   - CIFS/SMB Share
   - Azure Blob Storage

#### Step 2: Basic Configuration
- **Repository Name**: Unique identifier for the repository
- **Description**: Optional descriptive text
- **Type-specific Fields**: Varies by repository type

#### Step 3: Connection Testing
- **Test Connection**: Validate repository accessibility
- **Status Feedback**: Success/error indicators with details
- **Retry Logic**: Automatic retry on transient failures

#### Step 4: Repository Creation
- **Save Repository**: Commit configuration to system
- **Immediate Availability**: Repository ready for use in protection flows

### Repository Type Specifications

#### Local Storage
```
Required Fields:
- Storage Path: /mnt/storage or C:\Storage
- Description: Optional context

Example: /mnt/primary-storage
```

#### Amazon S3
```
Required Fields:
- Bucket Name: my-backup-bucket
- Region: us-east-1, eu-west-1, etc.
- Access Key: AWS IAM access key
- Secret Key: AWS IAM secret key
- Endpoint: Optional custom S3 endpoint

Example: sendense-backups (us-east-1)
```

#### NFS Share
```
Required Fields:
- NFS Server: nfs-server.example.com
- Export Path: /export/backups
- Mount Options: vers=4,soft,timeo=30

Example: nas-server:/export/archive
```

#### CIFS/SMB Share
```
Required Fields:
- Server: fileserver.example.com
- Share Name: Backups
- Username: domain\user or user@domain
- Password: Share access password
- Domain: Optional workgroup/domain

Example: \\fileserver\BackupShare
```

#### Azure Blob Storage
```
Required Fields:
- Account Name: mystorageaccount
- Container Name: backups
- Account Key: Azure storage account key

Example: sendense-storage/backups
```

## 📊 Monitoring & Health

### Status Indicators
- **🟢 Online**: Repository accessible and healthy
- **🟡 Warning**: Repository accessible but with issues (high usage, slow response)
- **🔴 Offline**: Repository inaccessible or failed

### Capacity Monitoring
- **Real-time Usage**: Current storage utilization percentage
- **Trend Analysis**: Usage patterns over time
- **Threshold Alerts**: Configurable warning levels (75%, 90%)
- **Growth Projections**: Estimated time to capacity

### Connection Health
- **Last Tested**: Timestamp of last connectivity check
- **Test Frequency**: Configurable testing intervals
- **Response Time**: Connection latency measurements
- **Error Tracking**: Historical failure patterns

## 🔄 Repository Operations

### Editing Repositories
1. Click **⋯** menu on repository card
2. Select **"Edit"**
3. Modify configuration in multi-step modal
4. Test connection if credentials changed
5. Save changes

### Testing Connections
1. Click **⋯** menu on repository card
2. Select **"Test Connection"**
3. View real-time test results
4. Review connection details and latency

### Deleting Repositories
1. Click **⋯** menu on repository card
2. Select **"Delete"**
3. Confirm deletion in dialog
4. Repository removed from system
5. Associated protection flows updated

## 🔗 Integration with Protection Flows

### Repository Selection
- Repositories available in protection flow configuration
- Type filtering based on flow requirements
- Health status considerations for selection

### Usage Tracking
- Storage consumption by protection flow
- Repository utilization analytics
- Cost allocation for cloud storage

### Failover Scenarios
- Automatic repository failover on failure
- Multi-repository load balancing
- Disaster recovery repository activation

## 🛠️ Administration

### Access Control
- Role-based permissions for repository management
- Audit logging for all repository operations
- Approval workflows for critical changes

### Maintenance Operations
- Repository defragmentation
- Capacity expansion procedures
- Backup and restore of repository configurations

## 📡 API Integration

### Backend Endpoints
All repository operations integrate with Phase 1 APIs:

```
POST   /api/v1/repositories           # Create repository
GET    /api/v1/repositories           # List repositories
GET    /api/v1/repositories/{id}      # Get repository details
GET    /api/v1/repositories/{id}/storage  # Capacity information
POST   /api/v1/repositories/test      # Test configuration
DELETE /api/v1/repositories/{id}      # Delete repository
```

### Real-time Updates
- WebSocket integration for status changes
- Polling fallback for compatibility
- Event-driven UI updates

## 🚨 Troubleshooting

### Common Issues

#### Connection Failures
- **Cause**: Network issues, credential problems, service outages
- **Solution**: Use "Test Connection" to diagnose
- **Prevention**: Regular automated testing

#### Capacity Issues
- **Cause**: Storage exhaustion, unexpected growth
- **Solution**: Monitor usage trends, implement alerts
- **Prevention**: Capacity planning and expansion

#### Performance Problems
- **Cause**: Network latency, concurrent access conflicts
- **Solution**: Load balancing, connection optimization
- **Prevention**: Performance monitoring and tuning

### Support Resources
- Built-in diagnostic tools
- Comprehensive logging and audit trails
- Integration with Sendense support systems

## 🎯 Best Practices

### Repository Planning
- Design for growth (20-30% headroom)
- Implement redundancy across repository types
- Regular capacity reviews and planning

### Security Considerations
- Use secure credential storage
- Implement access controls and auditing
- Regular credential rotation

### Performance Optimization
- Distribute load across multiple repositories
- Monitor and optimize network connections
- Implement caching where appropriate

### Monitoring Strategy
- Set up comprehensive alerting
- Regular health checks and reporting
- Trend analysis for capacity planning

---

## 📋 Quick Start Guide

1. **Access Interface**: Navigate to Repositories in sidebar
2. **Add Repository**: Click "Add Repository" button
3. **Choose Type**: Select appropriate storage type
4. **Configure**: Enter connection details and credentials
5. **Test**: Validate connection before saving
6. **Monitor**: Use dashboard for ongoing health monitoring

This interface provides enterprise-grade repository management with professional usability and comprehensive functionality for all backup storage needs.
