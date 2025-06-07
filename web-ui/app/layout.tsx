import type { Metadata, Viewport } from 'next'
import { Inter } from 'next/font/google'
import './globals.css'
import '@/styles/design-system.css'
import { Providers } from '@/providers/Providers'
import { SkipToMain } from '@/lib/accessibility'
import '@/lib/startup-validation' // Run startup validation

const inter = Inter({ 
  subsets: ['latin'],
  display: 'swap',
  preload: true,
})

export const metadata: Metadata = {
  title: 'MCP Memory - AI Memory Management System',
  description: 'Persistent memory capabilities for AI assistants using Model Context Protocol',
  keywords: ['MCP', 'Memory', 'AI', 'Assistant', 'Context', 'Protocol'],
  authors: [{ name: 'MCP Memory Team' }],
  manifest: '/manifest.json',
  robots: {
    index: true,
    follow: true,
  },
  openGraph: {
    title: 'MCP Memory - AI Memory Management System',
    description: 'Persistent memory capabilities for AI assistants using Model Context Protocol',
    type: 'website',
    locale: 'en_US',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'MCP Memory - AI Memory Management System',
    description: 'Persistent memory capabilities for AI assistants using Model Context Protocol',
  },
}

export const viewport: Viewport = {
  width: 'device-width',
  initialScale: 1,
  maximumScale: 5,
  userScalable: true,
  themeColor: [
    { media: '(prefers-color-scheme: light)', color: '#ffffff' },
    { media: '(prefers-color-scheme: dark)', color: '#000000' },
  ],
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body className={inter.className}>
        <SkipToMain />
        <Providers>
          <main id="main-content">
            {children}
          </main>
        </Providers>
      </body>
    </html>
  )
}