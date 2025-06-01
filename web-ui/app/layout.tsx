import type { Metadata } from 'next'
import { Inter } from 'next/font/google'
import './globals.css'
import { Providers } from '@/providers/Providers'

const inter = Inter({ subsets: ['latin'] })

export const metadata: Metadata = {
  title: 'MCP Memory - AI Memory Management System',
  description: 'Persistent memory capabilities for AI assistants using Model Context Protocol',
  keywords: ['MCP', 'Memory', 'AI', 'Assistant', 'Context', 'Protocol'],
  authors: [{ name: 'MCP Memory Team' }],
  viewport: 'width=device-width, initial-scale=1',
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body className={inter.className}>
        <Providers>
          {children}
        </Providers>
      </body>
    </html>
  )
}