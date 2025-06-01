'use client'

import { Provider } from 'react-redux'
import { ApolloProvider } from '@apollo/client'
import { store } from '@/store/store'
import { apolloClient } from '@/lib/apollo-client'

interface ProvidersProps {
  children: React.ReactNode
}

export function Providers({ children }: ProvidersProps) {
  return (
    <Provider store={store}>
      <ApolloProvider client={apolloClient}>
        {children}
      </ApolloProvider>
    </Provider>
  )
}