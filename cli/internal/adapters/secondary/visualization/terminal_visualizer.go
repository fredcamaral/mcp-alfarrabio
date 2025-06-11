package visualization

import (
	"lerian-mcp-memory-cli/internal/domain/entities"
)

// TerminalVisualizer implements visualization for terminal output
// This is a simple wrapper around the full terminal visualizer implementation
type TerminalVisualizer struct {
	impl Visualizer
}

// NewSimpleTerminalVisualizer creates a new simple terminal visualizer
func NewSimpleTerminalVisualizer() Visualizer {
	return &TerminalVisualizer{
		impl: NewTerminalVisualizer(DefaultTerminalVisualizerConfig()),
	}
}

// GenerateVisualization creates terminal-based visualization
func (v *TerminalVisualizer) GenerateVisualization(metrics *entities.WorkflowMetrics, format entities.VisFormat) ([]byte, error) {
	return v.impl.GenerateVisualization(metrics, format)
}

// RenderProductivityChart renders productivity chart
func (v *TerminalVisualizer) RenderProductivityChart(metrics entities.ProductivityMetrics) string {
	return v.impl.RenderProductivityChart(metrics)
}

// RenderVelocityChart renders velocity chart
func (v *TerminalVisualizer) RenderVelocityChart(metrics entities.VelocityMetrics) string {
	return v.impl.RenderVelocityChart(metrics)
}

// RenderBottlenecks renders bottleneck visualization
func (v *TerminalVisualizer) RenderBottlenecks(bottlenecks []*entities.Bottleneck) string {
	return v.impl.RenderBottlenecks(bottlenecks)
}

// RenderCycleTimeChart renders cycle time chart
func (v *TerminalVisualizer) RenderCycleTimeChart(metrics entities.CycleTimeMetrics) string {
	return v.impl.RenderCycleTimeChart(metrics)
}

// RenderCompletionChart renders completion chart
func (v *TerminalVisualizer) RenderCompletionChart(metrics entities.CompletionMetrics) string {
	return v.impl.RenderCompletionChart(metrics)
}

// RenderTrends renders trends visualization
func (v *TerminalVisualizer) RenderTrends(trends entities.TrendAnalysis) string {
	return v.impl.RenderTrends(trends)
}
