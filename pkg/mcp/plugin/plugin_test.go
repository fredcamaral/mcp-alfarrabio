package plugin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock plugin implementation for testing
type mockPlugin struct {
	*BasePlugin
	initCalled  bool
	startCalled bool
	stopCalled  bool
	initError   error
	startError  error
	stopError   error
	healthError error
}

func newMockPlugin(name string, pluginType PluginType) *mockPlugin {
	return &mockPlugin{
		BasePlugin: NewBasePlugin(name, "1.0.0", "Mock plugin", pluginType, slog.Default()),
	}
}

func (p *mockPlugin) Init(ctx context.Context, config map[string]interface{}) error {
	p.initCalled = true
	if p.initError != nil {
		return p.initError
	}
	return p.BasePlugin.Init(ctx, config)
}

func (p *mockPlugin) Start(ctx context.Context) error {
	p.startCalled = true
	if p.startError != nil {
		return p.startError
	}
	return p.BasePlugin.Start(ctx)
}

func (p *mockPlugin) Stop(ctx context.Context) error {
	p.stopCalled = true
	if p.stopError != nil {
		return p.stopError
	}
	return p.BasePlugin.Stop(ctx)
}

func (p *mockPlugin) Health() error {
	return p.healthError
}

// Mock middleware plugin
type mockMiddlewarePlugin struct {
	*mockPlugin
	middleware interface{}
}

func newMockMiddlewarePlugin(name string) *mockMiddlewarePlugin {
	return &mockMiddlewarePlugin{
		mockPlugin: newMockPlugin(name, PluginTypeMiddleware),
		middleware: "mock-middleware",
	}
}

func (p *mockMiddlewarePlugin) Middleware() interface{} {
	return p.middleware
}

// Mock tool plugin
type mockToolPlugin struct {
	*mockPlugin
	tools []interface{}
}

func newMockToolPlugin(name string) *mockToolPlugin {
	return &mockToolPlugin{
		mockPlugin: newMockPlugin(name, PluginTypeTool),
		tools:      []interface{}{"tool1", "tool2"},
	}
}

func (p *mockToolPlugin) Tools() []interface{} {
	return p.tools
}

func TestPluginRegistry(t *testing.T) {
	logger := slog.Default()
	
	t.Run("register and get plugin", func(t *testing.T) {
		registry := NewPluginRegistry(logger)
		plugin := newMockPlugin("test-plugin", PluginTypeHandler)
		
		// Register plugin
		err := registry.Register(plugin)
		assert.NoError(t, err)
		
		// Try to register again
		err = registry.Register(plugin)
		assert.Error(t, err)
		assert.Equal(t, ErrPluginAlreadyExists, err)
		
		// Get plugin before loading
		_, err = registry.Get("test-plugin")
		assert.Error(t, err)
		assert.Equal(t, ErrPluginDisabled, err)
		
		// Load plugin
		err = registry.Load(context.Background(), "test-plugin", nil)
		assert.NoError(t, err)
		assert.True(t, plugin.initCalled)
		assert.True(t, plugin.startCalled)
		
		// Get loaded plugin
		retrieved, err := registry.Get("test-plugin")
		assert.NoError(t, err)
		assert.Equal(t, plugin, retrieved)
	})
	
	t.Run("unregister plugin", func(t *testing.T) {
		registry := NewPluginRegistry(logger)
		plugin := newMockPlugin("test-plugin", PluginTypeHandler)
		
		// Register and load
		registry.Register(plugin)
		registry.Load(context.Background(), "test-plugin", nil)
		
		// Unregister
		err := registry.Unregister("test-plugin")
		assert.NoError(t, err)
		assert.True(t, plugin.stopCalled)
		
		// Try to get unregistered plugin
		_, err = registry.Get("test-plugin")
		assert.Error(t, err)
		assert.Equal(t, ErrPluginNotFound, err)
		
		// Try to unregister non-existent plugin
		err = registry.Unregister("non-existent")
		assert.Error(t, err)
		assert.Equal(t, ErrPluginNotFound, err)
	})
	
	t.Run("load with init error", func(t *testing.T) {
		registry := NewPluginRegistry(logger)
		plugin := newMockPlugin("test-plugin", PluginTypeHandler)
		plugin.initError = errors.New("init failed")
		
		registry.Register(plugin)
		err := registry.Load(context.Background(), "test-plugin", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "init failed")
		
		// Plugin should be in error state
		infos := registry.List()
		require.Len(t, infos, 1)
		assert.Equal(t, PluginStateError, infos[0].State)
		assert.NotNil(t, infos[0].Error)
	})
	
	t.Run("load with start error", func(t *testing.T) {
		registry := NewPluginRegistry(logger)
		plugin := newMockPlugin("test-plugin", PluginTypeHandler)
		plugin.startError = errors.New("start failed")
		
		registry.Register(plugin)
		err := registry.Load(context.Background(), "test-plugin", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "start failed")
		assert.True(t, plugin.initCalled)
		
		// Plugin should be in error state
		infos := registry.List()
		require.Len(t, infos, 1)
		assert.Equal(t, PluginStateError, infos[0].State)
	})
	
	t.Run("unload plugin", func(t *testing.T) {
		registry := NewPluginRegistry(logger)
		plugin := newMockPlugin("test-plugin", PluginTypeHandler)
		
		registry.Register(plugin)
		registry.Load(context.Background(), "test-plugin", nil)
		
		// Unload
		err := registry.Unload(context.Background(), "test-plugin")
		assert.NoError(t, err)
		assert.True(t, plugin.stopCalled)
		
		// Plugin should be unloaded
		infos := registry.List()
		require.Len(t, infos, 1)
		assert.Equal(t, PluginStateUnloaded, infos[0].State)
		
		// Try to get unloaded plugin
		_, err = registry.Get("test-plugin")
		assert.Error(t, err)
		assert.Equal(t, ErrPluginDisabled, err)
	})
	
	t.Run("get by type", func(t *testing.T) {
		registry := NewPluginRegistry(logger)
		
		// Register and load multiple plugins
		middleware1 := newMockMiddlewarePlugin("middleware1")
		middleware2 := newMockMiddlewarePlugin("middleware2")
		tool1 := newMockToolPlugin("tool1")
		
		registry.Register(middleware1)
		registry.Register(middleware2)
		registry.Register(tool1)
		
		registry.Load(context.Background(), "middleware1", nil)
		registry.Load(context.Background(), "middleware2", nil)
		registry.Load(context.Background(), "tool1", nil)
		
		// Get middleware plugins
		middlewares := registry.GetByType(PluginTypeMiddleware)
		assert.Len(t, middlewares, 2)
		
		// Get tool plugins
		tools := registry.GetByType(PluginTypeTool)
		assert.Len(t, tools, 1)
		
		// Get non-existent type
		handlers := registry.GetByType(PluginTypeHandler)
		assert.Len(t, handlers, 0)
	})
	
	t.Run("list plugins", func(t *testing.T) {
		registry := NewPluginRegistry(logger)
		
		// Register multiple plugins
		plugin1 := newMockPlugin("plugin1", PluginTypeHandler)
		plugin2 := newMockPlugin("plugin2", PluginTypeMiddleware)
		
		registry.Register(plugin1)
		registry.Register(plugin2)
		registry.Load(context.Background(), "plugin1", map[string]interface{}{"key": "value"})
		
		// List all plugins
		infos := registry.List()
		assert.Len(t, infos, 2)
		
		// Find loaded plugin
		var loadedInfo *PluginInfo
		for _, info := range infos {
			if info.Name == "plugin1" {
				loadedInfo = info
				break
			}
		}
		
		require.NotNil(t, loadedInfo)
		assert.Equal(t, PluginStateEnabled, loadedInfo.State)
		assert.Equal(t, "1.0.0", loadedInfo.Version)
		assert.Equal(t, map[string]interface{}{"key": "value"}, loadedInfo.Config)
		assert.False(t, loadedInfo.LoadedAt.IsZero())
	})
	
	t.Run("health check", func(t *testing.T) {
		registry := NewPluginRegistry(logger)
		
		plugin1 := newMockPlugin("plugin1", PluginTypeHandler)
		plugin2 := newMockPlugin("plugin2", PluginTypeHandler)
		plugin2.healthError = errors.New("unhealthy")
		
		registry.Register(plugin1)
		registry.Register(plugin2)
		registry.Load(context.Background(), "plugin1", nil)
		registry.Load(context.Background(), "plugin2", nil)
		
		// Check health
		health := registry.Health()
		assert.Len(t, health, 2)
		assert.NoError(t, health["plugin1"])
		assert.Error(t, health["plugin2"])
		assert.Equal(t, "unhealthy", health["plugin2"].Error())
	})
}

func TestPluginHooks(t *testing.T) {
	logger := slog.Default()
	
	t.Run("load hooks", func(t *testing.T) {
		registry := NewPluginRegistry(logger)
		
		var loadedPlugins []Plugin
		registry.OnLoad(func(p Plugin) {
			loadedPlugins = append(loadedPlugins, p)
		})
		
		plugin := newMockPlugin("test-plugin", PluginTypeHandler)
		registry.Register(plugin)
		registry.Load(context.Background(), "test-plugin", nil)
		
		assert.Len(t, loadedPlugins, 1)
		assert.Equal(t, plugin, loadedPlugins[0])
	})
	
	t.Run("unload hooks", func(t *testing.T) {
		registry := NewPluginRegistry(logger)
		
		var unloadedPlugins []Plugin
		registry.OnUnload(func(p Plugin) {
			unloadedPlugins = append(unloadedPlugins, p)
		})
		
		plugin := newMockPlugin("test-plugin", PluginTypeHandler)
		registry.Register(plugin)
		registry.Load(context.Background(), "test-plugin", nil)
		registry.Unregister("test-plugin")
		
		assert.Len(t, unloadedPlugins, 1)
		assert.Equal(t, plugin, unloadedPlugins[0])
	})
	
	t.Run("error hooks", func(t *testing.T) {
		registry := NewPluginRegistry(logger)
		
		var errorPlugins []Plugin
		var capturedErrors []error
		registry.OnError(func(p Plugin, err error) {
			errorPlugins = append(errorPlugins, p)
			capturedErrors = append(capturedErrors, err)
		})
		
		plugin := newMockPlugin("test-plugin", PluginTypeHandler)
		plugin.initError = errors.New("init error")
		
		registry.Register(plugin)
		registry.Load(context.Background(), "test-plugin", nil)
		
		assert.Len(t, errorPlugins, 1)
		assert.Equal(t, plugin, errorPlugins[0])
		assert.Len(t, capturedErrors, 1)
		assert.Equal(t, "init error", capturedErrors[0].Error())
	})
}

// mockLoader implements PluginLoader for testing
type mockLoader struct {
	plugins map[string]Plugin
}

func (m *mockLoader) Load(path string) (Plugin, error) {
	if plugin, ok := m.plugins[path]; ok {
		return plugin, nil
	}
	return nil, fmt.Errorf("plugin not found: %s", path)
}

func TestPluginManager(t *testing.T) {
	logger := slog.Default()
	
	loader := &mockLoader{
		plugins: make(map[string]Plugin),
	}
	
	t.Run("load from path", func(t *testing.T) {
		manager := NewPluginManager(loader, logger)
		
		plugin := newMockPlugin("loaded-plugin", PluginTypeHandler)
		loader.plugins["/path/to/plugin"] = plugin
		
		err := manager.LoadFromPath(context.Background(), "/path/to/plugin", nil)
		assert.NoError(t, err)
		
		// Verify plugin was loaded
		retrieved, err := manager.Registry().Get("loaded-plugin")
		assert.NoError(t, err)
		assert.Equal(t, plugin, retrieved)
	})
	
	t.Run("load from path with error", func(t *testing.T) {
		manager := NewPluginManager(loader, logger)
		
		err := manager.LoadFromPath(context.Background(), "/non/existent", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "plugin not found")
	})
}

func TestConcurrentPluginOperations(t *testing.T) {
	logger := slog.Default()
	registry := NewPluginRegistry(logger)
	
	// Register multiple plugins
	for i := 0; i < 10; i++ {
		plugin := newMockPlugin(fmt.Sprintf("plugin%d", i), PluginTypeHandler)
		registry.Register(plugin)
	}
	
	// Concurrent operations
	var wg sync.WaitGroup
	errors := make(chan error, 100)
	
	// Load plugins concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			err := registry.Load(context.Background(), fmt.Sprintf("plugin%d", index), nil)
			if err != nil {
				errors <- err
			}
		}(i)
	}
	
	// Get plugins concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			// Wait a bit for plugins to load
			time.Sleep(10 * time.Millisecond)
			_, err := registry.Get(fmt.Sprintf("plugin%d", index))
			if err != nil && err != ErrPluginDisabled {
				errors <- err
			}
		}(i)
	}
	
	// List plugins concurrently
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			infos := registry.List()
			if len(infos) == 0 {
				errors <- fmt.Errorf("no plugins found")
			}
		}()
	}
	
	wg.Wait()
	close(errors)
	
	// Check for errors
	var errorCount int
	for err := range errors {
		t.Errorf("concurrent operation error: %v", err)
		errorCount++
	}
	assert.Equal(t, 0, errorCount)
}

func TestBasePlugin(t *testing.T) {
	logger := slog.Default()
	
	t.Run("basic functionality", func(t *testing.T) {
		plugin := NewBasePlugin("test", "1.0.0", "Test plugin", PluginTypeHandler, logger)
		
		assert.Equal(t, "test", plugin.Name())
		assert.Equal(t, "1.0.0", plugin.Version())
		assert.Equal(t, "Test plugin", plugin.Description())
		assert.Equal(t, PluginTypeHandler, plugin.Type())
		
		// Init
		config := map[string]interface{}{"key": "value"}
		err := plugin.Init(context.Background(), config)
		assert.NoError(t, err)
		
		// Config should be stored
		storedConfig := plugin.Config()
		assert.Equal(t, config, storedConfig)
		
		// Start
		err = plugin.Start(context.Background())
		assert.NoError(t, err)
		
		// Health
		err = plugin.Health()
		assert.NoError(t, err)
		
		// Stop
		err = plugin.Stop(context.Background())
		assert.NoError(t, err)
	})
}

func BenchmarkPluginRegistry(b *testing.B) {
	logger := slog.Default()
	registry := NewPluginRegistry(logger)
	
	// Register and load some plugins
	for i := 0; i < 10; i++ {
		plugin := newMockPlugin(fmt.Sprintf("plugin%d", i), PluginTypeHandler)
		registry.Register(plugin)
		registry.Load(context.Background(), plugin.Name(), nil)
	}
	
	b.Run("Get", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			registry.Get(fmt.Sprintf("plugin%d", i%10))
		}
	})
	
	b.Run("GetByType", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			registry.GetByType(PluginTypeHandler)
		}
	})
	
	b.Run("List", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			registry.List()
		}
	})
	
	b.Run("Health", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			registry.Health()
		}
	})
}

func BenchmarkConcurrentGet(b *testing.B) {
	logger := slog.Default()
	registry := NewPluginRegistry(logger)
	
	// Register and load a plugin
	plugin := newMockPlugin("test-plugin", PluginTypeHandler)
	registry.Register(plugin)
	registry.Load(context.Background(), "test-plugin", nil)
	
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			registry.Get("test-plugin")
		}
	})
}