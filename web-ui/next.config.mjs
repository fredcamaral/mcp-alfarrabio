import path from 'path'

/** @type {import('next').NextConfig} */
const nextConfig = {
  eslint: {
    ignoreDuringBuilds: true,
  },
  typescript: {
    ignoreBuildErrors: false,
  },
  images: {
    formats: ['image/webp', 'image/avif'],
    minimumCacheTTL: 31536000, // 1 year
    dangerouslyAllowSVG: true,
    contentSecurityPolicy: "default-src 'self'; script-src 'none'; sandbox;",
    deviceSizes: [640, 768, 1024, 1280, 1536],
    imageSizes: [16, 32, 48, 64, 96, 128, 256, 384],
    domains: ['localhost'],
    unoptimized: false,
  },
  experimental: {
    optimizePackageImports: [
      "lucide-react", 
      "@radix-ui/react-icons",
      "recharts",
      "@apollo/client"
    ],
    webVitalsAttribution: ['CLS', 'LCP'],
  },
  
  turbopack: {
    rules: {
      '*.svg': {
        loaders: ['@svgr/webpack'],
        as: '*.js',
      },
    },
  },
  output: 'standalone',
  
  // Performance optimizations
  compiler: {
    removeConsole: process.env.NODE_ENV === 'production',
    reactRemoveProperties: process.env.NODE_ENV === 'production' ? { properties: ['^data-testid$'] } : false,
  },
  
  // Enable gzip compression
  compress: true,
  
  // Enable HTTP/2 Server Push
  generateEtags: true,
  
  // PoweredByHeader removal for security
  poweredByHeader: false,
  
  // Security and performance headers
  async headers() {
    return [
      {
        source: '/:path*',
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
            key: 'X-XSS-Protection',
            value: '1; mode=block',
          },
          {
            key: 'Referrer-Policy',
            value: 'strict-origin-when-cross-origin',
          },
          {
            key: 'Permissions-Policy',
            value: 'camera=(), microphone=(), geolocation=()',
          },
          {
            key: 'Strict-Transport-Security',
            value: 'max-age=31536000; includeSubDomains',
          },
          {
            key: 'Content-Security-Policy',
            value: process.env.NODE_ENV === 'production' && !process.env.LOCAL_DEVELOPMENT
              ? "default-src 'self'; script-src 'self' 'sha256-LcsuUMiDkprrt6ZKeiLP4iYNhWo8NqaSbAgtoZxVK3s=' 'sha256-OBTN3RiyCV4Bq7dFqZ5a2pAXjnCcCYeTJMO2I/LYKeo=' 'sha256-XnU4G9MTL+3AsBi+wwOj8Fkq3KjHluQ9X1uRJrNRf1A=' 'sha256-h64Lp6xl0iJlckc7RXMBm5s70CDl892D31s57jZ9u5U=' 'sha256-aEp63qGzkw08+kWjGhdSyJ8I4r83qMlrJLNLKNgMWPU=' 'sha256-KYqDutyyohQ+tqNeLlo8lDTXLfftBZr/5WNmbpS+jUk=' 'sha256-KyCkp2vQnctSGJBakfuL8C2LK3aA+CHAlv4zQcH6q34=' 'sha256-8tKyeSNcV38Pqhc9/pSas6RDpvaiuao7kYHZGgOAksc='; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'; connect-src 'self' http://localhost:* ws://localhost:* wss://localhost:* ws: wss: http: https:; frame-ancestors 'none'; base-uri 'self'; form-action 'self';"
              : "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'; connect-src 'self' ws: wss: http: https:; frame-ancestors 'none'; base-uri 'self'; form-action 'self';",
          },
        ],
      },
      {
        source: '/static/(.*)',
        headers: [
          {
            key: 'Cache-Control',
            value: 'public, max-age=31536000, immutable',
          },
        ],
      },
      {
        source: '/_next/static/(.*)',
        headers: [
          {
            key: 'Cache-Control',
            value: 'public, max-age=31536000, immutable',
          },
        ],
      },
      {
        source: '/assets/:path*',
        headers: [
          {
            key: 'Cache-Control',
            value: 'public, max-age=31536000, immutable',
          },
        ],
      },
    ]
  },
  
  // Configure API proxy for backend communication
  async rewrites() {
    return [
      {
        source: '/api/mcp',
        destination: `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:9080'}/mcp`,
      },
      {
        source: '/api/:path*',
        destination: `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:9080'}/:path*`,
      },
      {
        source: '/graphql',
        destination: `${process.env.NEXT_PUBLIC_GRAPHQL_URL || 'http://localhost:9080/graphql'}`,
      }
    ]
  },
  webpack: (config, { buildId, dev, isServer, defaultLoaders, webpack }) => {
    // Alias configuration
    config.resolve.alias = {
      ...config.resolve.alias,
      '@': path.resolve('./'),
    }

    // Bundle splitting optimization
    if (!isServer) {
      config.optimization.splitChunks = {
        chunks: 'all',
        cacheGroups: {
          default: false,
          vendors: false,
          // Vendor chunk for third-party libraries
          vendor: {
            name: 'vendor',
            chunks: 'all',
            test: /node_modules/,
            priority: 20,
          },
          // Common chunk for shared code
          common: {
            name: 'common',
            minChunks: 2,
            chunks: 'all',
            priority: 10,
            reuseExistingChunk: true,
            enforce: true,
          },
          // UI libraries chunk
          ui: {
            name: 'ui',
            test: /[\\/]node_modules[\\/](@radix-ui|lucide-react|recharts)[\\/]/,
            chunks: 'all',
            priority: 30,
          },
          // Apollo/GraphQL chunk
          apollo: {
            name: 'apollo',
            test: /[\\/]node_modules[\\/](@apollo|graphql)[\\/]/,
            chunks: 'all',
            priority: 30,
          },
        },
      }
    }

    // Tree shaking improvements
    config.optimization.usedExports = true
    config.optimization.sideEffects = false

    // Minimize bundle size
    if (!dev) {
      config.optimization.minimize = true
    }

    return config
  },
};

export default nextConfig;