'use client'

import { useState, useMemo } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useToast } from '@/components/ui/use-toast'
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
    DialogTrigger,
} from '@/components/ui/dialog'
import {
    GitBranch,
    Plus,
    Search,
    MoreHorizontal,
    Activity,
    Clock,
    Database,
    Trash2,
    Settings,
    RefreshCw,
    ExternalLink,
    AlertCircle
} from 'lucide-react'
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
    useListRepositories,
    useGetRepositoryStats,
    useAddRepository,
    useSyncRepository,
    useRemoveRepository,
    useGraphQLError
} from '@/lib/graphql/hooks'
import { logger } from '@/lib/logger'
import { formatDistanceToNow } from 'date-fns'

type RepositoryStatus = 'ACTIVE' | 'INACTIVE' | 'SYNCING' | 'ERROR'

export function RepositoryManager() {
    const [searchQuery, setSearchQuery] = useState('')
    const [selectedTab, setSelectedTab] = useState<'all' | RepositoryStatus>('all')
    const [isAddDialogOpen, setIsAddDialogOpen] = useState(false)
    const [newRepoUrl, setNewRepoUrl] = useState('')
    const [newRepoDescription, setNewRepoDescription] = useState('')
    
    const { toast } = useToast()
    const { handleError } = useGraphQLError()
    
    // Get repository status filter
    const statusFilter = selectedTab === 'all' ? undefined : selectedTab
    
    // Fetch repositories
    const { data: repoData, loading: reposLoading, error: reposError, refetch: refetchRepos } = useListRepositories(statusFilter)
    const { data: statsData, loading: statsLoading, error: statsError, refetch: refetchStats } = useGetRepositoryStats()
    
    // Mutations
    const [addRepository, { loading: addingRepo }] = useAddRepository()
    const [syncRepository, { loading: syncingRepo }] = useSyncRepository()
    const [removeRepository, { loading: removingRepo }] = useRemoveRepository()
    
    const repositories = useMemo(() => repoData?.listRepositories || [], [repoData?.listRepositories])
    
    const stats = statsData?.getRepositoryStats || {
        totalRepositories: 0,
        activeRepositories: 0,
        totalMemories: 0,
        totalPatterns: 0,
        recentActivity: 0
    }
    
    const isLoading = reposLoading || statsLoading
    const isMutating = addingRepo || syncingRepo || removingRepo

    const filteredRepositories = useMemo(() => {
        return repositories.filter(repo => {
            const matchesSearch = searchQuery.trim() === '' || 
                repo.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
                repo.description?.toLowerCase().includes(searchQuery.toLowerCase()) ||
                repo.url.toLowerCase().includes(searchQuery.toLowerCase())
            return matchesSearch
        })
    }, [repositories, searchQuery])

    const getStatusColor = (status: RepositoryStatus) => {
        switch (status) {
            case 'ACTIVE': return 'bg-success-muted text-success border-success-muted'
            case 'INACTIVE': return 'bg-muted text-muted-foreground border-border'
            case 'SYNCING': return 'bg-info-muted text-info border-info-muted'
            case 'ERROR': return 'bg-destructive/10 text-destructive border-destructive/20'
            default: return 'bg-muted text-muted-foreground border-border'
        }
    }
    
    const getStatusLabel = (status: RepositoryStatus) => {
        return status.charAt(0) + status.slice(1).toLowerCase()
    }

    const handleAddRepository = async () => {
        if (!newRepoUrl.trim()) {
            toast({
                title: 'Error',
                description: 'Please enter a repository URL',
                variant: 'destructive'
            })
            return
        }
        
        try {
            logger.info('Adding repository', { 
                component: 'RepositoryManager',
                url: newRepoUrl 
            })
            
            await addRepository({
                variables: {
                    input: {
                        url: newRepoUrl.trim(),
                        description: newRepoDescription.trim() || undefined
                    }
                }
            })
            
            toast({
                title: 'Success',
                description: 'Repository added successfully'
            })
            
            setNewRepoUrl('')
            setNewRepoDescription('')
            setIsAddDialogOpen(false)
            
            // Refetch stats
            refetchStats()
        } catch (error) {
            logger.error('Failed to add repository', error, {
                component: 'RepositoryManager',
                url: newRepoUrl
            })
            
            toast({
                title: 'Error',
                description: handleError(error),
                variant: 'destructive'
            })
        }
    }
    
    const handleSyncRepository = async (id: string) => {
        try {
            logger.info('Syncing repository', { 
                component: 'RepositoryManager',
                repositoryId: id 
            })
            
            await syncRepository({ variables: { id } })
            
            toast({
                title: 'Syncing',
                description: 'Repository sync started'
            })
        } catch (error) {
            logger.error('Failed to sync repository', error, {
                component: 'RepositoryManager',
                repositoryId: id
            })
            
            toast({
                title: 'Error',
                description: handleError(error),
                variant: 'destructive'
            })
        }
    }
    
    const handleRemoveRepository = async (id: string) => {
        try {
            logger.info('Removing repository', { 
                component: 'RepositoryManager',
                repositoryId: id 
            })
            
            await removeRepository({ variables: { id } })
            
            toast({
                title: 'Success',
                description: 'Repository removed successfully'
            })
            
            // Refetch stats
            refetchStats()
        } catch (error) {
            logger.error('Failed to remove repository', error, {
                component: 'RepositoryManager',
                repositoryId: id
            })
            
            toast({
                title: 'Error',
                description: handleError(error),
                variant: 'destructive'
            })
        }
    }
    
    const handleSyncAll = async () => {
        try {
            logger.info('Syncing all repositories', { 
                component: 'RepositoryManager'
            })
            
            // Sync all active repositories
            const activeRepos = repositories.filter(r => r.status === 'ACTIVE')
            await Promise.all(activeRepos.map(repo => 
                syncRepository({ variables: { id: repo.id } })
            ))
            
            toast({
                title: 'Syncing',
                description: `Started sync for ${activeRepos.length} repositories`
            })
        } catch (error) {
            logger.error('Failed to sync all repositories', error, {
                component: 'RepositoryManager'
            })
            
            toast({
                title: 'Error',
                description: handleError(error),
                variant: 'destructive'
            })
        }
    }

    if (isLoading) {
        return (
            <div className="space-y-6">
                <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
                    {Array.from({ length: 4 }).map((_, i) => (
                        <Card key={i} className="animate-pulse">
                            <CardHeader className="pb-2">
                                <div className="h-4 bg-muted rounded w-3/4" />
                            </CardHeader>
                            <CardContent>
                                <div className="h-8 bg-muted rounded w-1/2" />
                            </CardContent>
                        </Card>
                    ))}
                </div>
            </div>
        )
    }
    
    if (reposError || statsError) {
        const error = reposError || statsError
        logger.error('Failed to load repositories', error, {
            component: 'RepositoryManager'
        })
        
        return (
            <Card className="border-destructive">
                <CardHeader>
                    <CardTitle className="text-destructive flex items-center gap-2">
                        <AlertCircle className="h-5 w-5" />
                        Error Loading Repositories
                    </CardTitle>
                </CardHeader>
                <CardContent>
                    <p className="text-muted-foreground mb-4">
                        Unable to load repository data. Please try again.
                    </p>
                    <Button 
                        onClick={() => {
                            refetchRepos()
                            refetchStats()
                        }} 
                        variant="outline"
                    >
                        <RefreshCw className="h-4 w-4 mr-2" />
                        Retry
                    </Button>
                </CardContent>
            </Card>
        )
    }

    return (
        <div className="space-y-6">
            {/* Stats Overview */}
            <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
                <Card className="border-2 hover:shadow-lg transition-all">
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                        <CardTitle className="text-sm font-medium">Total Repositories</CardTitle>
                        <GitBranch className="h-4 w-4 text-muted-foreground" />
                    </CardHeader>
                    <CardContent>
                        <div className="text-2xl font-bold">{stats.totalRepositories}</div>
                        <p className="text-xs text-muted-foreground">
                            Connected repositories
                        </p>
                    </CardContent>
                </Card>

                <Card className="border-2 hover:shadow-lg transition-all">
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                        <CardTitle className="text-sm font-medium">Active</CardTitle>
                        <Activity className="h-4 w-4 text-muted-foreground" />
                    </CardHeader>
                    <CardContent>
                        <div className="text-2xl font-bold text-success">{stats.activeRepositories}</div>
                        <p className="text-xs text-muted-foreground">
                            Currently active
                        </p>
                    </CardContent>
                </Card>

                <Card className="border-2 hover:shadow-lg transition-all">
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                        <CardTitle className="text-sm font-medium">Total Memories</CardTitle>
                        <Database className="h-4 w-4 text-muted-foreground" />
                    </CardHeader>
                    <CardContent>
                        <div className="text-2xl font-bold text-info">{stats.totalMemories}</div>
                        <p className="text-xs text-muted-foreground">
                            Across all repositories
                        </p>
                    </CardContent>
                </Card>

                <Card className="border-2 hover:shadow-lg transition-all">
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                        <CardTitle className="text-sm font-medium">Recent Activity</CardTitle>
                        <Clock className="h-4 w-4 text-muted-foreground" />
                    </CardHeader>
                    <CardContent>
                        <div className="text-2xl font-bold text-purple">{stats.recentActivity}</div>
                        <p className="text-xs text-muted-foreground">
                            New memories this week
                        </p>
                    </CardContent>
                </Card>
            </div>

            {/* Repository Management */}
            <Card>
                <CardHeader>
                    <div className="flex items-center justify-between">
                        <CardTitle className="flex items-center gap-2">
                            <GitBranch className="h-5 w-5" />
                            Repository Management
                        </CardTitle>
                        <div className="flex items-center gap-2">
                            <Button 
                                variant="outline" 
                                size="sm"
                                onClick={handleSyncAll}
                                disabled={isMutating}
                            >
                                <RefreshCw className={`h-4 w-4 mr-2 ${syncingRepo ? 'animate-spin' : ''}`} />
                                Sync All
                            </Button>
                            <Dialog open={isAddDialogOpen} onOpenChange={setIsAddDialogOpen}>
                                <DialogTrigger asChild>
                                    <Button size="sm">
                                        <Plus className="h-4 w-4 mr-2" />
                                        Add Repository
                                    </Button>
                                </DialogTrigger>
                                <DialogContent>
                                    <DialogHeader>
                                        <DialogTitle>Add New Repository</DialogTitle>
                                        <DialogDescription>
                                            Connect a new repository to start tracking memories and patterns.
                                        </DialogDescription>
                                    </DialogHeader>
                                    <div className="space-y-4">
                                        <div className="space-y-2">
                                            <Label htmlFor="repo-url">Repository URL</Label>
                                            <Input
                                                id="repo-url"
                                                value={newRepoUrl}
                                                onChange={(e) => setNewRepoUrl(e.target.value)}
                                                placeholder="github.com/user/repository"
                                                disabled={addingRepo}
                                            />
                                        </div>
                                        <div className="space-y-2">
                                            <Label htmlFor="repo-description">Description (optional)</Label>
                                            <Input
                                                id="repo-description"
                                                value={newRepoDescription}
                                                onChange={(e) => setNewRepoDescription(e.target.value)}
                                                placeholder="Brief description of the repository"
                                                disabled={addingRepo}
                                            />
                                        </div>
                                    </div>
                                    <DialogFooter>
                                        <Button variant="outline" onClick={() => setIsAddDialogOpen(false)}>
                                            Cancel
                                        </Button>
                                        <Button 
                                            onClick={handleAddRepository}
                                            disabled={addingRepo}
                                        >
                                            {addingRepo ? 'Adding...' : 'Add Repository'}
                                        </Button>
                                    </DialogFooter>
                                </DialogContent>
                            </Dialog>
                        </div>
                    </div>
                </CardHeader>
                <CardContent>
                    {/* Search and Filters */}
                    <div className="flex items-center gap-4 mb-6">
                        <div className="relative flex-1 max-w-md">
                            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                            <Input
                                placeholder="Search repositories..."
                                value={searchQuery}
                                onChange={(e) => setSearchQuery(e.target.value)}
                                className="pl-10"
                            />
                        </div>

                        <Tabs value={selectedTab} onValueChange={(v) => setSelectedTab(v as typeof selectedTab)}>
                            <TabsList>
                                <TabsTrigger value="all">All</TabsTrigger>
                                <TabsTrigger value="ACTIVE">Active</TabsTrigger>
                                <TabsTrigger value="INACTIVE">Inactive</TabsTrigger>
                                <TabsTrigger value="SYNCING">Syncing</TabsTrigger>
                                <TabsTrigger value="ERROR">Error</TabsTrigger>
                            </TabsList>
                        </Tabs>
                    </div>

                    {/* Repository List */}
                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                        {filteredRepositories.map((repo) => (
                            <Card key={repo.id} className="hover:shadow-md transition-all">
                                <CardHeader className="pb-3">
                                    <div className="flex items-start justify-between">
                                        <div className="flex-1">
                                            <CardTitle className="text-lg mb-1 flex items-center gap-2">
                                                <GitBranch className="h-4 w-4" />
                                                {repo.name}
                                            </CardTitle>
                                            <p className="text-sm text-muted-foreground line-clamp-2">
                                                {repo.description}
                                            </p>
                                        </div>
                                        <DropdownMenu>
                                            <DropdownMenuTrigger asChild>
                                                <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
                                                    <MoreHorizontal className="h-4 w-4" />
                                                </Button>
                                            </DropdownMenuTrigger>
                                            <DropdownMenuContent align="end">
                                                <DropdownMenuItem>
                                                    <ExternalLink className="mr-2 h-4 w-4" />
                                                    View Repository
                                                </DropdownMenuItem>
                                                <DropdownMenuItem>
                                                    <Settings className="mr-2 h-4 w-4" />
                                                    Configure
                                                </DropdownMenuItem>
                                                <DropdownMenuItem
                                                    onClick={() => handleSyncRepository(repo.id)}
                                                    disabled={repo.status === 'SYNCING'}
                                                >
                                                    <RefreshCw className="mr-2 h-4 w-4" />
                                                    Sync Now
                                                </DropdownMenuItem>
                                                <DropdownMenuItem 
                                                    className="text-destructive"
                                                    onClick={() => handleRemoveRepository(repo.id)}
                                                >
                                                    <Trash2 className="mr-2 h-4 w-4" />
                                                    Remove
                                                </DropdownMenuItem>
                                            </DropdownMenuContent>
                                        </DropdownMenu>
                                    </div>
                                </CardHeader>
                                <CardContent className="space-y-4">
                                    <div className="flex items-center justify-between">
                                        <Badge className={getStatusColor(repo.status)}>
                                            {getStatusLabel(repo.status)}
                                        </Badge>
                                        <div className="text-sm text-muted-foreground">
                                            {repo.lastActivity ? formatDistanceToNow(new Date(repo.lastActivity), { addSuffix: true }) : 'Never'}
                                        </div>
                                    </div>

                                    <div className="grid grid-cols-3 gap-4 text-sm">
                                        <div className="text-center">
                                            <div className="font-semibold text-lg">{repo.memoryCount}</div>
                                            <div className="text-muted-foreground">Memories</div>
                                        </div>
                                        <div className="text-center">
                                            <div className="font-semibold text-lg">{repo.patternCount}</div>
                                            <div className="text-muted-foreground">Patterns</div>
                                        </div>
                                        <div className="text-center">
                                            <div className="font-semibold text-lg">{repo.metadata?.contributors || 0}</div>
                                            <div className="text-muted-foreground">Contributors</div>
                                        </div>
                                    </div>

                                    {repo.metadata?.technologies && repo.metadata.technologies.length > 0 && (
                                        <div className="space-y-2">
                                            <div className="text-sm font-medium">Technologies:</div>
                                            <div className="flex flex-wrap gap-1">
                                                {repo.metadata.technologies.map((tech) => (
                                                    <Badge key={tech} variant="outline" className="text-xs">
                                                        {tech}
                                                    </Badge>
                                                ))}
                                            </div>
                                        </div>
                                    )}

                                    <div className="text-xs text-muted-foreground">
                                        {repo.url}
                                    </div>
                                </CardContent>
                            </Card>
                        ))}
                    </div>

                    {filteredRepositories.length === 0 && (
                        <div className="text-center py-12">
                            <GitBranch className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
                            <h3 className="text-lg font-medium mb-2">No repositories found</h3>
                            <p className="text-muted-foreground mb-4">
                                {searchQuery ? 'No repositories match your search.' : 'Get started by adding your first repository.'}
                            </p>
                            {!searchQuery && (
                                <Button onClick={() => setIsAddDialogOpen(true)}>
                                    <Plus className="h-4 w-4 mr-2" />
                                    Add Repository
                                </Button>
                            )}
                        </div>
                    )}
                </CardContent>
            </Card>
        </div>
    )
} 