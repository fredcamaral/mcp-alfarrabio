package visualization

import (
	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
)

// visualizerAdapter adapts visualization.Visualizer to ports.Visualizer interface
type visualizerAdapter struct {
	terminalViz Visualizer
}

// NewVisualizerAdapter creates a new visualizer adapter
func NewVisualizerAdapter(terminalViz Visualizer) ports.Visualizer {
	return &visualizerAdapter{
		terminalViz: terminalViz,
	}
}

// RenderProductivityChart adapts the productivity chart method
func (v *visualizerAdapter) RenderProductivityChart(metrics *entities.WorkflowMetrics) ([]byte, error) {
	if metrics == nil {
		return []byte("No productivity metrics available"), nil
	}
	result := v.terminalViz.RenderProductivityChart(metrics.Productivity)
	return []byte(result), nil
}

// RenderVelocityChart adapts the velocity chart method
func (v *visualizerAdapter) RenderVelocityChart(metrics *entities.VelocityMetrics) ([]byte, error) {
	if metrics == nil {
		return []byte("No velocity metrics available"), nil
	}
	result := v.terminalViz.RenderVelocityChart(*metrics)
	return []byte(result), nil
}

// RenderCycleTimeChart adapts the cycle time chart method
func (v *visualizerAdapter) RenderCycleTimeChart(metrics *entities.CycleTimeMetrics) ([]byte, error) {
	if metrics == nil {
		return []byte("No cycle time metrics available"), nil
	}
	result := v.terminalViz.RenderCycleTimeChart(*metrics)
	return []byte(result), nil
}

// RenderBottlenecks adapts the bottlenecks visualization method
func (v *visualizerAdapter) RenderBottlenecks(bottlenecks []*entities.Bottleneck) ([]byte, error) {
	result := v.terminalViz.RenderBottlenecks(bottlenecks)
	return []byte(result), nil
}

// GenerateVisualization adapts the general visualization method
func (v *visualizerAdapter) GenerateVisualization(metrics *entities.WorkflowMetrics, format entities.VisFormat) ([]byte, error) {
	return v.terminalViz.GenerateVisualization(metrics, format)
}
