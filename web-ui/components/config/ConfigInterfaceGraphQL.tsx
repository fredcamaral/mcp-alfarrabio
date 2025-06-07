/**
 * Configuration Interface Component (GraphQL Version)
 * 
 * Provides UI for viewing and modifying system configuration.
 * Uses GraphQL instead of REST API.
 */

'use client'

import { useState, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Separator } from '@/components/ui/separator'
import { Slider } from '@/components/ui/slider'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Database,
  Shield,
  Globe,
  Zap,
  Activity,
  Save,
  RefreshCw,
  AlertTriangle,
  CheckCircle,
  Brain,
  Server,
  Eye,
  EyeOff
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { useGetConfig, useUpdateConfig } from '@/lib/graphql/hooks-extended'
import type { ConfigInput } from '@/lib/graphql/config-operations'
import { useAppDispatch } from '@/store/store'
import { addNotification } from '@/store/slices/uiSlice'

interface ConfigInterfaceProps {
  className?: string
  readOnly?: boolean
}

export function ConfigInterface({ className, readOnly = false }: ConfigInterfaceProps) {
  const dispatch = useAppDispatch()
  
  // GraphQL hooks
  const { data, loading: isLoading, error: queryError, refetch } = useGetConfig()
  const [updateConfig, { loading: isSaving }] = useUpdateConfig()
  
  const config = data?.getConfig
  
  // Local state for form
  const [formData, setFormData] = useState<ConfigInput>({})
  const [showApiKey, setShowApiKey] = useState(false)
  const [hasChanges, setHasChanges] = useState(false)

  // Initialize form data when config loads
  useEffect(() => {
    if (config) {
      setFormData({
        host: config.host,
        port: config.port,
        protocol: config.protocol,
        transportProtocols: { ...config.transportProtocols },
        vectorDb: { ...config.vectorDb },
        openai: { ...config.openai },
        features: { ...config.features }
      })
    }
  }, [config])

  const handleInputChange = (section: keyof ConfigInput, key: string, value: string | number | boolean) => {
    setFormData(prev => ({
      ...prev,
      [section]: section === 'host' || section === 'port' || section === 'protocol' 
        ? value 
        : {
            ...((prev[section] || {}) as Record<string, unknown>),
            [key]: value
          }
    }))
    setHasChanges(true)
  }

  const handleSave = async () => {
    try {
      const result = await updateConfig({
        variables: { input: formData }
      })
      
      if (result.data?.updateConfig) {
        dispatch(addNotification({
          type: 'success',
          title: 'Configuration Saved',
          message: 'System configuration has been updated successfully',
          duration: 5000
        }))
        setHasChanges(false)
        await refetch()
      }
    } catch (error) {
      dispatch(addNotification({
        type: 'error',
        title: 'Save Failed',
        message: error instanceof Error ? error.message : 'Failed to save configuration',
        duration: 5000
      }))
    }
  }

  const handleReset = () => {
    if (config) {
      setFormData({
        host: config.host,
        port: config.port,
        protocol: config.protocol,
        transportProtocols: { ...config.transportProtocols },
        vectorDb: { ...config.vectorDb },
        openai: { ...config.openai },
        features: { ...config.features }
      })
      setHasChanges(false)
    }
  }

  if (isLoading) {
    return (
      <div className={cn('space-y-6', className)}>
        <Card className="animate-pulse">
          <CardHeader>
            <div className="h-6 bg-muted rounded w-1/4" />
            <div className="h-4 bg-muted rounded w-1/2 mt-2" />
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div className="h-10 bg-muted rounded" />
              <div className="h-10 bg-muted rounded" />
              <div className="h-10 bg-muted rounded" />
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (queryError || !config) {
    return (
      <div className={cn('space-y-6', className)}>
        <Alert variant="destructive">
          <AlertTriangle className="h-4 w-4" />
          <AlertDescription>
            {queryError?.message || 'Failed to load configuration'}
          </AlertDescription>
        </Alert>
        <Button onClick={() => refetch()}>
          <RefreshCw className="h-4 w-4 mr-2" />
          Retry
        </Button>
      </div>
    )
  }

  return (
    <div className={cn('space-y-6', className)}>
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">System Configuration</h2>
          <p className="text-muted-foreground">
            Manage server settings, integrations, and feature flags
          </p>
        </div>
        
        <div className="flex gap-2">
          <Button 
            variant="outline" 
            onClick={handleReset}
            disabled={!hasChanges || readOnly}
          >
            <RefreshCw className="h-4 w-4 mr-2" />
            Reset
          </Button>
          <Button 
            onClick={handleSave}
            disabled={!hasChanges || isSaving || readOnly}
          >
            {isSaving ? (
              <>
                <RefreshCw className="h-4 w-4 mr-2 animate-spin" />
                Saving...
              </>
            ) : (
              <>
                <Save className="h-4 w-4 mr-2" />
                Save Changes
              </>
            )}
          </Button>
        </div>
      </div>

      {readOnly && (
        <Alert>
          <Shield className="h-4 w-4" />
          <AlertDescription>
            Configuration is in read-only mode
          </AlertDescription>
        </Alert>
      )}

      {/* Configuration Tabs */}
      <Tabs defaultValue="server" className="space-y-4">
        <TabsList className="grid w-full grid-cols-5">
          <TabsTrigger value="server">
            <Server className="h-4 w-4 mr-2" />
            Server
          </TabsTrigger>
          <TabsTrigger value="database">
            <Database className="h-4 w-4 mr-2" />
            Database
          </TabsTrigger>
          <TabsTrigger value="ai">
            <Brain className="h-4 w-4 mr-2" />
            AI
          </TabsTrigger>
          <TabsTrigger value="features">
            <Zap className="h-4 w-4 mr-2" />
            Features
          </TabsTrigger>
          <TabsTrigger value="monitoring">
            <Activity className="h-4 w-4 mr-2" />
            Monitoring
          </TabsTrigger>
        </TabsList>

        {/* Server Configuration */}
        <TabsContent value="server">
          <Card>
            <CardHeader>
              <CardTitle>Server Settings</CardTitle>
              <CardDescription>
                Configure the main server host, port, and transport protocols
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="host">Host</Label>
                  <Input
                    id="host"
                    value={formData.host || ''}
                    onChange={(e) => handleInputChange('host', '', e.target.value)}
                    placeholder="localhost"
                    disabled={readOnly}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="port">Port</Label>
                  <Input
                    id="port"
                    type="number"
                    value={formData.port || ''}
                    onChange={(e) => handleInputChange('port', '', parseInt(e.target.value))}
                    placeholder="9080"
                    disabled={readOnly}
                  />
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="protocol">Protocol</Label>
                <Select
                  value={formData.protocol || 'http'}
                  onValueChange={(value) => handleInputChange('protocol', '', value as 'http' | 'https')}
                  disabled={readOnly}
                >
                  <SelectTrigger id="protocol">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="http">HTTP</SelectItem>
                    <SelectItem value="https">HTTPS</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <Separator />

              <div className="space-y-4">
                <h4 className="text-sm font-medium">Transport Protocols</h4>
                <div className="space-y-3">
                  <div className="flex items-center justify-between">
                    <Label htmlFor="http-enabled" className="flex items-center gap-2">
                      <Globe className="h-4 w-4" />
                      HTTP/REST API
                    </Label>
                    <Switch
                      id="http-enabled"
                      checked={formData.transportProtocols?.http ?? true}
                      onCheckedChange={(checked) => handleInputChange('transportProtocols', 'http', checked)}
                      disabled={readOnly}
                    />
                  </div>
                  <div className="flex items-center justify-between">
                    <Label htmlFor="ws-enabled" className="flex items-center gap-2">
                      <Zap className="h-4 w-4" />
                      WebSocket
                    </Label>
                    <Switch
                      id="ws-enabled"
                      checked={formData.transportProtocols?.websocket ?? true}
                      onCheckedChange={(checked) => handleInputChange('transportProtocols', 'websocket', checked)}
                      disabled={readOnly}
                    />
                  </div>
                  <div className="flex items-center justify-between">
                    <Label htmlFor="grpc-enabled" className="flex items-center gap-2">
                      <Server className="h-4 w-4" />
                      gRPC
                    </Label>
                    <Switch
                      id="grpc-enabled"
                      checked={formData.transportProtocols?.grpc ?? false}
                      onCheckedChange={(checked) => handleInputChange('transportProtocols', 'grpc', checked)}
                      disabled={readOnly}
                    />
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Database Configuration */}
        <TabsContent value="database">
          <Card>
            <CardHeader>
              <CardTitle>Vector Database</CardTitle>
              <CardDescription>
                Configure the vector database for storing embeddings
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="space-y-2">
                <Label htmlFor="vector-provider">Provider</Label>
                <Select
                  value={formData.vectorDb?.provider || 'qdrant'}
                  onValueChange={(value) => handleInputChange('vectorDb', 'provider', value as 'qdrant' | 'chroma')}
                  disabled={readOnly}
                >
                  <SelectTrigger id="vector-provider">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="qdrant">Qdrant</SelectItem>
                    <SelectItem value="chroma">Chroma</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="vector-host">Host</Label>
                  <Input
                    id="vector-host"
                    value={formData.vectorDb?.host || ''}
                    onChange={(e) => handleInputChange('vectorDb', 'host', e.target.value)}
                    placeholder="localhost"
                    disabled={readOnly}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="vector-port">Port</Label>
                  <Input
                    id="vector-port"
                    type="number"
                    value={formData.vectorDb?.port || ''}
                    onChange={(e) => handleInputChange('vectorDb', 'port', parseInt(e.target.value))}
                    placeholder="6333"
                    disabled={readOnly}
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="vector-collection">Collection Name</Label>
                  <Input
                    id="vector-collection"
                    value={formData.vectorDb?.collection || ''}
                    onChange={(e) => handleInputChange('vectorDb', 'collection', e.target.value)}
                    placeholder="memories"
                    disabled={readOnly}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="vector-dimension">Vector Dimension</Label>
                  <Input
                    id="vector-dimension"
                    type="number"
                    value={formData.vectorDb?.dimension || ''}
                    onChange={(e) => handleInputChange('vectorDb', 'dimension', parseInt(e.target.value))}
                    placeholder="1536"
                    disabled={readOnly}
                  />
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* AI Configuration */}
        <TabsContent value="ai">
          <Card>
            <CardHeader>
              <CardTitle>OpenAI Integration</CardTitle>
              <CardDescription>
                Configure OpenAI API settings for embeddings and AI features
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="space-y-2">
                <Label htmlFor="openai-key">API Key</Label>
                <div className="flex gap-2">
                  <div className="relative flex-1">
                    <Input
                      id="openai-key"
                      type={showApiKey ? "text" : "password"}
                      value={formData.openai?.apiKey || ''}
                      onChange={(e) => handleInputChange('openai', 'apiKey', e.target.value)}
                      placeholder="sk-..."
                      disabled={readOnly}
                    />
                  </div>
                  <Button
                    type="button"
                    variant="outline"
                    size="icon"
                    onClick={() => setShowApiKey(!showApiKey)}
                  >
                    {showApiKey ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                  </Button>
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="openai-model">Model</Label>
                <Select
                  value={formData.openai?.model || 'text-embedding-ada-002'}
                  onValueChange={(value) => handleInputChange('openai', 'model', value)}
                  disabled={readOnly}
                >
                  <SelectTrigger id="openai-model">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="text-embedding-ada-002">text-embedding-ada-002</SelectItem>
                    <SelectItem value="text-embedding-3-small">text-embedding-3-small</SelectItem>
                    <SelectItem value="text-embedding-3-large">text-embedding-3-large</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label htmlFor="max-tokens">Max Tokens</Label>
                <Input
                  id="max-tokens"
                  type="number"
                  value={formData.openai?.maxTokens || ''}
                  onChange={(e) => handleInputChange('openai', 'maxTokens', parseInt(e.target.value))}
                  placeholder="8191"
                  disabled={readOnly}
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="temperature">Temperature</Label>
                <div className="flex items-center gap-4">
                  <Slider
                    id="temperature"
                    min={0}
                    max={2}
                    step={0.1}
                    value={[formData.openai?.temperature || 0]}
                    onValueChange={([value]) => handleInputChange('openai', 'temperature', value)}
                    disabled={readOnly}
                    className="flex-1"
                  />
                  <span className="w-12 text-sm text-muted-foreground">
                    {formData.openai?.temperature || 0}
                  </span>
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="timeout">Timeout (ms)</Label>
                <Input
                  id="timeout"
                  type="number"
                  value={formData.openai?.timeout || ''}
                  onChange={(e) => handleInputChange('openai', 'timeout', parseInt(e.target.value))}
                  placeholder="30000"
                  disabled={readOnly}
                />
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Features Configuration */}
        <TabsContent value="features">
          <Card>
            <CardHeader>
              <CardTitle>Feature Flags</CardTitle>
              <CardDescription>
                Enable or disable specific features and capabilities
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="flex items-center justify-between">
                  <Label htmlFor="csrf-enabled">CSRF Protection</Label>
                  <Switch
                    id="csrf-enabled"
                    checked={formData.features?.csrf ?? true}
                    onCheckedChange={(checked) => handleInputChange('features', 'csrf', checked)}
                    disabled={readOnly}
                  />
                </div>
                <div className="flex items-center justify-between">
                  <Label htmlFor="websocket-enabled">WebSocket Support</Label>
                  <Switch
                    id="websocket-enabled"
                    checked={formData.features?.websocket ?? true}
                    onCheckedChange={(checked) => handleInputChange('features', 'websocket', checked)}
                    disabled={readOnly}
                  />
                </div>
                <div className="flex items-center justify-between">
                  <Label htmlFor="graphql-enabled">GraphQL API</Label>
                  <Switch
                    id="graphql-enabled"
                    checked={formData.features?.graphql ?? true}
                    onCheckedChange={(checked) => handleInputChange('features', 'graphql', checked)}
                    disabled={readOnly}
                  />
                </div>
                <div className="flex items-center justify-between">
                  <Label htmlFor="monitoring-enabled">Monitoring</Label>
                  <Switch
                    id="monitoring-enabled"
                    checked={formData.features?.monitoring ?? true}
                    onCheckedChange={(checked) => handleInputChange('features', 'monitoring', checked)}
                    disabled={readOnly}
                  />
                </div>
                <div className="flex items-center justify-between">
                  <Label htmlFor="cache-enabled">Caching</Label>
                  <Switch
                    id="cache-enabled"
                    checked={formData.features?.cacheEnabled ?? true}
                    onCheckedChange={(checked) => handleInputChange('features', 'cacheEnabled', checked)}
                    disabled={readOnly}
                  />
                </div>
                <div className="flex items-center justify-between">
                  <Label htmlFor="realtime-enabled">Realtime Updates</Label>
                  <Switch
                    id="realtime-enabled"
                    checked={formData.features?.realtimeEnabled ?? true}
                    onCheckedChange={(checked) => handleInputChange('features', 'realtimeEnabled', checked)}
                    disabled={readOnly}
                  />
                </div>
                <div className="flex items-center justify-between">
                  <Label htmlFor="analytics-enabled">Analytics</Label>
                  <Switch
                    id="analytics-enabled"
                    checked={formData.features?.analyticsEnabled ?? true}
                    onCheckedChange={(checked) => handleInputChange('features', 'analyticsEnabled', checked)}
                    disabled={readOnly}
                  />
                </div>
                <div className="flex items-center justify-between">
                  <Label htmlFor="auth-enabled">Authentication</Label>
                  <Switch
                    id="auth-enabled"
                    checked={formData.features?.authEnabled ?? false}
                    onCheckedChange={(checked) => handleInputChange('features', 'authEnabled', checked)}
                    disabled={readOnly}
                  />
                </div>
              </div>

              <Separator />

              <div className="flex items-center justify-between p-3 rounded-lg bg-muted">
                <Label htmlFor="debug-mode" className="flex items-center gap-2">
                  <AlertTriangle className="h-4 w-4 text-yellow-600" />
                  Debug Mode
                </Label>
                <Switch
                  id="debug-mode"
                  checked={formData.features?.debugMode ?? false}
                  onCheckedChange={(checked) => handleInputChange('features', 'debugMode', checked)}
                  disabled={readOnly}
                />
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Monitoring Configuration */}
        <TabsContent value="monitoring">
          <Card>
            <CardHeader>
              <CardTitle>System Monitoring</CardTitle>
              <CardDescription>
                View current system status and performance metrics
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="grid grid-cols-2 gap-4">
                <Card>
                  <CardContent className="pt-6">
                    <div className="flex items-center justify-between">
                      <div>
                        <p className="text-sm text-muted-foreground">Status</p>
                        <p className="text-2xl font-bold">Operational</p>
                      </div>
                      <CheckCircle className="h-8 w-8 text-green-600" />
                    </div>
                  </CardContent>
                </Card>
                
                <Card>
                  <CardContent className="pt-6">
                    <div className="flex items-center justify-between">
                      <div>
                        <p className="text-sm text-muted-foreground">Active Features</p>
                        <p className="text-2xl font-bold">
                          {Object.values(formData.features || {}).filter(v => v).length}
                        </p>
                      </div>
                      <Zap className="h-8 w-8 text-blue-600" />
                    </div>
                  </CardContent>
                </Card>
              </div>

              <Alert>
                <Activity className="h-4 w-4" />
                <AlertDescription>
                  For detailed performance metrics and monitoring, use the dedicated monitoring dashboard.
                </AlertDescription>
              </Alert>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}