import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Production optimizations
  compress: true, // Enable gzip compression
  poweredByHeader: false, // Remove X-Powered-By header for security

  // Bundle optimization
  experimental: {
    optimizePackageImports: ['lucide-react', 'date-fns', 'recharts'],
  },

  // Image optimization
  images: {
    formats: ['image/webp', 'image/avif'],
    minimumCacheTTL: 60,
  },

  // Performance optimizations
  // swcMinify is enabled by default in Next.js 15

  // Build optimizations
  // output: 'standalone', // Enable standalone output for deployment (commented out for now)

  // API Proxy to SHA backend
  // NOTE: rewrites have limited timeout control, use custom API routes for long operations
  async rewrites() {
    return [
      {
        source: '/api/v1/:path*',
        destination: 'http://localhost:8082/api/v1/:path*',
      },
    ];
  },

  // Security headers
  async headers() {
    return [
      {
        source: '/(.*)',
        headers: [
          {
            key: 'X-Frame-Options',
            value: 'DENY',
          },
          {
            key: 'X-Content-Type-Options',
            value: 'nosniff',
          },
          {
            key: 'Referrer-Policy',
            value: 'strict-origin-when-cross-origin',
          },
        ],
      },
    ];
  },
};

export default nextConfig;
