import type { Metadata, Viewport } from 'next'
import Link from 'next/link'

export const metadata: Metadata = {
    title: '404 - Page Not Found | MCP Memory',
    description: 'The page you are looking for could not be found.',
}

export const viewport: Viewport = {
    width: 'device-width',
    initialScale: 1,
}

export default function NotFound() {
    return (
        <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-background via-purple-900/20 to-background">
            <div className="text-center space-y-6 p-8">
                <div className="space-y-2">
                    <h1 className="text-6xl font-bold text-white">404</h1>
                    <h2 className="text-2xl font-semibold text-purple">Page Not Found</h2>
                    <p className="text-muted-foreground max-w-md mx-auto">
                        The page you are looking for might have been removed, had its name changed, or is temporarily unavailable.
                    </p>
                </div>

                <div className="space-y-4">
                    <Link
                        href="/"
                        className="inline-flex items-center px-6 py-3 bg-purple hover:bg-purple/90 text-white font-medium rounded-lg transition-colors duration-200"
                    >
                        Return Home
                    </Link>

                    <div className="text-sm text-muted-foreground">
                        <Link href="/" className="hover:text-purple transition-colors">
                            Go back to MCP Memory Dashboard
                        </Link>
                    </div>
                </div>
            </div>
        </div>
    )
} 