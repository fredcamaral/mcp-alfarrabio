package tui

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type REPLMode string

const (
	Interactive REPLMode = "interactive"
	Dashboard   REPLMode = "dashboard"
	Analytics   REPLMode = "analytics"
	Workflow    REPLMode = "workflow"
	Debug       REPLMode = "debug"
)

type ViewMode string

const (
	ViewModeCommand   ViewMode = "command"
	ViewModeDashboard ViewMode = "dashboard"
	ViewModeAnalytics ViewMode = "analytics"
	ViewModeTaskList  ViewMode = "tasklist"
	ViewModePatterns  ViewMode = "patterns"
	ViewModeInsights  ViewMode = "insights"
)

type REPLModel struct {
	mode         REPLMode
	viewMode     ViewMode
	httpPort     int
	server       *http.Server
	input        string
	output       []string
	cursor       int
	width        int
	height       int
	style        lipgloss.Style
	focused      bool
	history      []string
	historyIndex int

	// Dashboard data
	dashboardData *DashboardData
	selectedRepo  string
	repositories  []string
	activePane    int

	// Analytics data
	analyticsData *AnalyticsData
	chartType     string
	timeRange     string
}

type DashboardData struct {
	TaskStats      TaskStats
	RepoMetrics    map[string]RepoMetrics
	RecentTasks    []TaskSummary
	TopPatterns    []PatternSummary
	CrossRepoStats CrossRepoStats
}

type TaskStats struct {
	Total      int
	Completed  int
	InProgress int
	Blocked    int
	TodayCount int
	WeekCount  int
}

type RepoMetrics struct {
	Name           string
	TaskCount      int
	Productivity   float64
	Velocity       float64
	CompletionRate float64
	LastUpdate     time.Time
}

type TaskSummary struct {
	ID            string
	Title         string
	Status        string
	Priority      string
	Repository    string
	CreatedAt     time.Time
	EstimatedMins int
}

type PatternSummary struct {
	Name        string
	Type        string
	Frequency   float64
	SuccessRate float64
	Repository  string
}

type CrossRepoStats struct {
	TotalRepos      int
	ActiveRepos     int
	CommonPatterns  int
	SharedInsights  int
	SimilarityScore float64
}

type AnalyticsData struct {
	ProductivityChart []ChartPoint
	VelocityChart     []ChartPoint
	CompletionChart   []ChartPoint
	PatternChart      []ChartPoint
	Correlations      map[string]float64
	Outliers          []OutlierInfo
}

type ChartPoint struct {
	Label string
	Value float64
	Date  time.Time
}

type OutlierInfo struct {
	Repository string
	Metric     string
	Value      float64
	Severity   string
}

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func NewREPLModel(mode REPLMode, httpPort int) *REPLModel {
	return &REPLModel{
		mode:         mode,
		viewMode:     ViewModeCommand,
		httpPort:     httpPort,
		output:       []string{},
		focused:      true,
		history:      []string{},
		historyIndex: -1,
		repositories: []string{},
		activePane:   0,
		chartType:    "productivity",
		timeRange:    "week",
		dashboardData: &DashboardData{
			RepoMetrics: make(map[string]RepoMetrics),
		},
		analyticsData: &AnalyticsData{
			Correlations: make(map[string]float64),
		},
		style: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1, 2),
	}
}

func (m REPLModel) Init() tea.Cmd {
	// Start HTTP server if port is specified
	if m.httpPort > 0 {
		go m.startHTTPServer()
	}

	return tea.Batch(
		tickCmd(),
		tea.EnterAltScreen,
	)
}

func (m REPLModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			if m.server != nil {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = m.server.Shutdown(ctx)
			}
			return m, tea.Quit

		// View mode switching
		case "F1":
			m.viewMode = ViewModeCommand
			return m, nil
		case "F2":
			m.viewMode = ViewModeDashboard
			m.loadDashboardData()
			return m, nil
		case "F3":
			m.viewMode = ViewModeAnalytics
			m.loadAnalyticsData()
			return m, nil
		case "F4":
			m.viewMode = ViewModeTaskList
			return m, nil
		case "F5":
			m.viewMode = ViewModePatterns
			return m, nil
		case "F6":
			m.viewMode = ViewModeInsights
			return m, nil

		// Navigation in dashboard/analytics modes
		case "tab":
			if m.viewMode == ViewModeDashboard || m.viewMode == ViewModeAnalytics {
				m.activePane = (m.activePane + 1) % 4
				return m, nil
			}

		// Handle input based on current view mode
		default:
			if m.viewMode == ViewModeCommand {
				return m.handleCommandInput(msg), nil
			} else {
				return m.handleNavigationInput(msg), nil
			}
		}

	case tickMsg:
		return m, tickCmd()
	}

	return m, nil
}

// handleCommandInput handles input in command mode
func (m REPLModel) handleCommandInput(msg tea.KeyMsg) REPLModel {
	switch msg.String() {
	case "enter":
		if m.input != "" {
			return m.executeCommand(m.input)
		}
		return m

	case "up":
		if len(m.history) > 0 && m.historyIndex < len(m.history)-1 {
			m.historyIndex++
			m.input = m.history[len(m.history)-1-m.historyIndex]
			m.cursor = len(m.input)
		}
		return m

	case "down":
		if m.historyIndex > 0 {
			m.historyIndex--
			m.input = m.history[len(m.history)-1-m.historyIndex]
			m.cursor = len(m.input)
		} else if m.historyIndex == 0 {
			m.historyIndex = -1
			m.input = ""
			m.cursor = 0
		}
		return m

	case "left":
		if m.cursor > 0 {
			m.cursor--
		}
		return m

	case "right":
		if m.cursor < len(m.input) {
			m.cursor++
		}
		return m

	case "backspace":
		if m.cursor > 0 {
			m.input = m.input[:m.cursor-1] + m.input[m.cursor:]
			m.cursor--
		}
		return m

	case "delete":
		if m.cursor < len(m.input) {
			m.input = m.input[:m.cursor] + m.input[m.cursor+1:]
		}
		return m

	default:
		if len(msg.String()) == 1 {
			m.input = m.input[:m.cursor] + msg.String() + m.input[m.cursor:]
			m.cursor++
		}
		return m
	}
}

// handleNavigationInput handles input in dashboard/analytics modes
func (m REPLModel) handleNavigationInput(msg tea.KeyMsg) REPLModel {
	switch msg.String() {
	case "j", "down":
		// Navigate down in current pane
		return m
	case "k", "up":
		// Navigate up in current pane
		return m
	case "h", "left":
		// Navigate left between panes
		if m.activePane > 0 {
			m.activePane--
		}
		return m
	case "l", "right":
		// Navigate right between panes
		if m.activePane < 3 {
			m.activePane++
		}
		return m
	case "r":
		// Refresh data
		if m.viewMode == ViewModeDashboard {
			m.loadDashboardData()
		} else if m.viewMode == ViewModeAnalytics {
			m.loadAnalyticsData()
		}
		return m
	case "1", "2", "3", "4":
		// Quick chart switching in analytics mode
		if m.viewMode == ViewModeAnalytics {
			charts := []string{"productivity", "velocity", "completion", "patterns"}
			if idx := int(msg.String()[0] - '1'); idx < len(charts) {
				m.chartType = charts[idx]
			}
		}
		return m
	case "d", "w", "m":
		// Time range switching
		timeRanges := map[string]string{"d": "day", "w": "week", "m": "month"}
		if tr, ok := timeRanges[msg.String()]; ok {
			m.timeRange = tr
			if m.viewMode == ViewModeAnalytics {
				m.loadAnalyticsData()
			}
		}
		return m
	}
	return m
}

// loadDashboardData loads dashboard data (mock implementation)
func (m *REPLModel) loadDashboardData() {
	// In real implementation, this would call the analytics engine
	m.dashboardData = &DashboardData{
		TaskStats: TaskStats{
			Total:      156,
			Completed:  98,
			InProgress: 34,
			Blocked:    24,
			TodayCount: 8,
			WeekCount:  45,
		},
		RepoMetrics: map[string]RepoMetrics{
			"lerian-mcp-memory": {
				Name:           "lerian-mcp-memory",
				TaskCount:      89,
				Productivity:   87.3,
				Velocity:       12.4,
				CompletionRate: 0.84,
				LastUpdate:     time.Now(),
			},
			"web-frontend": {
				Name:           "web-frontend",
				TaskCount:      67,
				Productivity:   78.9,
				Velocity:       9.8,
				CompletionRate: 0.76,
				LastUpdate:     time.Now().Add(-2 * time.Hour),
			},
		},
		CrossRepoStats: CrossRepoStats{
			TotalRepos:      5,
			ActiveRepos:     3,
			CommonPatterns:  12,
			SharedInsights:  8,
			SimilarityScore: 0.73,
		},
	}

	// Populate repositories list
	m.repositories = make([]string, 0, len(m.dashboardData.RepoMetrics))
	for repo := range m.dashboardData.RepoMetrics {
		m.repositories = append(m.repositories, repo)
	}
	if len(m.repositories) > 0 && m.selectedRepo == "" {
		m.selectedRepo = m.repositories[0]
	}
}

// loadAnalyticsData loads analytics data (mock implementation)
func (m *REPLModel) loadAnalyticsData() {
	// Mock data for charts
	m.analyticsData = &AnalyticsData{
		ProductivityChart: []ChartPoint{
			{Label: "Mon", Value: 85.2, Date: time.Now().AddDate(0, 0, -6)},
			{Label: "Tue", Value: 78.5, Date: time.Now().AddDate(0, 0, -5)},
			{Label: "Wed", Value: 92.1, Date: time.Now().AddDate(0, 0, -4)},
			{Label: "Thu", Value: 88.7, Date: time.Now().AddDate(0, 0, -3)},
			{Label: "Fri", Value: 79.3, Date: time.Now().AddDate(0, 0, -2)},
			{Label: "Sat", Value: 71.8, Date: time.Now().AddDate(0, 0, -1)},
			{Label: "Sun", Value: 83.4, Date: time.Now()},
		},
		VelocityChart: []ChartPoint{
			{Label: "Week 1", Value: 12.4, Date: time.Now().AddDate(0, 0, -21)},
			{Label: "Week 2", Value: 15.2, Date: time.Now().AddDate(0, 0, -14)},
			{Label: "Week 3", Value: 11.8, Date: time.Now().AddDate(0, 0, -7)},
			{Label: "Week 4", Value: 13.9, Date: time.Now()},
		},
		Correlations: map[string]float64{
			"productivity_velocity":   0.73,
			"productivity_completion": 0.82,
			"velocity_completion":     0.64,
		},
		Outliers: []OutlierInfo{
			{Repository: "legacy-app", Metric: "productivity", Value: 45.2, Severity: "high"},
			{Repository: "experimental", Metric: "velocity", Value: 22.1, Severity: "medium"},
		},
	}
}

func (m REPLModel) executeCommand(cmd string) REPLModel {
	// Add to history
	if cmd != "" && (len(m.history) == 0 || m.history[len(m.history)-1] != cmd) {
		m.history = append(m.history, cmd)
		if len(m.history) > 100 { // Keep last 100 commands
			m.history = m.history[1:]
		}
	}
	m.historyIndex = -1

	// Add command to output
	m.output = append(m.output, fmt.Sprintf("lmmc> %s", cmd))

	// Process command
	switch {
	case cmd == "help":
		m.output = append(m.output, m.getHelpText()...)
	case cmd == "clear":
		m.output = []string{}
	case cmd == "exit" || cmd == "quit":
		if m.server != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = m.server.Shutdown(ctx)
		}
		return m
	case cmd == "dashboard" || cmd == "dash":
		m.viewMode = ViewModeDashboard
		m.loadDashboardData()
		m.output = append(m.output, "Switched to dashboard view (F2)")
	case cmd == "analytics" || cmd == "stats":
		m.viewMode = ViewModeAnalytics
		m.loadAnalyticsData()
		m.output = append(m.output, "Switched to analytics view (F3)")
	case cmd == "tasks" || cmd == "list":
		m.viewMode = ViewModeTaskList
		m.output = append(m.output, "Switched to task list view (F4)")
	case cmd == "patterns":
		m.viewMode = ViewModePatterns
		m.output = append(m.output, "Switched to patterns view (F5)")
	case cmd == "insights":
		m.viewMode = ViewModeInsights
		m.output = append(m.output, "Switched to insights view (F6)")
	case cmd == "command" || cmd == "cmd":
		m.viewMode = ViewModeCommand
		m.output = append(m.output, "Switched to command mode (F1)")
	case strings.HasPrefix(cmd, "prd create"):
		m.output = append(m.output, "Starting interactive PRD creation...")
		m.output = append(m.output, "Feature not yet implemented - coming soon!")
	case strings.HasPrefix(cmd, "trd create"):
		m.output = append(m.output, "Starting TRD generation from PRD...")
		m.output = append(m.output, "Feature not yet implemented - coming soon!")
	case strings.HasPrefix(cmd, "workflow run"):
		m.output = append(m.output, "Running complete workflow automation...")
		m.output = append(m.output, "Feature not yet implemented - coming soon!")
	case strings.HasPrefix(cmd, "status"):
		m.output = append(m.output, m.getStatusInfo()...)
	default:
		m.output = append(m.output, fmt.Sprintf("Unknown command: %s", cmd))
		m.output = append(m.output, "Type 'help' for available commands")
	}

	// Clear input
	m.input = ""
	m.cursor = 0

	return m
}

func (m REPLModel) getHelpText() []string {
	return []string{
		"",
		"LMMC TUI - Interactive Multi-Repository Intelligence Platform",
		"",
		"View Navigation:",
		"  F1 / command      Switch to command mode",
		"  F2 / dashboard    Switch to dashboard view",
		"  F3 / analytics    Switch to analytics view",
		"  F4 / tasks        Switch to task list view",
		"  F5 / patterns     Switch to patterns view",
		"  F6 / insights     Switch to insights view",
		"",
		"Dashboard Navigation:",
		"  Tab               Switch between panes",
		"  h/j/k/l          Vim-style navigation",
		"  r                 Refresh data",
		"  1-4              Quick chart switching (analytics)",
		"  d/w/m            Day/Week/Month time range",
		"",
		"Basic Commands:",
		"  help              Show this help message",
		"  status            Show current status and configuration",
		"  clear             Clear the output",
		"  exit, quit        Exit the TUI",
		"",
		"Document Generation:",
		"  prd create        Start interactive PRD creation",
		"  trd create        Generate TRD from existing PRD",
		"  workflow run      Run complete automation workflow",
		"",
		"Key Features:",
		"  📊 Real-time multi-repo dashboard",
		"  📈 Advanced analytics with ASCII charts",
		"  🔄 Pattern detection and workflow analysis",
		"  💡 Cross-repository insights and recommendations",
		"  📋 Interactive task management",
		"",
	}
}

func (m REPLModel) getStatusInfo() []string {
	status := []string{
		"",
		"REPL Status:",
		fmt.Sprintf("  Mode: %s", m.mode),
	}

	if m.httpPort > 0 {
		status = append(status, fmt.Sprintf("  HTTP Server: Running on port %d", m.httpPort))
	} else {
		status = append(status, "  HTTP Server: Disabled")
	}

	status = append(status, fmt.Sprintf("  Commands in history: %d", len(m.history)))
	status = append(status, "")

	return status
}

func (m REPLModel) View() string {
	switch m.viewMode {
	case ViewModeCommand:
		return m.renderCommandView()
	case ViewModeDashboard:
		return m.renderDashboardView()
	case ViewModeAnalytics:
		return m.renderAnalyticsView()
	case ViewModeTaskList:
		return m.renderTaskListView()
	case ViewModePatterns:
		return m.renderPatternsView()
	case ViewModeInsights:
		return m.renderInsightsView()
	default:
		return m.renderCommandView()
	}
}

func (m REPLModel) renderCommandView() string {
	var view strings.Builder

	// Header with mode indicator
	header := m.renderHeader("Command Mode")
	view.WriteString(header + "\n\n")

	// Output area
	outputHeight := m.height - 6
	if outputHeight < 1 {
		outputHeight = 1
	}

	outputLines := m.output
	if len(outputLines) > outputHeight {
		outputLines = outputLines[len(outputLines)-outputHeight:]
	}

	for _, line := range outputLines {
		view.WriteString(line + "\n")
	}

	// Fill remaining space
	for i := len(outputLines); i < outputHeight; i++ {
		view.WriteString("\n")
	}

	// Input line
	promptStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Bold(true)

	prompt := promptStyle.Render("lmmc> ")
	inputDisplay := m.input

	// Add cursor
	if m.cursor < len(inputDisplay) {
		inputDisplay = inputDisplay[:m.cursor] + "█" + inputDisplay[m.cursor+1:]
	} else {
		inputDisplay += "█"
	}

	view.WriteString(prompt + inputDisplay)

	// Footer
	footer := m.renderFooter("F1:Command F2:Dashboard F3:Analytics F4:Tasks F5:Patterns F6:Insights | Ctrl+C:Exit")
	view.WriteString("\n" + footer)

	return view.String()
}

func (m REPLModel) renderDashboardView() string {
	if m.dashboardData == nil {
		return "Loading dashboard data..."
	}

	var view strings.Builder

	// Header
	header := m.renderHeader("Dashboard - Multi-Repository Overview")
	view.WriteString(header + "\n\n")

	// Calculate layout dimensions
	panelWidth := (m.width - 6) / 2
	panelHeight := (m.height - 8) / 2

	// Top row: Task Stats + Repository Metrics
	topRow := lipgloss.JoinHorizontal(lipgloss.Top,
		m.renderTaskStatsPanel(panelWidth, panelHeight),
		" ",
		m.renderRepoMetricsPanel(panelWidth, panelHeight),
	)
	view.WriteString(topRow + "\n\n")

	// Bottom row: Cross-Repo Stats + Recent Activity
	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top,
		m.renderCrossRepoPanel(panelWidth, panelHeight),
		" ",
		m.renderRecentActivityPanel(panelWidth, panelHeight),
	)
	view.WriteString(bottomRow + "\n")

	// Footer
	footer := m.renderFooter("Tab:Switch Pane | h/l:Navigate | r:Refresh | F1-F6:Switch Views")
	view.WriteString(footer)

	return view.String()
}

func (m REPLModel) renderAnalyticsView() string {
	if m.analyticsData == nil {
		return "Loading analytics data..."
	}

	var view strings.Builder

	// Header
	header := m.renderHeader(fmt.Sprintf("Analytics - %s Chart (%s)", strings.Title(m.chartType), strings.Title(m.timeRange)))
	view.WriteString(header + "\n\n")

	// Calculate layout dimensions
	chartWidth := m.width - 4
	chartHeight := (m.height - 12) / 2

	// Main chart
	chart := m.renderChart(m.chartType, chartWidth, chartHeight)
	view.WriteString(chart + "\n\n")

	// Bottom panels: Correlations + Outliers
	panelWidth := (m.width - 6) / 2
	panelHeight := chartHeight

	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top,
		m.renderCorrelationsPanel(panelWidth, panelHeight),
		" ",
		m.renderOutliersPanel(panelWidth, panelHeight),
	)
	view.WriteString(bottomRow + "\n")

	// Footer
	footer := m.renderFooter("1-4:Charts | d/w/m:TimeRange | r:Refresh | h/l:Navigate")
	view.WriteString(footer)

	return view.String()
}

func (m REPLModel) renderTaskListView() string {
	var view strings.Builder

	header := m.renderHeader("Task List - All Repositories")
	view.WriteString(header + "\n\n")

	// Mock task list
	taskListStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1).
		Width(m.width - 4)

	taskList := "📋 Recent Tasks:\n\n" +
		"🟢 [HIGH] Implement cross-repo analysis     lerian-mcp-memory\n" +
		"🟡 [MED]  Add TUI dashboard interface       lerian-mcp-memory\n" +
		"🔴 [HIGH] Fix authentication middleware     web-frontend\n" +
		"🟢 [LOW]  Update documentation             docs-site\n" +
		"🟡 [MED]  Optimize database queries        api-backend\n\n" +
		"Press Enter to view details, j/k to navigate"

	view.WriteString(taskListStyle.Render(taskList))

	footer := m.renderFooter("j/k:Navigate | Enter:Details | F1-F6:Switch Views")
	view.WriteString("\n" + footer)

	return view.String()
}

func (m REPLModel) renderPatternsView() string {
	var view strings.Builder

	header := m.renderHeader("Pattern Analysis - Detected Workflows")
	view.WriteString(header + "\n\n")

	// Mock patterns display
	patternsStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1).
		Width(m.width - 4)

	patterns := "🔄 Detected Patterns:\n\n" +
		"📈 Feature Development Workflow    (87% success, 15.2h avg)\n" +
		"  └─ Planning → Implementation → Testing → Review\n\n" +
		"🐛 Bug Fix Pattern               (94% success, 3.4h avg)\n" +
		"  └─ Investigation → Fix → Validation\n\n" +
		"🔧 Refactoring Sequence          (76% success, 8.7h avg)\n" +
		"  └─ Analysis → Cleanup → Testing → Documentation\n\n" +
		"📊 Release Preparation           (82% success, 12.1h avg)\n" +
		"  └─ Testing → Documentation → Deployment\n"

	view.WriteString(patternsStyle.Render(patterns))

	footer := m.renderFooter("j/k:Navigate | Enter:Details | F1-F6:Switch Views")
	view.WriteString("\n" + footer)

	return view.String()
}

func (m REPLModel) renderInsightsView() string {
	var view strings.Builder

	header := m.renderHeader("Cross-Repository Insights")
	view.WriteString(header + "\n\n")

	// Mock insights display
	insightsStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("220")).
		Padding(1).
		Width(m.width - 4)

	insights := "💡 Key Insights:\n\n" +
		"🎯 High Similarity Pattern Detected\n" +
		"   lerian-mcp-memory and web-frontend show 78% workflow similarity\n" +
		"   → Consider cross-team knowledge sharing\n\n" +
		"⚠️  Productivity Variance Alert\n" +
		"   40% performance difference across repositories\n" +
		"   → Review processes in underperforming teams\n\n" +
		"🚀 Optimization Opportunity\n" +
		"   Testing workflows could be streamlined by 25%\n" +
		"   → Implement automated testing patterns\n\n" +
		"📊 Best Practice Identified\n" +
		"   'Feature Planning' pattern shows 95% success rate\n" +
		"   → Adopt across all repositories\n"

	view.WriteString(insightsStyle.Render(insights))

	footer := m.renderFooter("j/k:Navigate | Enter:Details | F1-F6:Switch Views")
	view.WriteString("\n" + footer)

	return view.String()
}

// Panel rendering methods
func (m REPLModel) renderTaskStatsPanel(width, height int) string {
	style := m.getPanelStyle(width, height, m.activePane == 0)

	content := fmt.Sprintf("📊 Task Statistics\n\n"+
		"Total Tasks:     %d\n"+
		"✅ Completed:    %d (%.1f%%)\n"+
		"🔄 In Progress:  %d\n"+
		"⛔ Blocked:      %d\n\n"+
		"📅 Today:        %d tasks\n"+
		"📈 This Week:    %d tasks",
		m.dashboardData.TaskStats.Total,
		m.dashboardData.TaskStats.Completed,
		float64(m.dashboardData.TaskStats.Completed)/float64(m.dashboardData.TaskStats.Total)*100,
		m.dashboardData.TaskStats.InProgress,
		m.dashboardData.TaskStats.Blocked,
		m.dashboardData.TaskStats.TodayCount,
		m.dashboardData.TaskStats.WeekCount)

	return style.Render(content)
}

func (m REPLModel) renderRepoMetricsPanel(width, height int) string {
	style := m.getPanelStyle(width, height, m.activePane == 1)

	content := "🏛️ Repository Metrics\n\n"
	for _, repo := range m.repositories {
		metrics := m.dashboardData.RepoMetrics[repo]
		content += fmt.Sprintf("📁 %s\n", repo)
		content += fmt.Sprintf("   Productivity: %.1f%%\n", metrics.Productivity)
		content += fmt.Sprintf("   Velocity: %.1f tasks/week\n", metrics.Velocity)
		content += fmt.Sprintf("   Completion: %.1f%%\n\n", metrics.CompletionRate*100)
	}

	return style.Render(content)
}

func (m REPLModel) renderCrossRepoPanel(width, height int) string {
	style := m.getPanelStyle(width, height, m.activePane == 2)

	content := fmt.Sprintf("🔗 Cross-Repository Analysis\n\n"+
		"Total Repositories: %d\n"+
		"Active Projects:    %d\n"+
		"Common Patterns:    %d\n"+
		"Shared Insights:    %d\n\n"+
		"Similarity Score:   %.1f%%\n"+
		"Knowledge Sharing:  🟢 Active",
		m.dashboardData.CrossRepoStats.TotalRepos,
		m.dashboardData.CrossRepoStats.ActiveRepos,
		m.dashboardData.CrossRepoStats.CommonPatterns,
		m.dashboardData.CrossRepoStats.SharedInsights,
		m.dashboardData.CrossRepoStats.SimilarityScore*100)

	return style.Render(content)
}

func (m REPLModel) renderRecentActivityPanel(width, height int) string {
	style := m.getPanelStyle(width, height, m.activePane == 3)

	content := "⚡ Recent Activity\n\n" +
		"🔄 Pattern detected in web-frontend\n" +
		"   'API Integration' workflow\n\n" +
		"💡 Insight generated\n" +
		"   Cross-repo optimization opportunity\n\n" +
		"📊 Analytics updated\n" +
		"   Productivity metrics refreshed\n\n" +
		"🎯 Recommendation added\n" +
		"   Workflow standardization\n"

	return style.Render(content)
}

func (m REPLModel) renderChart(chartType string, width, height int) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Padding(1).
		Width(width)

	// Simple ASCII chart rendering
	var data []ChartPoint
	switch chartType {
	case "productivity":
		data = m.analyticsData.ProductivityChart
	case "velocity":
		data = m.analyticsData.VelocityChart
	case "completion":
		data = m.analyticsData.CompletionChart
	case "patterns":
		data = m.analyticsData.PatternChart
	default:
		data = m.analyticsData.ProductivityChart
	}

	chart := m.renderASCIIChart(data, width-4, height-4)
	return style.Render(fmt.Sprintf("📈 %s Trend\n\n%s", strings.Title(chartType), chart))
}

func (m REPLModel) renderASCIIChart(data []ChartPoint, width, height int) string {
	if len(data) == 0 {
		return "No data available"
	}

	// Find min/max values
	minVal, maxVal := data[0].Value, data[0].Value
	for _, point := range data {
		if point.Value < minVal {
			minVal = point.Value
		}
		if point.Value > maxVal {
			maxVal = point.Value
		}
	}

	var chart strings.Builder

	// Simple bar chart
	for _, point := range data {
		if len(data) > 7 {
			break // Limit to 7 bars for readability
		}

		// Calculate bar height
		barHeight := int((point.Value - minVal) / (maxVal - minVal) * 10)
		bar := strings.Repeat("█", barHeight)
		if len(bar) == 0 {
			bar = "▁"
		}

		chart.WriteString(fmt.Sprintf("%-8s %s %.1f\n", point.Label, bar, point.Value))
	}

	return chart.String()
}

func (m REPLModel) renderCorrelationsPanel(width, height int) string {
	style := m.getPanelStyle(width, height, false)

	content := "🔗 Metric Correlations\n\n"
	for metric, correlation := range m.analyticsData.Correlations {
		content += fmt.Sprintf("%-20s %.2f\n", metric, correlation)
	}

	return style.Render(content)
}

func (m REPLModel) renderOutliersPanel(width, height int) string {
	style := m.getPanelStyle(width, height, false)

	content := "⚠️ Performance Outliers\n\n"
	for _, outlier := range m.analyticsData.Outliers {
		severity := "🟡"
		if outlier.Severity == "high" {
			severity = "🔴"
		}
		content += fmt.Sprintf("%s %s\n", severity, outlier.Repository)
		content += fmt.Sprintf("   %s: %.1f\n\n", outlier.Metric, outlier.Value)
	}

	return style.Render(content)
}

// Helper methods
func (m REPLModel) renderHeader(title string) string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Background(lipgloss.Color("235")).
		Padding(0, 1).
		Width(m.width)

	return headerStyle.Render(fmt.Sprintf("LMMC TUI - %s", title))
}

func (m REPLModel) renderFooter(text string) string {
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Italic(true).
		Width(m.width)

	return footerStyle.Render(text)
}

func (m REPLModel) getPanelStyle(width, height int, active bool) lipgloss.Style {
	borderColor := lipgloss.Color("240")
	if active {
		borderColor = lipgloss.Color("39")
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(1).
		Width(width).
		Height(height)
}

func (m REPLModel) startHTTPServer() error {
	mux := http.NewServeMux()

	// Push notification endpoint
	mux.HandleFunc("/notify", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Handle push notification
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	m.server = &http.Server{
		Addr:              fmt.Sprintf(":%d", m.httpPort),
		Handler:           mux,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return m.server.ListenAndServe()
}

// StartREPL starts the interactive REPL with the given configuration
func StartREPL(mode REPLMode, httpPort int) error {
	model := NewREPLModel(mode, httpPort)

	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()

	return err
}
