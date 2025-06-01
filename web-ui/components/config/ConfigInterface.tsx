'use client'

import { useState } from 'react'
import { useAppSelector, useAppDispatch } from '@/store/store'
import {
  selectConfig,
  selectConfigLoading,
  selectHasUnsavedChanges,
  selectConnectionStatus,
  selectConfigHistory,
  selectValidationErrors,
  updateServerConfig,
  updateVectorDbConfig,
  updateOpenAIConfig,
  updateTransportProtocols,
  toggleFeature,
  markSaved,
  resetConfig,
  restoreFromHistory
} from '@/store/slices/configSlice'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import {
  Server,
  Database,
  Brain,
  Zap,
  Shield,
  History,
  Save,
  RotateCcw,
  CheckCircle,
  AlertCircle,
  WifiOff,
  Settings,
  Download,
  Upload
} from 'lucide-react'

interface ConfigInterfaceProps {
  className?: string
}

export function ConfigInterface({ className }: ConfigInterfaceProps) {
  const dispatch = useAppDispatch()
  const config = useAppSelector(selectConfig)
  const isLoading = useAppSelector(selectConfigLoading)
  const hasUnsavedChanges = useAppSelector(selectHasUnsavedChanges)
  const connectionStatus = useAppSelector(selectConnectionStatus)
  const configHistory = useAppSelector(selectConfigHistory)
  const validationErrors = useAppSelector(selectValidationErrors)

  const [activeTab, setActiveTab] = useState('server')

  const handleSave = () => {
    // TODO: Implement save logic with validation
    dispatch(markSaved())
  }

  const handleReset = () => {
    dispatch(resetConfig())
  }

  const getStatusIcon = () => {
    switch (connectionStatus) {
      case 'connected':
        return <CheckCircle className="h-4 w-4 text-green-500" />
      case 'connecting':
        return <Settings className="h-4 w-4 text-yellow-500 animate-spin" />
      case 'error':
        return <AlertCircle className="h-4 w-4 text-red-500" />
      default:
        return <WifiOff className="h-4 w-4 text-gray-500" />
    }
  }

  const getStatusText = () => {
    switch (connectionStatus) {
      case 'connected':
        return 'Connected'
      case 'connecting':
        return 'Connecting...'
      case 'error':
        return 'Connection Error'
      default:
        return 'Disconnected'
    }
  }

  return (
    <div className={cn("space-y-6", className)}>
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-semibold text-foreground">
            MCP Configuration
          </h2>
          <p className="text-sm text-muted-foreground">
            Configure your MCP Memory server settings
          </p>
        </div>
        
        <div className="flex items-center space-x-3">
          <div className="flex items-center space-x-2">
            {getStatusIcon()}
            <span className="text-sm font-medium">{getStatusText()}</span>
          </div>
          
          {hasUnsavedChanges && (
            <Badge variant="outline" className="text-orange-600 border-orange-600">
              Unsaved Changes
            </Badge>
          )}
          
          <div className="flex items-center space-x-2">
            <AlertDialog>
              <AlertDialogTrigger asChild>
                <Button variant="outline" size="sm">
                  <RotateCcw className="mr-2 h-4 w-4" />
                  Reset
                </Button>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>Reset Configuration</AlertDialogTitle>
                  <AlertDialogDescription>
                    Are you sure you want to reset all settings to default values? 
                    This action cannot be undone.
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>Cancel</AlertDialogCancel>
                  <AlertDialogAction onClick={handleReset}>
                    Reset
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
            
            <Button 
              onClick={handleSave} 
              disabled={!hasUnsavedChanges || isLoading}
              size="sm"
            >
              <Save className="mr-2 h-4 w-4" />
              Save Changes
            </Button>
          </div>
        </div>
      </div>

      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList className="grid w-full grid-cols-6">
          <TabsTrigger value="server" className="flex items-center space-x-2">
            <Server className="h-4 w-4" />
            <span>Server</span>
          </TabsTrigger>
          <TabsTrigger value="database" className="flex items-center space-x-2">
            <Database className="h-4 w-4" />
            <span>Database</span>
          </TabsTrigger>
          <TabsTrigger value="ai" className="flex items-center space-x-2">
            <Brain className="h-4 w-4" />
            <span>AI</span>
          </TabsTrigger>
          <TabsTrigger value="performance" className="flex items-center space-x-2">
            <Zap className="h-4 w-4" />
            <span>Performance</span>
          </TabsTrigger>
          <TabsTrigger value="security" className="flex items-center space-x-2">
            <Shield className="h-4 w-4" />
            <span>Security</span>
          </TabsTrigger>
          <TabsTrigger value="history" className="flex items-center space-x-2">
            <History className="h-4 w-4" />
            <span>History</span>
          </TabsTrigger>
        </TabsList>

        {/* Server Configuration */}
        <TabsContent value="server" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Server Settings</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-3 gap-4">
                <div>
                  <Label htmlFor="host">Host</Label>
                  <Input
                    id="host"
                    value={config.host}
                    onChange={(e) => dispatch(updateServerConfig({ host: e.target.value }))}
                    className={validationErrors.host ? "border-red-500" : ""}
                  />
                  {validationErrors.host && (
                    <p className="text-xs text-red-500 mt-1">{validationErrors.host}</p>
                  )}
                </div>
                
                <div>
                  <Label htmlFor="port">Port</Label>
                  <Input
                    id="port"
                    type="number"
                    value={config.port}
                    onChange={(e) => dispatch(updateServerConfig({ port: parseInt(e.target.value) }))}
                    className={validationErrors.port ? "border-red-500" : ""}
                  />
                </div>
                
                <div>
                  <Label htmlFor="protocol">Protocol</Label>
                  <Select 
                    value={config.protocol} 
                    onValueChange={(value: 'http' | 'https') => 
                      dispatch(updateServerConfig({ protocol: value }))
                    }
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="http">HTTP</SelectItem>
                      <SelectItem value="https">HTTPS</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
              
              <Separator />
              
              <div>
                <h4 className="font-medium mb-3">Transport Protocols</h4>
                <div className="grid grid-cols-2 gap-4">
                  {Object.entries(config.transportProtocols).map(([protocol, enabled]) => (
                    <div key={protocol} className="flex items-center justify-between">
                      <Label htmlFor={protocol} className="capitalize">
                        {protocol}
                      </Label>
                      <Switch
                        id={protocol}
                        checked={enabled}
                        onCheckedChange={(checked) =>
                          dispatch(updateTransportProtocols({ [protocol]: checked }))
                        }
                      />
                    </div>
                  ))}
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Database Configuration */}
        <TabsContent value="database" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Vector Database</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <Label htmlFor="vectorProvider">Provider</Label>
                <Select 
                  value={config.vectorDb.provider}
                  onValueChange={(value: 'qdrant' | 'chroma') =>
                    dispatch(updateVectorDbConfig({ provider: value }))
                  }
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="qdrant">Qdrant</SelectItem>
                    <SelectItem value="chroma">ChromaDB</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="vectorHost">Host</Label>
                  <Input
                    id="vectorHost"
                    value={config.vectorDb.host}
                    onChange={(e) => dispatch(updateVectorDbConfig({ host: e.target.value }))}
                  />
                </div>
                
                <div>
                  <Label htmlFor="vectorPort">Port</Label>
                  <Input
                    id="vectorPort"
                    type="number"
                    value={config.vectorDb.port}
                    onChange={(e) => dispatch(updateVectorDbConfig({ port: parseInt(e.target.value) }))}
                  />
                </div>
              </div>
              
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="collection">Collection</Label>
                  <Input
                    id="collection"
                    value={config.vectorDb.collection}
                    onChange={(e) => dispatch(updateVectorDbConfig({ collection: e.target.value }))}
                  />
                </div>
                
                <div>
                  <Label htmlFor="dimension">Dimension</Label>
                  <Input
                    id="dimension"
                    type="number"
                    value={config.vectorDb.dimension}
                    onChange={(e) => dispatch(updateVectorDbConfig({ dimension: parseInt(e.target.value) }))}
                  />
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* AI Configuration */}
        <TabsContent value="ai" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>OpenAI Integration</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <Label htmlFor="apiKey">API Key</Label>
                <Input
                  id="apiKey"
                  type="password"
                  value={config.openai.apiKey || ''}
                  onChange={(e) => dispatch(updateOpenAIConfig({ apiKey: e.target.value }))}
                  placeholder="sk-..."
                />
              </div>
              
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="model">Model</Label>
                  <Select
                    value={config.openai.model}
                    onValueChange={(value) => dispatch(updateOpenAIConfig({ model: value }))}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="text-embedding-ada-002">text-embedding-ada-002</SelectItem>
                      <SelectItem value="text-embedding-3-small">text-embedding-3-small</SelectItem>
                      <SelectItem value="text-embedding-3-large">text-embedding-3-large</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                
                <div>
                  <Label htmlFor="maxTokens">Max Tokens</Label>
                  <Input
                    id="maxTokens"
                    type="number"
                    value={config.openai.maxTokens}
                    onChange={(e) => dispatch(updateOpenAIConfig({ maxTokens: parseInt(e.target.value) }))}
                  />
                </div>
              </div>
              
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="temperature">Temperature</Label>
                  <Input
                    id="temperature"
                    type="number"
                    step="0.1"
                    min="0"
                    max="2"
                    value={config.openai.temperature}
                    onChange={(e) => dispatch(updateOpenAIConfig({ temperature: parseFloat(e.target.value) }))}
                  />
                </div>
                
                <div>
                  <Label htmlFor="timeout">Timeout (ms)</Label>
                  <Input
                    id="timeout"
                    type="number"
                    value={config.openai.timeout}
                    onChange={(e) => dispatch(updateOpenAIConfig({ timeout: parseInt(e.target.value) }))}
                  />
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Performance Configuration */}
        <TabsContent value="performance" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Performance Settings</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <div>
                    <Label>Cache Enabled</Label>
                    <p className="text-xs text-muted-foreground">Enable query result caching</p>
                  </div>
                  <Switch
                    checked={config.cacheEnabled}
                    onCheckedChange={(checked) => dispatch(toggleFeature({ feature: 'cacheEnabled' }))}
                  />
                </div>
                
                <div className="flex items-center justify-between">
                  <div>
                    <Label>Realtime Updates</Label>
                    <p className="text-xs text-muted-foreground">Enable real-time memory updates</p>
                  </div>
                  <Switch
                    checked={config.realtimeEnabled}
                    onCheckedChange={(checked) => dispatch(toggleFeature({ feature: 'realtimeEnabled' }))}
                  />
                </div>
                
                <div className="flex items-center justify-between">
                  <div>
                    <Label>Analytics</Label>
                    <p className="text-xs text-muted-foreground">Enable usage analytics</p>
                  </div>
                  <Switch
                    checked={config.analyticsEnabled}
                    onCheckedChange={(checked) => dispatch(toggleFeature({ feature: 'analyticsEnabled' }))}
                  />
                </div>
                
                <div className="flex items-center justify-between">
                  <div>
                    <Label>Debug Mode</Label>
                    <p className="text-xs text-muted-foreground">Enable detailed logging</p>
                  </div>
                  <Switch
                    checked={config.debugMode}
                    onCheckedChange={(checked) => dispatch(toggleFeature({ feature: 'debugMode' }))}
                  />
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Security Configuration */}
        <TabsContent value="security" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Security Settings</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <Label>Authentication Enabled</Label>
                  <p className="text-xs text-muted-foreground">Require API key authentication</p>
                </div>
                <Switch
                  checked={config.authEnabled}
                  onCheckedChange={(checked) => 
                    dispatch(updateServerConfig({ authEnabled: checked } as any))
                  }
                />
              </div>
              
              {config.authEnabled && (
                <div>
                  <Label htmlFor="authApiKey">API Key</Label>
                  <Input
                    id="authApiKey"
                    type="password"
                    value={config.apiKey || ''}
                    onChange={(e) => dispatch(updateServerConfig({ apiKey: e.target.value } as any))}
                    placeholder="Enter API key"
                  />
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Configuration History */}
        <TabsContent value="history" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center justify-between">
                Configuration History
                <div className="flex items-center space-x-2">
                  <Button variant="outline" size="sm">
                    <Download className="mr-2 h-4 w-4" />
                    Export
                  </Button>
                  <Button variant="outline" size="sm">
                    <Upload className="mr-2 h-4 w-4" />
                    Import
                  </Button>
                </div>
              </CardTitle>
            </CardHeader>
            <CardContent>
              <ScrollArea className="h-64">
                <div className="space-y-3">
                  {configHistory.length === 0 ? (
                    <p className="text-muted-foreground text-center py-8">
                      No configuration history available
                    </p>
                  ) : (
                    configHistory.map((entry, index) => (
                      <div key={index} className="flex items-center justify-between p-3 border rounded-lg">
                        <div>
                          <p className="font-medium">{entry.description}</p>
                          <p className="text-xs text-muted-foreground">
                            {new Date(entry.timestamp).toLocaleString()}
                          </p>
                        </div>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => dispatch(restoreFromHistory(index))}
                        >
                          Restore
                        </Button>
                      </div>
                    ))
                  )}
                </div>
              </ScrollArea>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}