// Jest setup file for testing environment

// Mock environment variables
process.env.NODE_ENV = 'test'
process.env.NEXT_PUBLIC_API_URL = 'http://localhost:9080'
process.env.NEXT_PUBLIC_GRAPHQL_URL = 'http://localhost:9080/graphql'
process.env.NEXT_PUBLIC_WS_URL = 'ws://localhost:9080/ws'

// Global test utilities
global.mockFetch = jest.fn()
global.fetch = global.mockFetch