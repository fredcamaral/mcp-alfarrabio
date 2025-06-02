'use client'

import { Provider } from 'react-redux'
import { ApolloProvider } from '@apollo/client'
import { store } from '@/store/store'
import { apolloClient } from '@/lib/apollo-client'
import { CSRFProvider } from './CSRFProvider'

interface ProvidersProps {
  children: React.ReactNode
}

export function Providers({ children }: ProvidersProps) {
  return (
    <Provider store={store}>
      <ApolloProvider client={apolloClient}>
        <CSRFProvider>
          {children}
        </CSRFProvider>
      </ApolloProvider>
    </Provider>
  )
}