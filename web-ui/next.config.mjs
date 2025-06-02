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
    unoptimized: true,
  },
  experimental: {
    optimizePackageImports: ["lucide-react"],
  },
  output: 'standalone',
  // Configure API proxy for backend communication
  async rewrites() {
    return [
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
    config.resolve.alias = {
      ...config.resolve.alias,
      '@': path.resolve('./'),
    }
    return config
  },
};

export default nextConfig;