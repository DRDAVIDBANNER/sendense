# MigrateKit CloudStack - Web GUI Dashboard

## 🎯 **Overview**

The MigrateKit CloudStack Web GUI provides a professional, user-friendly interface for managing VMware to CloudStack migrations. Built with Next.js and Flowbite components, it offers real-time VM discovery, migration initiation, and job monitoring.

## 🌐 **Access Information**

- **URL**: `http://10.245.246.125:3001`
- **Network**: Accessible from OMA appliance (CloudStack side)
- **Technology**: Next.js 15.4.5 with TypeScript and Tailwind CSS
- **Design System**: Flowbite React components for professional admin interface

## 🚀 **Features**

### **VM Discovery**
- Interactive discovery of VMs from vCenter
- Real-time display of VM information:
  - VM Name and Path
  - Power State (poweredOn/poweredOff)
  - Guest OS Type
  - CPU and Memory allocation
- Automatic detection via VMA API tunnel integration

### **Migration Management**
- One-click migration initiation from discovered VMs
- Real-time job tracking and status updates
- Migration history with timestamps
- Status indicators (started, running, completed, failed)

### **Dashboard Components**
- **Summary Cards**: VM count, migration status overview
- **VM Table**: Sortable, responsive table of discovered VMs
- **Migration Monitor**: Active job tracking with progress indicators
- **Alert System**: Error handling and user notifications

## 🔧 **Technical Implementation**

### **Architecture**
```
Browser → Next.js GUI (Port 3001) → API Routes → VMA API (Port 9081) → SSH Tunnel → VMA
```

### **API Integration**
- **VM Discovery**: `POST /api/discover` → VMA `/api/v1/discover`
- **Migration Start**: `POST /api/replicate` → VMA `/api/v1/replicate`
- **Network**: All API calls go through SSH tunnel (localhost:9081)

### **File Structure**
```
~/migration-dashboard/
├── src/app/
│   ├── page.tsx              # Main dashboard component
│   ├── layout.tsx            # Root layout with metadata
│   └── api/
│       ├── discover/route.ts # VM discovery API route
│       └── replicate/route.ts# Migration start API route
├── package.json              # Dependencies and scripts
└── tailwind.config.ts        # Tailwind + Flowbite configuration
```

## 📋 **Usage Workflow**

### **1. VM Discovery**
1. Click "Discover VMs" button on dashboard
2. System queries vCenter via VMA API
3. Results populate in interactive table
4. View VM details, power state, and resources

### **2. Migration Initiation**
1. Select VM from discovery table
2. Click "Migrate" button next to desired VM
3. System creates migration job with dynamic port allocation
4. Job appears in "Active Migrations" section

### **3. Monitoring**
1. Track migration progress in real-time
2. View job status and timestamps
3. Monitor multiple concurrent migrations
4. Receive alerts for any issues

## 🛠️ **Development & Deployment**

### **Development Server**
```bash
# On OMA appliance
cd ~/migration-dashboard
npm run dev
```

### **Production Build**
```bash
# For production deployment
npm run build
npm start
```

### **Dependencies**
- **Next.js**: 15.4.5 (React framework)
- **TypeScript**: Type safety and development tools
- **Tailwind CSS**: Utility-first CSS framework
- **Flowbite React**: Professional UI components
- **React Icons**: Icon library for interface elements

## 🔒 **Security & Network**

### **Network Compliance**
- All API communication via SSH tunnel (ports 22, 80, 443 only)
- No direct connections between appliances outside tunnel
- Credentials handled securely through API endpoints

### **Authentication**
- Currently uses VMA API authentication
- GUI acts as proxy to authenticated VMA endpoints
- Future: Could add GUI-level authentication layer

## 📊 **Performance & Monitoring**

### **Response Times** (from logs)
- Dashboard load: ~156ms (cached), ~6s (initial)
- VM Discovery: ~649-1177ms
- Migration start: ~255-504ms

### **Real-time Updates**
- Migration status updates via API polling
- VM discovery refresh on demand
- Error state management with user notifications

## 🎨 **User Interface**

### **Design Features**
- **Responsive Design**: Works on desktop, tablet, and mobile
- **Dark Mode Support**: Automatic theme detection
- **Professional Layout**: Clean, modern admin interface
- **Status Indicators**: Color-coded badges for states
- **Loading States**: User feedback during operations

### **Key Components**
- **Header**: Project branding and description
- **Action Cards**: Quick access to main functions
- **Data Tables**: Sortable, filterable VM and job lists
- **Modal Dialogs**: Confirmation and detail views
- **Alert System**: Success, warning, and error messages

## 🚀 **Current Status**

✅ **Production Ready**
- GUI fully operational on OMA appliance
- All API endpoints responding successfully
- Real VM discovery and migration working
- Professional interface with real-time updates
- Network compliant (tunnel-only communication)

## 🔄 **Future Enhancements**

### **Planned Features**
- Real-time progress bars for active migrations
- Migration history and logs viewer
- Bulk migration operations
- Advanced filtering and search
- Performance metrics dashboard
- Export functionality for reports

### **Technical Improvements**
- WebSocket integration for real-time updates
- GUI-level authentication and user management
- Advanced error handling and retry logic
- Automated testing suite
- Production deployment optimization