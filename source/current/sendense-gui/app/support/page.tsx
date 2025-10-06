"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { PageHeader } from "@/components/common/PageHeader";
import {
  HelpCircle,
  FileText,
  MessageSquare,
  Phone,
  Mail,
  ExternalLink,
  Book,
  AlertCircle,
  CheckCircle,
  Clock,
  Users,
  Shield,
  Database,
  Zap,
  LifeBuoy,
  Download
} from "lucide-react";
import { useState } from "react";

interface SupportResource {
  id: string;
  title: string;
  description: string;
  type: 'documentation' | 'guide' | 'video' | 'faq';
  category: 'getting-started' | 'backup' | 'restore' | 'troubleshooting' | 'api';
  url?: string;
  icon: React.ReactNode;
}

interface SystemInfo {
  version: string;
  buildDate: string;
  uptime: string;
  license: string;
  supportLevel: string;
}

const supportResources: SupportResource[] = [
  {
    id: '1',
    title: 'Getting Started Guide',
    description: 'Complete guide for setting up Sendense and your first backup',
    type: 'guide',
    category: 'getting-started',
    url: '/docs/getting-started',
    icon: <Book className="h-5 w-5 text-blue-500" />
  },
  {
    id: '2',
    title: 'Backup Configuration',
    description: 'Learn how to configure backup policies and schedules',
    type: 'documentation',
    category: 'backup',
    url: '/docs/backup-configuration',
    icon: <Database className="h-5 w-5 text-green-500" />
  },
  {
    id: '3',
    title: 'Restore Operations',
    description: 'Step-by-step guide for restoring data and VMs',
    type: 'guide',
    category: 'restore',
    url: '/docs/restore-operations',
    icon: <Zap className="h-5 w-5 text-orange-500" />
  },
  {
    id: '4',
    title: 'Troubleshooting Common Issues',
    description: 'Solutions for frequently encountered problems',
    type: 'documentation',
    category: 'troubleshooting',
    url: '/docs/troubleshooting',
    icon: <AlertCircle className="h-5 w-5 text-red-500" />
  },
  {
    id: '5',
    title: 'API Reference',
    description: 'Complete API documentation for integrations',
    type: 'documentation',
    category: 'api',
    url: '/docs/api-reference',
    icon: <FileText className="h-5 w-5 text-purple-500" />
  },
  {
    id: '6',
    title: 'Video Tutorials',
    description: 'Watch video guides for key Sendense features',
    type: 'video',
    category: 'getting-started',
    url: '/videos',
    icon: <CheckCircle className="h-5 w-5 text-green-500" />
  },
  {
    id: '7',
    title: 'Frequently Asked Questions',
    description: 'Answers to common questions and issues',
    type: 'faq',
    category: 'troubleshooting',
    url: '/faq',
    icon: <HelpCircle className="h-5 w-5 text-blue-500" />
  },
  {
    id: '8',
    title: 'Security Best Practices',
    description: 'Learn about security features and best practices',
    type: 'guide',
    category: 'backup',
    url: '/docs/security',
    icon: <Shield className="h-5 w-5 text-red-500" />
  }
];

const mockSystemInfo: SystemInfo = {
  version: '2.1.0',
  buildDate: '2025-10-06',
  uptime: '7 days, 14 hours',
  license: 'Enterprise',
  supportLevel: '24/7 Premium Support'
};

export default function SupportPage() {
  const [selectedCategory, setSelectedCategory] = useState<string>('all');
  const [searchQuery, setSearchQuery] = useState('');
  const [isContactModalOpen, setIsContactModalOpen] = useState(false);
  const [contactForm, setContactForm] = useState({
    name: '',
    email: '',
    subject: '',
    priority: 'medium',
    message: ''
  });

  const handleContactFormChange = (field: string, value: string) => {
    setContactForm(prev => ({ ...prev, [field]: value }));
  };

  const handleSubmitSupportRequest = () => {
    // Simulate support request submission
    console.log('Support request submitted:', contactForm);
    setContactForm({ name: '', email: '', subject: '', priority: 'medium', message: '' });
    setIsContactModalOpen(false);
  };

  const filteredResources = supportResources.filter(resource => {
    const matchesCategory = selectedCategory === 'all' || resource.category === selectedCategory;
    const matchesSearch = searchQuery === '' ||
      resource.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
      resource.description.toLowerCase().includes(searchQuery.toLowerCase());
    return matchesCategory && matchesSearch;
  });

  const categories = [
    { id: 'all', label: 'All Resources', count: supportResources.length },
    { id: 'getting-started', label: 'Getting Started', count: supportResources.filter(r => r.category === 'getting-started').length },
    { id: 'backup', label: 'Backup', count: supportResources.filter(r => r.category === 'backup').length },
    { id: 'restore', label: 'Restore', count: supportResources.filter(r => r.category === 'restore').length },
    { id: 'troubleshooting', label: 'Troubleshooting', count: supportResources.filter(r => r.category === 'troubleshooting').length },
    { id: 'api', label: 'API', count: supportResources.filter(r => r.category === 'api').length }
  ];

  const getTypeBadge = (type: string) => {
    const variants = {
      documentation: 'bg-blue-500/10 text-blue-400 border-blue-500/20',
      guide: 'bg-green-500/10 text-green-400 border-green-500/20',
      video: 'bg-purple-500/10 text-purple-400 border-purple-500/20',
      faq: 'bg-orange-500/10 text-orange-400 border-orange-500/20'
    };
    return (
      <Badge className={variants[type as keyof typeof variants] || 'bg-gray-500/10 text-gray-400 border-gray-500/20'}>
        {type.charAt(0).toUpperCase() + type.slice(1)}
      </Badge>
    );
  };

  const getPriorityBadge = (priority: string) => {
    const variants = {
      low: 'bg-gray-500/10 text-gray-400 border-gray-500/20',
      medium: 'bg-yellow-500/10 text-yellow-400 border-yellow-500/20',
      high: 'bg-red-500/10 text-red-400 border-red-500/20',
      urgent: 'bg-red-600/10 text-red-400 border-red-600/20'
    };
    return (
      <Badge className={variants[priority as keyof typeof variants] || variants.medium}>
        {priority.charAt(0).toUpperCase() + priority.slice(1)}
      </Badge>
    );
  };

  return (
    <div className="h-full flex flex-col">
      <PageHeader
        title="Support & Documentation"
        breadcrumbs={[
          { label: "Dashboard", href: "/dashboard" },
          { label: "Support" }
        ]}
        actions={
          <div className="flex gap-2">
            <Button
              variant="outline"
              onClick={() => setIsContactModalOpen(true)}
              className="gap-2"
            >
              <MessageSquare className="h-4 w-4" />
              Contact Support
            </Button>
            <Button className="gap-2">
              <Download className="h-4 w-4" />
              Download Logs
            </Button>
          </div>
        }
      />

      <div className="flex-1 overflow-auto">
        <div className="p-6">
          <div className="mb-6">
            <h2 className="text-lg font-semibold text-foreground mb-2">
              Support Resources
            </h2>
            <p className="text-muted-foreground">
              Find documentation, guides, and get help with your Sendense deployment.
            </p>
          </div>

          {/* System Information */}
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Version</CardTitle>
                <FileText className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{mockSystemInfo.version}</div>
                <p className="text-xs text-muted-foreground">
                  Build: {mockSystemInfo.buildDate}
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Uptime</CardTitle>
                <Clock className="h-4 w-4 text-green-500" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold text-green-600">{mockSystemInfo.uptime}</div>
                <p className="text-xs text-muted-foreground">
                  System operational
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">License</CardTitle>
                <Shield className="h-4 w-4 text-blue-500" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{mockSystemInfo.license}</div>
                <p className="text-xs text-muted-foreground">
                  {mockSystemInfo.supportLevel}
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Status</CardTitle>
                <LifeBuoy className="h-4 w-4 text-green-500" />
              </CardHeader>
              <CardContent>
                <div className="flex items-center gap-2">
                  <CheckCircle className="h-5 w-5 text-green-500" />
                  <span className="text-lg font-semibold">Healthy</span>
                </div>
                <p className="text-xs text-muted-foreground">
                  All systems operational
                </p>
              </CardContent>
            </Card>
          </div>

          {/* Quick Actions */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
            <Card className="cursor-pointer hover:shadow-md transition-shadow" onClick={() => setIsContactModalOpen(true)}>
              <CardContent className="flex items-center p-6">
                <div className="w-12 h-12 rounded-lg bg-blue-500/10 flex items-center justify-center mr-4">
                  <MessageSquare className="h-6 w-6 text-blue-500" />
                </div>
                <div>
                  <h3 className="font-semibold text-foreground">Contact Support</h3>
                  <p className="text-sm text-muted-foreground">Get help from our support team</p>
                </div>
              </CardContent>
            </Card>

            <Card className="cursor-pointer hover:shadow-md transition-shadow">
              <CardContent className="flex items-center p-6">
                <div className="w-12 h-12 rounded-lg bg-green-500/10 flex items-center justify-center mr-4">
                  <Phone className="h-6 w-6 text-green-500" />
                </div>
                <div>
                  <h3 className="font-semibold text-foreground">Live Chat</h3>
                  <p className="text-sm text-muted-foreground">Chat with support specialists</p>
                </div>
              </CardContent>
            </Card>

            <Card className="cursor-pointer hover:shadow-md transition-shadow">
              <CardContent className="flex items-center p-6">
                <div className="w-12 h-12 rounded-lg bg-purple-500/10 flex items-center justify-center mr-4">
                  <Users className="h-6 w-6 text-purple-500" />
                </div>
                <div>
                  <h3 className="font-semibold text-foreground">Community</h3>
                  <p className="text-sm text-muted-foreground">Join the Sendense community</p>
                </div>
              </CardContent>
            </Card>
          </div>

          {/* Documentation Search and Filter */}
          <div className="mb-6">
            <div className="flex flex-col sm:flex-row gap-4">
              <div className="flex-1">
                <Input
                  placeholder="Search documentation..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="w-full"
                />
              </div>
              <div className="flex gap-2 overflow-x-auto">
                {categories.map((category) => (
                  <Button
                    key={category.id}
                    variant={selectedCategory === category.id ? "default" : "outline"}
                    size="sm"
                    onClick={() => setSelectedCategory(category.id)}
                    className="whitespace-nowrap"
                  >
                    {category.label} ({category.count})
                  </Button>
                ))}
              </div>
            </div>
          </div>

          {/* Documentation Resources */}
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {filteredResources.map((resource) => (
              <Card key={resource.id} className="hover:shadow-md transition-shadow">
                <CardHeader className="pb-3">
                  <div className="flex items-start justify-between">
                    <div className="flex items-center gap-3">
                      {resource.icon}
                      <div>
                        <CardTitle className="text-lg">{resource.title}</CardTitle>
                        <div className="flex items-center gap-2 mt-1">
                          {getTypeBadge(resource.type)}
                          <span className="text-xs text-muted-foreground capitalize">
                            {resource.category.replace('-', ' ')}
                          </span>
                        </div>
                      </div>
                    </div>
                  </div>
                </CardHeader>

                <CardContent>
                  <p className="text-sm text-muted-foreground mb-4">
                    {resource.description}
                  </p>

                  {resource.url && (
                    <Button variant="outline" size="sm" className="w-full gap-2">
                      <ExternalLink className="h-4 w-4" />
                      View Documentation
                    </Button>
                  )}
                </CardContent>
              </Card>
            ))}
          </div>

          {filteredResources.length === 0 && (
            <div className="text-center py-12">
              <HelpCircle className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
              <h3 className="text-lg font-medium text-foreground mb-2">No resources found</h3>
              <p className="text-muted-foreground">
                Try adjusting your search or filter criteria.
              </p>
            </div>
          )}
        </div>
      </div>

      {/* Contact Support Modal */}
      <Dialog open={isContactModalOpen} onOpenChange={setIsContactModalOpen}>
        <DialogContent className="sm:max-w-[600px]">
          <DialogHeader>
            <DialogTitle>Contact Support</DialogTitle>
            <DialogDescription>
              Submit a support request and our team will get back to you shortly.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="contact-name">Your Name</Label>
                <Input
                  id="contact-name"
                  placeholder="John Doe"
                  value={contactForm.name}
                  onChange={(e) => handleContactFormChange('name', e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="contact-email">Email Address</Label>
                <Input
                  id="contact-email"
                  type="email"
                  placeholder="john.doe@company.com"
                  value={contactForm.email}
                  onChange={(e) => handleContactFormChange('email', e.target.value)}
                />
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="contact-subject">Subject</Label>
              <Input
                id="contact-subject"
                placeholder="Brief description of your issue"
                value={contactForm.subject}
                onChange={(e) => handleContactFormChange('subject', e.target.value)}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="contact-priority">Priority</Label>
              <div className="flex gap-2">
                {['low', 'medium', 'high', 'urgent'].map((priority) => (
                  <Button
                    key={priority}
                    type="button"
                    variant={contactForm.priority === priority ? "default" : "outline"}
                    size="sm"
                    onClick={() => handleContactFormChange('priority', priority)}
                    className="capitalize"
                  >
                    {getPriorityBadge(priority)}
                  </Button>
                ))}
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="contact-message">Message</Label>
              <Textarea
                id="contact-message"
                placeholder="Please describe your issue in detail..."
                rows={4}
                value={contactForm.message}
                onChange={(e) => handleContactFormChange('message', e.target.value)}
              />
            </div>

            <div className="bg-muted/50 p-4 rounded-lg">
              <h4 className="font-medium text-foreground mb-2">System Information</h4>
              <div className="grid grid-cols-2 gap-4 text-sm">
                <div>
                  <span className="text-muted-foreground">Version:</span>
                  <span className="ml-2 font-medium">{mockSystemInfo.version}</span>
                </div>
                <div>
                  <span className="text-muted-foreground">License:</span>
                  <span className="ml-2 font-medium">{mockSystemInfo.license}</span>
                </div>
                <div>
                  <span className="text-muted-foreground">Uptime:</span>
                  <span className="ml-2 font-medium">{mockSystemInfo.uptime}</span>
                </div>
                <div>
                  <span className="text-muted-foreground">Support:</span>
                  <span className="ml-2 font-medium">{mockSystemInfo.supportLevel}</span>
                </div>
              </div>
            </div>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setIsContactModalOpen(false)}>
              Cancel
            </Button>
            <Button
              type="button"
              onClick={handleSubmitSupportRequest}
              disabled={!contactForm.name || !contactForm.email || !contactForm.subject || !contactForm.message}
            >
              Submit Request
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
