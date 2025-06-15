package tui

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Color palette - Modern and accessible
var (
	primaryColor   = lipgloss.Color("#00D9FF") // Cyan
	secondaryColor = lipgloss.Color("#FF006E") // Magenta
	successColor   = lipgloss.Color("#00F5FF") // Light cyan
	warningColor   = lipgloss.Color("#FFB700") // Orange
	errorColor     = lipgloss.Color("#FF006E") // Red
	mutedColor     = lipgloss.Color("#626262") // Gray
	bgColor        = lipgloss.Color("#1a1b26") // Dark background
	bgAltColor     = lipgloss.Color("#24283b") // Alternative background
	textColor      = lipgloss.Color("#c0caf5") // Light text
	borderColor    = lipgloss.Color("#414868") // Border
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

		// Enhanced F-key detection for better terminal compatibility
		case "f1", "F1", "ctrl+1", "alt+1":
			m.viewMode = ViewModeCommand
			return m, nil
		case "f2", "F2", "ctrl+2", "alt+2":
			m.viewMode = ViewModeDashboard
			m.loadDashboardData()
			return m, nil
		case "f3", "F3", "ctrl+3", "alt+3":
			m.viewMode = ViewModeAnalytics
			m.loadAnalyticsData()
			return m, nil
		case "f4", "F4", "ctrl+4", "alt+4":
			m.viewMode = ViewModeTaskList
			return m, nil
		case "f5", "F5", "ctrl+5", "alt+5":
			m.viewMode = ViewModePatterns
			return m, nil
		case "f6", "F6", "ctrl+6", "alt+6":
			m.viewMode = ViewModeInsights
			return m, nil

		// Additional help key
		case "ctrl+h", "?":
			if m.viewMode == ViewModeCommand {
				m.output = append(m.output, m.getHelpText()...)
			}
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
	m.output = append(m.output, "lmmc> "+cmd)

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
		m.output = append(m.output, "Unknown command: "+cmd)
		m.output = append(m.output, "Type 'help' for available commands")
	}

	// Clear input
	m.input = ""
	m.cursor = 0

	return m
}

func (m REPLModel) getHelpText() []string {
	// Style helpers
	headerStyle := lipgloss.NewStyle().Foreground(primaryColor).Bold(true)
	keyStyle := lipgloss.NewStyle().Foreground(secondaryColor).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(textColor)

	return []string{
		"",
		headerStyle.Render("LMMC TUI - Interactive Multi-Repository Intelligence Platform"),
		"",
		headerStyle.Render("View Navigation:"),
		"  " + keyStyle.Render("F1/Ctrl+1/Alt+1") + " " + descStyle.Render("Switch to command mode"),
		"  " + keyStyle.Render("F2/Ctrl+2/Alt+2") + " " + descStyle.Render("Switch to dashboard view"),
		"  " + keyStyle.Render("F3/Ctrl+3/Alt+3") + " " + descStyle.Render("Switch to analytics view"),
		"  " + keyStyle.Render("F4/Ctrl+4/Alt+4") + " " + descStyle.Render("Switch to task list view"),
		"  " + keyStyle.Render("F5/Ctrl+5/Alt+5") + " " + descStyle.Render("Switch to patterns view"),
		"  " + keyStyle.Render("F6/Ctrl+6/Alt+6") + " " + descStyle.Render("Switch to insights view"),
		"",
		headerStyle.Render("Dashboard Navigation:"),
		"  " + keyStyle.Render("Tab") + "             " + descStyle.Render("Switch between panes"),
		"  " + keyStyle.Render("h/j/k/l") + "         " + descStyle.Render("Vim-style navigation"),
		"  " + keyStyle.Render("r") + "               " + descStyle.Render("Refresh data"),
		"  " + keyStyle.Render("1-4") + "             " + descStyle.Render("Quick chart switching (analytics)"),
		"  " + keyStyle.Render("d/w/m") + "           " + descStyle.Render("Day/Week/Month time range"),
		"",
		headerStyle.Render("Basic Commands:"),
		"  " + keyStyle.Render("help") + "            " + descStyle.Render("Show this help message"),
		"  " + keyStyle.Render("status") + "          " + descStyle.Render("Show current status and configuration"),
		"  " + keyStyle.Render("clear") + "           " + descStyle.Render("Clear the output"),
		"  " + keyStyle.Render("exit, quit") + "      " + descStyle.Render("Exit the TUI"),
		"  " + keyStyle.Render("Ctrl+H, ?") + "       " + descStyle.Render("Quick help"),
		"  " + keyStyle.Render("Ctrl+C, Esc") + "     " + descStyle.Render("Exit the TUI"),
		"",
		headerStyle.Render("Document Generation:"),
		"  " + keyStyle.Render("prd create") + "      " + descStyle.Render("Start interactive PRD creation"),
		"  " + keyStyle.Render("trd create") + "      " + descStyle.Render("Generate TRD from existing PRD"),
		"  " + keyStyle.Render("workflow run") + "    " + descStyle.Render("Run complete automation workflow"),
		"",
		headerStyle.Render("Key Features:"),
		"  📊 Real-time multi-repo dashboard with live metrics",
		"  📈 Advanced analytics with gradient ASCII charts",
		"  🔄 Pattern detection and workflow analysis",
		"  💡 Cross-repository insights and recommendations",
		"  📋 Interactive task management with status tracking",
		"  🎨 Beautiful terminal UI with modern color scheme",
		"",
		headerStyle.Render("Tips:"),
		"  • F-keys not working? Try Ctrl+1-6 or Alt+1-6 instead",
		"  • Terminal colors look wrong? Try setting " + keyStyle.Render("COLORTERM=truecolor"),
		"  • For best experience, use a terminal with 256-color support",
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

	// Input line with enhanced styling
	promptStyle := lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true)

	inputStyle := lipgloss.NewStyle().
		Foreground(textColor)

	cursorStyle := lipgloss.NewStyle().
		Foreground(primaryColor).
		Background(primaryColor).
		Bold(true)

	prompt := promptStyle.Render("lmmc> ")

	// Build input with cursor
	var inputDisplay string
	if m.cursor < len(m.input) {
		inputDisplay = inputStyle.Render(m.input[:m.cursor]) +
			cursorStyle.Render(" ") +
			inputStyle.Render(m.input[m.cursor:])
	} else {
		inputDisplay = inputStyle.Render(m.input) + cursorStyle.Render(" ")
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

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true).
		MarginBottom(1)

	// Number styles
	numberStyle := lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true)

	labelStyle := lipgloss.NewStyle().
		Foreground(textColor)

	// Calculate completion rate
	stats := m.dashboardData.TaskStats
	completionRate := float64(stats.Completed) / float64(stats.Total)

	// Progress bar
	progressBar := m.renderProgressBar("Progress", completionRate, width-4)

	// Build content with styled elements
	var contentBuilder strings.Builder
	contentBuilder.WriteString(titleStyle.Render("📊 Task Statistics") + "\n\n")
	contentBuilder.WriteString(progressBar + "\n\n")

	// Stats grid
	contentBuilder.WriteString(labelStyle.Render("Total Tasks:    ") + numberStyle.Render(strconv.Itoa(stats.Total)) + "\n")
	contentBuilder.WriteString(labelStyle.Render("✅ Completed:   ") + numberStyle.Render(fmt.Sprintf("%d (%.1f%%)", stats.Completed, completionRate*100)) + "\n")
	contentBuilder.WriteString(labelStyle.Render("🔄 In Progress: ") + numberStyle.Render(strconv.Itoa(stats.InProgress)) + "\n")
	contentBuilder.WriteString(labelStyle.Render("⛔ Blocked:     ") + numberStyle.Render(strconv.Itoa(stats.Blocked)) + "\n\n")
	contentBuilder.WriteString(labelStyle.Render("📅 Today:       ") + numberStyle.Render(fmt.Sprintf("%d tasks", stats.TodayCount)) + "\n")
	contentBuilder.WriteString(labelStyle.Render("📈 This Week:   ") + numberStyle.Render(fmt.Sprintf("%d tasks", stats.WeekCount)))

	return style.Render(contentBuilder.String())
}

// Add the progress bar rendering function if not already present
func (m REPLModel) renderProgressBar(label string, value float64, width int) string {
	barWidth := width - len(label) - 10
	if barWidth < 10 {
		barWidth = 10
	}

	filledWidth := int(value * float64(barWidth))
	emptyWidth := barWidth - filledWidth

	// Choose color based on value
	var barColor lipgloss.Color
	if value < 0.3 {
		barColor = errorColor
	} else if value < 0.7 {
		barColor = warningColor
	} else {
		barColor = successColor
	}

	filled := lipgloss.NewStyle().
		Foreground(barColor).
		Render(strings.Repeat("█", filledWidth))

	empty := lipgloss.NewStyle().
		Foreground(mutedColor).
		Render(strings.Repeat("░", emptyWidth))

	percentage := lipgloss.NewStyle().
		Foreground(textColor).
		Bold(true).
		Render(fmt.Sprintf(" %3.0f%%", value*100))

	labelStyle := lipgloss.NewStyle().
		Width(len(label)).
		Foreground(textColor).
		Render(label)

	return fmt.Sprintf("%s %s%s%s", labelStyle, filled, empty, percentage)
}

func (m REPLModel) renderRepoMetricsPanel(width, height int) string {
	style := m.getPanelStyle(width, height, m.activePane == 1)

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true).
		MarginBottom(1)

	// Repository name style
	repoStyle := lipgloss.NewStyle().
		Foreground(secondaryColor).
		Bold(true)

	// Metric styles (removed unused variables)

	var contentBuilder strings.Builder
	contentBuilder.WriteString(titleStyle.Render("🏛️ Repository Metrics") + "\n\n")

	// Render each repository with mini progress bars
	for i, repo := range m.repositories {
		if i >= 2 { // Limit to 2 repos for space
			break
		}

		metrics := m.dashboardData.RepoMetrics[repo]

		// Repository name
		contentBuilder.WriteString(repoStyle.Render("📁 "+repo) + "\n")

		// Mini progress bars for each metric
		prodBar := m.renderMiniBar("Prod", metrics.Productivity/100, width-8)
		contentBuilder.WriteString("  " + prodBar + "\n")

		velocityNorm := metrics.Velocity / 20.0 // Normalize to 0-1 (assuming 20 is high)
		if velocityNorm > 1.0 {
			velocityNorm = 1.0
		}
		velBar := m.renderMiniBar("Vel ", velocityNorm, width-8)
		contentBuilder.WriteString("  " + velBar + "\n")

		compBar := m.renderMiniBar("Comp", metrics.CompletionRate, width-8)
		contentBuilder.WriteString("  " + compBar + "\n")

		if i < len(m.repositories)-1 && i < 1 {
			contentBuilder.WriteString("\n")
		}
	}

	return style.Render(contentBuilder.String())
}

// Mini progress bar for compact display
func (m REPLModel) renderMiniBar(label string, value float64, width int) string {
	barWidth := width - len(label) - 8
	if barWidth < 5 {
		barWidth = 5
	}

	filledWidth := int(value * float64(barWidth))
	emptyWidth := barWidth - filledWidth

	// Compact bar characters
	filled := lipgloss.NewStyle().
		Foreground(primaryColor).
		Render(strings.Repeat("▰", filledWidth))

	empty := lipgloss.NewStyle().
		Foreground(mutedColor).
		Render(strings.Repeat("▱", emptyWidth))

	percentage := lipgloss.NewStyle().
		Foreground(textColor).
		Render(fmt.Sprintf("%3.0f%%", value*100))

	return fmt.Sprintf("%s %s%s %s", label, filled, empty, percentage)
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
		return lipgloss.NewStyle().Foreground(mutedColor).Render("No data available")
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

	// Enhanced bar chart with gradient colors
	for i, point := range data {
		if i >= 7 { // Limit to 7 bars
			break
		}

		// Calculate bar height (0-10 scale)
		normalized := (point.Value - minVal) / (maxVal - minVal)
		barHeight := int(normalized * 10)

		// Build gradient bar
		var bar string
		for j := 0; j < barHeight; j++ {
			if j < 3 {
				bar += lipgloss.NewStyle().Foreground(errorColor).Render("▂")
			} else if j < 7 {
				bar += lipgloss.NewStyle().Foreground(warningColor).Render("▄")
			} else {
				bar += lipgloss.NewStyle().Foreground(successColor).Render("█")
			}
		}

		if barHeight == 0 {
			bar = lipgloss.NewStyle().Foreground(mutedColor).Render("▁")
		}

		// Format label and value
		label := lipgloss.NewStyle().
			Width(8).
			Foreground(textColor).
			Render(point.Label)

		value := lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			Render(fmt.Sprintf("%.1f", point.Value))

		// Add percentage indicator
		percentage := normalized * 100
		var indicator string
		if percentage >= 80 {
			indicator = lipgloss.NewStyle().Foreground(successColor).Render("↑")
		} else if percentage >= 50 {
			indicator = lipgloss.NewStyle().Foreground(warningColor).Render("→")
		} else {
			indicator = lipgloss.NewStyle().Foreground(errorColor).Render("↓")
		}

		chart.WriteString(fmt.Sprintf("%s %s %s %s\n", label, bar, value, indicator))
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
	width := m.width
	if width < 80 {
		width = 80
	}

	// Title style with gradient effect
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Background(bgAltColor).
		Padding(0, 2).
		Width(width)

	// Mode indicators
	modes := []struct {
		key      string
		label    string
		viewMode ViewMode
		icon     string
	}{
		{"F1", "Command", ViewModeCommand, "💻"},
		{"F2", "Dashboard", ViewModeDashboard, "📊"},
		{"F3", "Analytics", ViewModeAnalytics, "📈"},
		{"F4", "Tasks", ViewModeTaskList, "📋"},
		{"F5", "Patterns", ViewModePatterns, "🔄"},
		{"F6", "Insights", ViewModeInsights, "💡"},
	}

	// Build mode buttons
	var modeButtons []string
	for _, mode := range modes {
		style := lipgloss.NewStyle().
			Padding(0, 1).
			MarginRight(1)

		if m.viewMode == mode.viewMode {
			style = style.
				Foreground(bgColor).
				Background(primaryColor).
				Bold(true)
		} else {
			style = style.
				Foreground(mutedColor).
				Background(bgColor)
		}

		button := style.Render(fmt.Sprintf("%s %s:%s", mode.icon, mode.key, mode.label))
		modeButtons = append(modeButtons, button)
	}

	// Title section
	titleSection := titleStyle.Render("🚀 LMMC TUI - " + title)

	// Mode bar
	modeBar := lipgloss.JoinHorizontal(lipgloss.Left, modeButtons...)
	modeSection := lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		Background(bgColor).
		Padding(0, 1).
		Render(modeBar)

	return titleSection + "\n" + modeSection
}

func (m REPLModel) renderFooter(text string) string {
	width := m.width
	if width < 80 {
		width = 80
	}

	// Status indicators
	var status []string

	// Connection status
	statusStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Bold(true)

	if m.httpPort > 0 {
		status = append(status, statusStyle.Copy().
			Foreground(bgColor).
			Background(successColor).
			Render("● HTTP:"+strconv.Itoa(m.httpPort)))
	} else {
		status = append(status, statusStyle.Copy().
			Foreground(mutedColor).
			Render("○ Offline"))
	}

	// Time
	timeStr := time.Now().Format("15:04:05")
	status = append(status, lipgloss.NewStyle().
		Foreground(mutedColor).
		Render(timeStr))

	// Help hints based on mode
	var helpHints []string
	switch m.viewMode {
	case ViewModeCommand:
		helpHints = []string{"Enter: Execute", "Ctrl+H: Help", "Ctrl+C: Exit"}
	case ViewModeDashboard, ViewModeAnalytics:
		helpHints = []string{"Tab: Switch Pane", "R: Refresh", "Ctrl+C: Exit"}
	default:
		helpHints = []string{"↑↓: Navigate", "Enter: Select", "Ctrl+C: Exit"}
	}

	helpText := lipgloss.NewStyle().
		Foreground(mutedColor).
		Render(strings.Join(helpHints, " | "))

	// Combine elements
	leftContent := lipgloss.JoinHorizontal(lipgloss.Left, status...)

	// Calculate padding
	leftWidth := lipgloss.Width(leftContent)
	rightWidth := lipgloss.Width(helpText)
	padding := width - leftWidth - rightWidth - 4
	if padding < 0 {
		padding = 0
	}

	footer := lipgloss.JoinHorizontal(
		lipgloss.Left,
		leftContent,
		strings.Repeat(" ", padding),
		helpText,
	)

	return lipgloss.NewStyle().
		Foreground(textColor).
		Background(bgColor).
		Padding(0, 1).
		Width(width).
		Render(footer)
}

func (m REPLModel) getPanelStyle(width, height int, active bool) lipgloss.Style {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1).
		Width(width).
		Height(height).
		Background(bgColor)

	if active {
		style = style.
			BorderForeground(primaryColor).
			BorderBackground(bgColor)
	} else {
		style = style.
			BorderForeground(borderColor).
			BorderBackground(bgColor)
	}

	return style
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
