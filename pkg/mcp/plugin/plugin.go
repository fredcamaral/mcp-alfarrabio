package plugin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Common errors
var (
	ErrPluginNotFound     = errors.New("plugin not found")
	ErrPluginAlreadyExists = errors.New("plugin already exists")
	ErrPluginDisabled     = errors.New("plugin is disabled")
	ErrInvalidPlugin      = errors.New("invalid plugin")
)

// PluginType represents the type of plugin
type PluginType string

const (
	PluginTypeMiddleware PluginType = "middleware"
	PluginTypeTransport  PluginType = "transport"
	PluginTypeTool       PluginType = "tool"
	PluginTypeHandler    PluginType = "handler"
)

// PluginState represents the state of a plugin
type PluginState string

const (
	PluginStateUnloaded PluginState = "unloaded"
	PluginStateLoading  PluginState = "loading"
	PluginStateLoaded   PluginState = "loaded"
	PluginStateEnabled  PluginState = "enabled"
	PluginStateDisabled PluginState = "disabled"
	PluginStateError    PluginState = "error"
)

// Plugin represents a plugin in the system
type Plugin interface {
	// Metadata
	Name() string
	Version() string
	Description() string
	Type() PluginType
	
	// Lifecycle
	Init(ctx context.Context, config map[string]interface{}) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	
	// Health
	Health() error
}

// Middleware plugin interface
type MiddlewarePlugin interface {
	Plugin
	Middleware() interface{} // Returns the actual middleware implementation
}

// Transport plugin interface
type TransportPlugin interface {
	Plugin
	Transport() interface{} // Returns the actual transport implementation
}

// Tool plugin interface
type ToolPlugin interface {
	Plugin
	Tools() []interface{} // Returns the tool implementations
}

// Handler plugin interface
type HandlerPlugin interface {
	Plugin
	Handlers() map[string]interface{} // Returns method -> handler mapping
}

// PluginInfo contains information about a plugin
type PluginInfo struct {
	Name        string
	Version     string
	Description string
	Type        PluginType
	State       PluginState
	Error       error
	LoadedAt    time.Time
	Config      map[string]interface{}
}

// PluginRegistry manages plugins
type PluginRegistry struct {
	plugins map[string]*pluginEntry
	mu      sync.RWMutex
	logger  *slog.Logger
	
	// Hooks
	onLoad   []func(plugin Plugin)
	onUnload []func(plugin Plugin)
	onError  []func(plugin Plugin, err error)
}

// pluginEntry represents a registered plugin with metadata
type pluginEntry struct {
	plugin   Plugin
	info     *PluginInfo
	state    PluginState
	error    error
	loadedAt time.Time
}

// NewPluginRegistry creates a new plugin registry
func NewPluginRegistry(logger *slog.Logger) *PluginRegistry {
	if logger == nil {
		logger = slog.Default()
	}
	
	return &PluginRegistry{
		plugins:  make(map[string]*pluginEntry),
		logger:   logger,
		onLoad:   make([]func(Plugin), 0),
		onUnload: make([]func(Plugin), 0),
		onError:  make([]func(Plugin, error), 0),
	}
}

// Register registers a plugin
func (r *PluginRegistry) Register(plugin Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	name := plugin.Name()
	if _, exists := r.plugins[name]; exists {
		return ErrPluginAlreadyExists
	}
	
	entry := &pluginEntry{
		plugin: plugin,
		info: &PluginInfo{
			Name:        plugin.Name(),
			Version:     plugin.Version(),
			Description: plugin.Description(),
			Type:        plugin.Type(),
			State:       PluginStateUnloaded,
		},
		state: PluginStateUnloaded,
	}
	
	r.plugins[name] = entry
	
	r.logger.Info("plugin registered",
		"name", name,
		"version", plugin.Version(),
		"type", plugin.Type())
	
	return nil
}

// Unregister removes a plugin from the registry
func (r *PluginRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	entry, exists := r.plugins[name]
	if !exists {
		return ErrPluginNotFound
	}
	
	// Stop the plugin if it's running
	if entry.state == PluginStateEnabled {
		if err := entry.plugin.Stop(context.Background()); err != nil {
			r.logger.Error("failed to stop plugin",
				"name", name,
				"error", err)
		}
	}
	
	// Call unload hooks
	for _, hook := range r.onUnload {
		hook(entry.plugin)
	}
	
	delete(r.plugins, name)
	
	r.logger.Info("plugin unregistered", "name", name)
	
	return nil
}

// Load initializes and starts a plugin
func (r *PluginRegistry) Load(ctx context.Context, name string, config map[string]interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	entry, exists := r.plugins[name]
	if !exists {
		return ErrPluginNotFound
	}
	
	// Update state
	entry.state = PluginStateLoading
	entry.info.State = PluginStateLoading
	
	// Initialize the plugin
	if err := entry.plugin.Init(ctx, config); err != nil {
		entry.state = PluginStateError
		entry.info.State = PluginStateError
		entry.error = err
		entry.info.Error = err
		
		// Call error hooks
		for _, hook := range r.onError {
			hook(entry.plugin, err)
		}
		
		return fmt.Errorf("failed to initialize plugin %s: %w", name, err)
	}
	
	// Start the plugin
	if err := entry.plugin.Start(ctx); err != nil {
		entry.state = PluginStateError
		entry.info.State = PluginStateError
		entry.error = err
		entry.info.Error = err
		
		// Call error hooks
		for _, hook := range r.onError {
			hook(entry.plugin, err)
		}
		
		return fmt.Errorf("failed to start plugin %s: %w", name, err)
	}
	
	// Update state
	entry.state = PluginStateEnabled
	entry.info.State = PluginStateEnabled
	entry.loadedAt = time.Now()
	entry.info.LoadedAt = entry.loadedAt
	entry.info.Config = config
	
	// Call load hooks
	for _, hook := range r.onLoad {
		hook(entry.plugin)
	}
	
	r.logger.Info("plugin loaded",
		"name", name,
		"version", entry.plugin.Version())
	
	return nil
}

// Unload stops and unloads a plugin
func (r *PluginRegistry) Unload(ctx context.Context, name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	entry, exists := r.plugins[name]
	if !exists {
		return ErrPluginNotFound
	}
	
	if entry.state != PluginStateEnabled {
		return fmt.Errorf("plugin %s is not loaded", name)
	}
	
	// Stop the plugin
	if err := entry.plugin.Stop(ctx); err != nil {
		r.logger.Error("failed to stop plugin",
			"name", name,
			"error", err)
		return fmt.Errorf("failed to stop plugin %s: %w", name, err)
	}
	
	// Update state
	entry.state = PluginStateUnloaded
	entry.info.State = PluginStateUnloaded
	
	// Call unload hooks
	for _, hook := range r.onUnload {
		hook(entry.plugin)
	}
	
	r.logger.Info("plugin unloaded", "name", name)
	
	return nil
}

// Get returns a plugin by name
func (r *PluginRegistry) Get(name string) (Plugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	entry, exists := r.plugins[name]
	if !exists {
		return nil, ErrPluginNotFound
	}
	
	if entry.state != PluginStateEnabled {
		return nil, ErrPluginDisabled
	}
	
	return entry.plugin, nil
}

// GetByType returns all plugins of a specific type
func (r *PluginRegistry) GetByType(pluginType PluginType) []Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	plugins := make([]Plugin, 0)
	for _, entry := range r.plugins {
		if entry.plugin.Type() == pluginType && entry.state == PluginStateEnabled {
			plugins = append(plugins, entry.plugin)
		}
	}
	
	return plugins
}

// List returns information about all registered plugins
func (r *PluginRegistry) List() []*PluginInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	infos := make([]*PluginInfo, 0, len(r.plugins))
	for _, entry := range r.plugins {
		info := *entry.info // Copy the info
		infos = append(infos, &info)
	}
	
	return infos
}

// Health checks the health of all loaded plugins
func (r *PluginRegistry) Health() map[string]error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	health := make(map[string]error)
	for name, entry := range r.plugins {
		if entry.state == PluginStateEnabled {
			health[name] = entry.plugin.Health()
		}
	}
	
	return health
}

// OnLoad registers a hook to be called when a plugin is loaded
func (r *PluginRegistry) OnLoad(hook func(Plugin)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.onLoad = append(r.onLoad, hook)
}

// OnUnload registers a hook to be called when a plugin is unloaded
func (r *PluginRegistry) OnUnload(hook func(Plugin)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.onUnload = append(r.onUnload, hook)
}

// OnError registers a hook to be called when a plugin encounters an error
func (r *PluginRegistry) OnError(hook func(Plugin, error)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.onError = append(r.onError, hook)
}

// PluginManager provides high-level plugin management
type PluginManager struct {
	registry *PluginRegistry
	loader   PluginLoader
	logger   *slog.Logger
}

// PluginLoader loads plugins from various sources
type PluginLoader interface {
	Load(path string) (Plugin, error)
}

// NewPluginManager creates a new plugin manager
func NewPluginManager(loader PluginLoader, logger *slog.Logger) *PluginManager {
	if logger == nil {
		logger = slog.Default()
	}
	
	return &PluginManager{
		registry: NewPluginRegistry(logger),
		loader:   loader,
		logger:   logger,
	}
}

// LoadFromPath loads a plugin from a file path
func (m *PluginManager) LoadFromPath(ctx context.Context, path string, config map[string]interface{}) error {
	// Load the plugin
	plugin, err := m.loader.Load(path)
	if err != nil {
		return fmt.Errorf("failed to load plugin from %s: %w", path, err)
	}
	
	// Register the plugin
	if err := m.registry.Register(plugin); err != nil {
		return fmt.Errorf("failed to register plugin: %w", err)
	}
	
	// Load the plugin
	if err := m.registry.Load(ctx, plugin.Name(), config); err != nil {
		// Unregister on failure
		m.registry.Unregister(plugin.Name())
		return err
	}
	
	return nil
}

// Registry returns the plugin registry
func (m *PluginManager) Registry() *PluginRegistry {
	return m.registry
}

// Example base plugin implementation

// BasePlugin provides common plugin functionality
type BasePlugin struct {
	name        string
	version     string
	description string
	pluginType  PluginType
	logger      *slog.Logger
	config      map[string]interface{}
	mu          sync.RWMutex
}

// NewBasePlugin creates a new base plugin
func NewBasePlugin(name, version, description string, pluginType PluginType, logger *slog.Logger) *BasePlugin {
	if logger == nil {
		logger = slog.Default()
	}
	
	return &BasePlugin{
		name:        name,
		version:     version,
		description: description,
		pluginType:  pluginType,
		logger:      logger,
	}
}

// Name returns the plugin name
func (p *BasePlugin) Name() string {
	return p.name
}

// Version returns the plugin version
func (p *BasePlugin) Version() string {
	return p.version
}

// Description returns the plugin description
func (p *BasePlugin) Description() string {
	return p.description
}

// Type returns the plugin type
func (p *BasePlugin) Type() PluginType {
	return p.pluginType
}

// Init initializes the plugin with configuration
func (p *BasePlugin) Init(ctx context.Context, config map[string]interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.config = config
	p.logger.Info("plugin initialized",
		"name", p.name,
		"config", config)
	
	return nil
}

// Start starts the plugin
func (p *BasePlugin) Start(ctx context.Context) error {
	p.logger.Info("plugin started", "name", p.name)
	return nil
}

// Stop stops the plugin
func (p *BasePlugin) Stop(ctx context.Context) error {
	p.logger.Info("plugin stopped", "name", p.name)
	return nil
}

// Health checks the plugin health
func (p *BasePlugin) Health() error {
	return nil
}

// Config returns the plugin configuration
func (p *BasePlugin) Config() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	// Return a copy to prevent modification
	config := make(map[string]interface{})
	for k, v := range p.config {
		config[k] = v
	}
	
	return config
}