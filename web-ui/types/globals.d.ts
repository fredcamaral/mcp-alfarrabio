declare global {
  namespace NodeJS {
    interface ProcessEnv {
      NODE_ENV: 'development' | 'production' | 'test'
      NEXT_PUBLIC_API_URL?: string
      NEXT_PUBLIC_WS_URL?: string
      NEXT_PUBLIC_GRAPHQL_URL?: string
      [key: string]: string | undefined
    }
  }

  const process: {
    env: NodeJS.ProcessEnv
  }

  const require: (id: string) => any
}

export {}