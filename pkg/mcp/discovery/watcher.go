package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mcp-memory/pkg/mcp/protocol"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// PluginManifest describes a plugin's MCP capabilities
type PluginManifest struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Author      string            `json:"author"`
	Tools       []protocol.Tool     `json:"tools"`
	Resources   []protocol.Resource `json:"resources"`
	Prompts     []protocol.Prompt   `json:"prompts"`
	Tags        []string            `json:"tags"`
}

// PluginWatcher watches a directory for MCP plugin manifests
type PluginWatcher struct {
	path         string
	scanInterval time.Duration
	registry     *Registry
	plugins      map[string]*PluginManifest
	mutex        sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

// NewPluginWatcher creates a new plugin watcher
func NewPluginWatcher(path string, scanInterval time.Duration, registry *Registry) (*PluginWatcher, error) {
	// Ensure the plugin directory exists
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("failed to create plugin directory: %w", err)
	}

	return &PluginWatcher{
		path:         path,
		scanInterval: scanInterval,
		registry:     registry,
		plugins:      make(map[string]*PluginManifest),
	}, nil
}

// Start begins watching for plugins
func (w *PluginWatcher) Start(ctx context.Context) error {
	w.ctx, w.cancel = context.WithCancel(ctx)
	
	// Do initial scan
	if err := w.scan(); err != nil {
		return fmt.Errorf("initial scan failed: %w", err)
	}
	
	// Start periodic scanning
	w.wg.Add(1)
	go w.watchLoop()
	
	return nil
}

// Stop stops watching for plugins
func (w *PluginWatcher) Stop() error {
	if w.cancel != nil {
		w.cancel()
	}
	w.wg.Wait()
	return nil
}

// watchLoop periodically scans for plugin changes
func (w *PluginWatcher) watchLoop() {
	defer w.wg.Done()
	
	ticker := time.NewTicker(w.scanInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			if err := w.scan(); err != nil {
				// Log error but continue watching
				fmt.Printf("Plugin scan error: %v\n", err)
			}
		}
	}
}

// scan looks for plugin manifest files
func (w *PluginWatcher) scan() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	
	// Track which plugins we've seen in this scan
	seenPlugins := make(map[string]bool)
	
	// Look for manifest files
	pattern := filepath.Join(w.path, "*", "mcp-manifest.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to glob for manifests: %w", err)
	}
	
	// Also check root directory
	rootManifest := filepath.Join(w.path, "mcp-manifest.json")
	if _, err := os.Stat(rootManifest); err == nil {
		matches = append(matches, rootManifest)
	}
	
	// Process each manifest
	for _, manifestPath := range matches {
		pluginID := filepath.Dir(manifestPath)
		if pluginID == w.path {
			pluginID = filepath.Base(w.path)
		} else {
			pluginID = filepath.Base(pluginID)
		}
		
		seenPlugins[pluginID] = true
		
		// Check if plugin is new or updated
		_, err := os.Stat(manifestPath)
		if err != nil {
			continue
		}
		
		existingPlugin, exists := w.plugins[pluginID]
		if exists {
			// Check if manifest was modified
			// For simplicity, we'll re-read all manifests each scan
			// In production, you'd check modification time
		}
		
		// Load manifest
		manifest, err := w.loadManifest(manifestPath)
		if err != nil {
			fmt.Printf("Failed to load manifest %s: %v\n", manifestPath, err)
			continue
		}
		
		// Register or update plugin
		if !exists {
			// New plugin
			if err := w.registerPlugin(pluginID, manifest); err != nil {
				fmt.Printf("Failed to register plugin %s: %v\n", pluginID, err)
				continue
			}
		} else if !w.manifestsEqual(existingPlugin, manifest) {
			// Updated plugin
			if err := w.updatePlugin(pluginID, manifest); err != nil {
				fmt.Printf("Failed to update plugin %s: %v\n", pluginID, err)
				continue
			}
		}
		
		w.plugins[pluginID] = manifest
	}
	
	// Unregister plugins that are no longer present
	for pluginID := range w.plugins {
		if !seenPlugins[pluginID] {
			if err := w.unregisterPlugin(pluginID); err != nil {
				fmt.Printf("Failed to unregister plugin %s: %v\n", pluginID, err)
			}
			delete(w.plugins, pluginID)
		}
	}
	
	return nil
}

// loadManifest loads a plugin manifest from disk
func (w *PluginWatcher) loadManifest(path string) (*PluginManifest, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}
	
	var manifest PluginManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}
	
	return &manifest, nil
}

// registerPlugin registers all items from a plugin
func (w *PluginWatcher) registerPlugin(pluginID string, manifest *PluginManifest) error {
	source := fmt.Sprintf("plugin:%s", pluginID)
	
	// Register tools
	for _, tool := range manifest.Tools {
		// Note: In a real implementation, you'd need to load the actual tool handler
		// This would typically involve loading a shared library or connecting to a subprocess
		if err := w.registry.RegisterTool(tool, nil, source, manifest.Tags); err != nil {
			return fmt.Errorf("failed to register tool %s: %w", tool.Name, err)
		}
	}
	
	// Register resources
	for _, resource := range manifest.Resources {
		if err := w.registry.RegisterResource(resource, source, manifest.Tags); err != nil {
			return fmt.Errorf("failed to register resource %s: %w", resource.URI, err)
		}
	}
	
	// Register prompts
	for _, prompt := range manifest.Prompts {
		if err := w.registry.RegisterPrompt(prompt, source, manifest.Tags); err != nil {
			return fmt.Errorf("failed to register prompt %s: %w", prompt.Name, err)
		}
	}
	
	return nil
}

// updatePlugin updates a plugin's registrations
func (w *PluginWatcher) updatePlugin(pluginID string, manifest *PluginManifest) error {
	// For simplicity, unregister old and register new
	if err := w.unregisterPlugin(pluginID); err != nil {
		return err
	}
	return w.registerPlugin(pluginID, manifest)
}

// unregisterPlugin removes all items from a plugin
func (w *PluginWatcher) unregisterPlugin(pluginID string) error {
	
	if manifest, exists := w.plugins[pluginID]; exists {
		// Unregister tools
		for _, tool := range manifest.Tools {
			w.registry.UnregisterTool(tool.Name)
		}
		
		// Unregister resources
		for _, resource := range manifest.Resources {
			w.registry.UnregisterResource(resource.URI)
		}
		
		// Unregister prompts
		for _, prompt := range manifest.Prompts {
			w.registry.UnregisterPrompt(prompt.Name)
		}
	}
	
	return nil
}

// manifestsEqual checks if two manifests are equal
func (w *PluginWatcher) manifestsEqual(a, b *PluginManifest) bool {
	// Simple comparison - in production you'd want a more sophisticated check
	aJSON, _ := json.Marshal(a)
	bJSON, _ := json.Marshal(b)
	return string(aJSON) == string(bJSON)
}