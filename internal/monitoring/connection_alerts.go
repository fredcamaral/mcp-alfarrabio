// Package monitoring provides connection alerting for WebSocket monitoring
package monitoring

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

// ConnectionAlerting manages alerts for WebSocket connections
type ConnectionAlerting struct {
	mu              sync.RWMutex
	config          *AlertConfig
	rules           map[string]*AlertRule
	activeAlerts    map[string]*Alert
	alertHistory    []*Alert
	alertQueue      chan *Alert
	done            chan struct{}
	alertHandlers   []AlertHandler
	suppressions    map[string]time.Time
	escalationQueue chan *Alert
}

// AlertConfig configures connection alerting behavior
type AlertConfig struct {
	CheckInterval     time.Duration `json:"check_interval" yaml:"check_interval"`
	MaxAlertHistory   int           `json:"max_alert_history" yaml:"max_alert_history"`
	AlertCooldown     time.Duration `json:"alert_cooldown" yaml:"alert_cooldown"`
	EscalationTimeout time.Duration `json:"escalation_timeout" yaml:"escalation_timeout"`
	EnableEscalation  bool          `json:"enable_escalation" yaml:"enable_escalation"`
	MaxQueueSize      int           `json:"max_queue_size" yaml:"max_queue_size"`
	EnableSuppression bool          `json:"enable_suppression" yaml:"enable_suppression"`
	DefaultSeverity   Severity      `json:"default_severity" yaml:"default_severity"`
}

// AlertRule defines conditions for triggering alerts
type AlertRule struct {
	ID                   string                 `json:"id" yaml:"id"`
	Name                 string                 `json:"name" yaml:"name"`
	Description          string                 `json:"description" yaml:"description"`
	Metric               MetricType             `json:"metric" yaml:"metric"`
	Operator             Operator               `json:"operator" yaml:"operator"`
	Threshold            float64                `json:"threshold" yaml:"threshold"`
	Duration             time.Duration          `json:"duration" yaml:"duration"`
	Severity             Severity               `json:"severity" yaml:"severity"`
	Enabled              bool                   `json:"enabled" yaml:"enabled"`
	Conditions           []Condition            `json:"conditions" yaml:"conditions"`
	NotificationChannels []string               `json:"notification_channels" yaml:"notification_channels"`
	Metadata             map[string]interface{} `json:"metadata" yaml:"metadata"`
}

// Alert represents a triggered alert
type Alert struct {
	ID                string                 `json:"id"`
	RuleID            string                 `json:"rule_id"`
	RuleName          string                 `json:"rule_name"`
	ConnectionID      string                 `json:"connection_id,omitempty"`
	Severity          Severity               `json:"severity"`
	Status            AlertStatus            `json:"status"`
	Message           string                 `json:"message"`
	Description       string                 `json:"description"`
	Timestamp         time.Time              `json:"timestamp"`
	ResolvedAt        *time.Time             `json:"resolved_at,omitempty"`
	AcknowledgedAt    *time.Time             `json:"acknowledged_at,omitempty"`
	AcknowledgedBy    string                 `json:"acknowledged_by,omitempty"`
	EscalatedAt       *time.Time             `json:"escalated_at,omitempty"`
	CurrentValue      float64                `json:"current_value"`
	ThresholdValue    float64                `json:"threshold_value"`
	Metadata          map[string]interface{} `json:"metadata"`
	NotificationsSent []string               `json:"notifications_sent"`
}

// Condition represents an alert condition
type Condition struct {
	Metric   MetricType    `json:"metric" yaml:"metric"`
	Operator Operator      `json:"operator" yaml:"operator"`
	Value    float64       `json:"value" yaml:"value"`
	Duration time.Duration `json:"duration" yaml:"duration"`
}

// AlertHandler interface for handling alerts
type AlertHandler interface {
	HandleAlert(ctx context.Context, alert *Alert) error
	GetName() string
	IsEnabled() bool
}

// MetricType represents types of metrics to monitor
type MetricType string

const (
	MetricLatency         MetricType = "latency"
	MetricErrorRate       MetricType = "error_rate"
	MetricConnectionCount MetricType = "connection_count"
	MetricMessageRate     MetricType = "message_rate"
	MetricBandwidth       MetricType = "bandwidth"
	MetricHealthScore     MetricType = "health_score"
	MetricReconnections   MetricType = "reconnections"
	MetricUptime          MetricType = "uptime"
)

// Operator represents comparison operators
type Operator string

const (
	OperatorGreaterThan    Operator = "gt"
	OperatorLessThan       Operator = "lt"
	OperatorEquals         Operator = "eq"
	OperatorNotEquals      Operator = "ne"
	OperatorGreaterOrEqual Operator = "gte"
	OperatorLessOrEqual    Operator = "lte"
)

// Severity represents alert severity levels
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

// AlertStatus represents the status of an alert
type AlertStatus string

const (
	StatusTriggered    AlertStatus = "triggered"
	StatusAcknowledged AlertStatus = "acknowledged"
	StatusResolved     AlertStatus = "resolved"
	StatusEscalated    AlertStatus = "escalated"
	StatusSuppressed   AlertStatus = "suppressed"
)

// NewConnectionAlerting creates a new connection alerting system
func NewConnectionAlerting(config *AlertConfig) *ConnectionAlerting {
	if config == nil {
		config = DefaultAlertConfig()
	}

	ca := &ConnectionAlerting{
		config:          config,
		rules:           make(map[string]*AlertRule),
		activeAlerts:    make(map[string]*Alert),
		alertHistory:    make([]*Alert, 0, config.MaxAlertHistory),
		alertQueue:      make(chan *Alert, config.MaxQueueSize),
		done:            make(chan struct{}),
		alertHandlers:   make([]AlertHandler, 0),
		suppressions:    make(map[string]time.Time),
		escalationQueue: make(chan *Alert, config.MaxQueueSize),
	}

	// Start processing routines
	go ca.alertProcessor()
	if config.EnableEscalation {
		go ca.escalationProcessor()
	}

	// Add default alert rules
	ca.addDefaultRules()

	return ca
}

// DefaultAlertConfig returns default alert configuration
func DefaultAlertConfig() *AlertConfig {
	return &AlertConfig{
		CheckInterval:     30 * time.Second,
		MaxAlertHistory:   1000,
		AlertCooldown:     5 * time.Minute,
		EscalationTimeout: 15 * time.Minute,
		EnableEscalation:  true,
		MaxQueueSize:      1000,
		EnableSuppression: true,
		DefaultSeverity:   SeverityWarning,
	}
}

// addDefaultRules adds default alerting rules
func (ca *ConnectionAlerting) addDefaultRules() {
	defaultRules := []*AlertRule{
		{
			ID:          "high_latency",
			Name:        "High Latency",
			Description: "Alert when connection latency exceeds threshold",
			Metric:      MetricLatency,
			Operator:    OperatorGreaterThan,
			Threshold:   500, // 500ms
			Duration:    2 * time.Minute,
			Severity:    SeverityWarning,
			Enabled:     true,
			Metadata: map[string]interface{}{
				"default_rule": true,
			},
		},
		{
			ID:          "high_error_rate",
			Name:        "High Error Rate",
			Description: "Alert when error rate exceeds threshold",
			Metric:      MetricErrorRate,
			Operator:    OperatorGreaterThan,
			Threshold:   0.05, // 5%
			Duration:    5 * time.Minute,
			Severity:    SeverityError,
			Enabled:     true,
			Metadata: map[string]interface{}{
				"default_rule": true,
			},
		},
		{
			ID:          "low_health_score",
			Name:        "Low Health Score",
			Description: "Alert when connection health score is low",
			Metric:      MetricHealthScore,
			Operator:    OperatorLessThan,
			Threshold:   0.7,
			Duration:    3 * time.Minute,
			Severity:    SeverityWarning,
			Enabled:     true,
			Metadata: map[string]interface{}{
				"default_rule": true,
			},
		},
		{
			ID:          "frequent_reconnections",
			Name:        "Frequent Reconnections",
			Description: "Alert when reconnection rate is high",
			Metric:      MetricReconnections,
			Operator:    OperatorGreaterThan,
			Threshold:   3,
			Duration:    10 * time.Minute,
			Severity:    SeverityError,
			Enabled:     true,
			Metadata: map[string]interface{}{
				"default_rule": true,
			},
		},
		{
			ID:          "connection_count_critical",
			Name:        "Critical Connection Count",
			Description: "Alert when active connection count is critically low",
			Metric:      MetricConnectionCount,
			Operator:    OperatorLessThan,
			Threshold:   1,
			Duration:    time.Minute,
			Severity:    SeverityCritical,
			Enabled:     true,
			Metadata: map[string]interface{}{
				"default_rule": true,
				"system_wide":  true,
			},
		},
	}

	for _, rule := range defaultRules {
		ca.AddRule(rule)
	}
}

// AddRule adds an alert rule
func (ca *ConnectionAlerting) AddRule(rule *AlertRule) error {
	if rule.ID == "" {
		return errors.New("rule ID cannot be empty")
	}

	ca.mu.Lock()
	defer ca.mu.Unlock()

	ca.rules[rule.ID] = rule
	log.Printf("Added alert rule: %s (%s)", rule.ID, rule.Name)

	return nil
}

// RemoveRule removes an alert rule
func (ca *ConnectionAlerting) RemoveRule(ruleID string) error {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	if _, exists := ca.rules[ruleID]; !exists {
		return fmt.Errorf("rule %s not found", ruleID)
	}

	delete(ca.rules, ruleID)
	log.Printf("Removed alert rule: %s", ruleID)

	return nil
}

// AddHandler adds an alert handler
func (ca *ConnectionAlerting) AddHandler(handler AlertHandler) {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	ca.alertHandlers = append(ca.alertHandlers, handler)
	log.Printf("Added alert handler: %s", handler.GetName())
}

// EvaluateMetrics evaluates metrics against alert rules
func (ca *ConnectionAlerting) EvaluateMetrics(connectionID string, metrics map[MetricType]float64) {
	ca.mu.RLock()
	rules := make([]*AlertRule, 0, len(ca.rules))
	for _, rule := range ca.rules {
		if rule.Enabled {
			rules = append(rules, rule)
		}
	}
	ca.mu.RUnlock()

	for _, rule := range rules {
		value, exists := metrics[rule.Metric]
		if !exists {
			continue
		}

		if ca.evaluateCondition(rule, value) {
			ca.triggerAlert(rule, connectionID, value)
		} else {
			ca.resolveAlert(rule.ID, connectionID)
		}
	}
}

// evaluateCondition evaluates a single condition
func (ca *ConnectionAlerting) evaluateCondition(rule *AlertRule, value float64) bool {
	switch rule.Operator {
	case OperatorGreaterThan:
		return value > rule.Threshold
	case OperatorLessThan:
		return value < rule.Threshold
	case OperatorEquals:
		return value == rule.Threshold
	case OperatorNotEquals:
		return value != rule.Threshold
	case OperatorGreaterOrEqual:
		return value >= rule.Threshold
	case OperatorLessOrEqual:
		return value <= rule.Threshold
	default:
		return false
	}
}

// triggerAlert triggers an alert if conditions are met
func (ca *ConnectionAlerting) triggerAlert(rule *AlertRule, connectionID string, currentValue float64) {
	alertKey := fmt.Sprintf("%s:%s", rule.ID, connectionID)

	ca.mu.Lock()
	defer ca.mu.Unlock()

	// Check if alert is suppressed
	if ca.config.EnableSuppression {
		if suppressedUntil, exists := ca.suppressions[alertKey]; exists {
			if time.Now().Before(suppressedUntil) {
				return
			}
			delete(ca.suppressions, alertKey)
		}
	}

	// Check if alert already exists
	if existingAlert, exists := ca.activeAlerts[alertKey]; exists {
		// Update existing alert
		existingAlert.CurrentValue = currentValue
		existingAlert.Timestamp = time.Now()
		return
	}

	// Create new alert
	alert := &Alert{
		ID:             fmt.Sprintf("%s_%d", alertKey, time.Now().Unix()),
		RuleID:         rule.ID,
		RuleName:       rule.Name,
		ConnectionID:   connectionID,
		Severity:       rule.Severity,
		Status:         StatusTriggered,
		Message:        ca.generateAlertMessage(rule, connectionID, currentValue),
		Description:    rule.Description,
		Timestamp:      time.Now(),
		CurrentValue:   currentValue,
		ThresholdValue: rule.Threshold,
		Metadata:       make(map[string]interface{}),
	}

	// Copy rule metadata
	for k, v := range rule.Metadata {
		alert.Metadata[k] = v
	}

	ca.activeAlerts[alertKey] = alert
	ca.addToHistory(alert)

	// Queue alert for processing
	select {
	case ca.alertQueue <- alert:
	default:
		log.Printf("Alert queue full, dropping alert: %s", alert.ID)
	}

	log.Printf("Triggered alert: %s for connection %s", rule.Name, connectionID)
}

// resolveAlert resolves an alert
func (ca *ConnectionAlerting) resolveAlert(ruleID, connectionID string) {
	alertKey := fmt.Sprintf("%s:%s", ruleID, connectionID)

	ca.mu.Lock()
	defer ca.mu.Unlock()

	if alert, exists := ca.activeAlerts[alertKey]; exists {
		now := time.Now()
		alert.Status = StatusResolved
		alert.ResolvedAt = &now

		delete(ca.activeAlerts, alertKey)
		ca.addToHistory(alert)

		log.Printf("Resolved alert: %s for connection %s", alert.RuleName, connectionID)
	}
}

// generateAlertMessage generates an alert message
func (ca *ConnectionAlerting) generateAlertMessage(rule *AlertRule, connectionID string, value float64) string {
	switch rule.Metric {
	case MetricLatency:
		return fmt.Sprintf("High latency detected for connection %s: %.2fms (threshold: %.2fms)",
			connectionID, value, rule.Threshold)
	case MetricErrorRate:
		return fmt.Sprintf("High error rate detected for connection %s: %.2f%% (threshold: %.2f%%)",
			connectionID, value*100, rule.Threshold*100)
	case MetricHealthScore:
		return fmt.Sprintf("Low health score for connection %s: %.2f (threshold: %.2f)",
			connectionID, value, rule.Threshold)
	case MetricReconnections:
		return fmt.Sprintf("Frequent reconnections for connection %s: %.0f (threshold: %.0f)",
			connectionID, value, rule.Threshold)
	case MetricConnectionCount:
		return fmt.Sprintf("Low connection count: %.0f (threshold: %.0f)",
			value, rule.Threshold)
	default:
		return fmt.Sprintf("Alert triggered for %s: %.2f (threshold: %.2f)",
			rule.Metric, value, rule.Threshold)
	}
}

// alertProcessor processes alerts in the queue
func (ca *ConnectionAlerting) alertProcessor() {
	for {
		select {
		case alert := <-ca.alertQueue:
			ca.processAlert(alert)
		case <-ca.done:
			return
		}
	}
}

// processAlert processes a single alert
func (ca *ConnectionAlerting) processAlert(alert *Alert) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Send to all handlers
	for _, handler := range ca.alertHandlers {
		if handler.IsEnabled() {
			go func(h AlertHandler) {
				if err := h.HandleAlert(ctx, alert); err != nil {
					log.Printf("Alert handler %s failed: %v", h.GetName(), err)
				} else {
					ca.mu.Lock()
					alert.NotificationsSent = append(alert.NotificationsSent, h.GetName())
					ca.mu.Unlock()
				}
			}(handler)
		}
	}

	// Schedule escalation if enabled
	if ca.config.EnableEscalation && alert.Severity >= SeverityError {
		go ca.scheduleEscalation(alert)
	}
}

// scheduleEscalation schedules alert escalation
func (ca *ConnectionAlerting) scheduleEscalation(alert *Alert) {
	timer := time.NewTimer(ca.config.EscalationTimeout)
	defer timer.Stop()

	select {
	case <-timer.C:
		ca.escalateAlert(alert)
	case <-ca.done:
		return
	}
}

// escalateAlert escalates an alert
func (ca *ConnectionAlerting) escalateAlert(alert *Alert) {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	// Check if alert is still active
	alertKey := fmt.Sprintf("%s:%s", alert.RuleID, alert.ConnectionID)
	if activeAlert, exists := ca.activeAlerts[alertKey]; exists && activeAlert.Status == StatusTriggered {
		now := time.Now()
		activeAlert.Status = StatusEscalated
		activeAlert.EscalatedAt = &now

		// Increase severity if possible
		if activeAlert.Severity == SeverityWarning {
			activeAlert.Severity = SeverityError
		} else if activeAlert.Severity == SeverityError {
			activeAlert.Severity = SeverityCritical
		}

		// Re-queue for processing
		select {
		case ca.escalationQueue <- activeAlert:
		default:
			log.Printf("Escalation queue full, dropping escalated alert: %s", activeAlert.ID)
		}

		log.Printf("Escalated alert: %s", activeAlert.ID)
	}
}

// escalationProcessor processes escalated alerts
func (ca *ConnectionAlerting) escalationProcessor() {
	for {
		select {
		case alert := <-ca.escalationQueue:
			ca.processAlert(alert)
		case <-ca.done:
			return
		}
	}
}

// AcknowledgeAlert acknowledges an alert
func (ca *ConnectionAlerting) AcknowledgeAlert(alertID, acknowledgedBy string) error {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	for _, alert := range ca.activeAlerts {
		if alert.ID == alertID {
			now := time.Now()
			alert.Status = StatusAcknowledged
			alert.AcknowledgedAt = &now
			alert.AcknowledgedBy = acknowledgedBy

			log.Printf("Acknowledged alert: %s by %s", alertID, acknowledgedBy)
			return nil
		}
	}

	return fmt.Errorf("alert %s not found", alertID)
}

// SuppressAlert suppresses an alert for a duration
func (ca *ConnectionAlerting) SuppressAlert(ruleID, connectionID string, duration time.Duration) error {
	if !ca.config.EnableSuppression {
		return errors.New("alert suppression is disabled")
	}

	alertKey := fmt.Sprintf("%s:%s", ruleID, connectionID)

	ca.mu.Lock()
	defer ca.mu.Unlock()

	ca.suppressions[alertKey] = time.Now().Add(duration)
	log.Printf("Suppressed alert %s for %v", alertKey, duration)

	return nil
}

// addToHistory adds an alert to history
func (ca *ConnectionAlerting) addToHistory(alert *Alert) {
	ca.alertHistory = append(ca.alertHistory, alert)

	// Trim history if needed
	if len(ca.alertHistory) > ca.config.MaxAlertHistory {
		ca.alertHistory = ca.alertHistory[1:]
	}
}

// GetActiveAlerts returns currently active alerts
func (ca *ConnectionAlerting) GetActiveAlerts() []*Alert {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	alerts := make([]*Alert, 0, len(ca.activeAlerts))
	for _, alert := range ca.activeAlerts {
		alerts = append(alerts, alert)
	}

	return alerts
}

// GetAlertHistory returns alert history
func (ca *ConnectionAlerting) GetAlertHistory(limit int) []*Alert {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	if limit <= 0 || limit > len(ca.alertHistory) {
		limit = len(ca.alertHistory)
	}

	start := len(ca.alertHistory) - limit
	return ca.alertHistory[start:]
}

// GetRules returns all alert rules
func (ca *ConnectionAlerting) GetRules() map[string]*AlertRule {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	rules := make(map[string]*AlertRule)
	for id, rule := range ca.rules {
		rules[id] = rule
	}

	return rules
}

// GetAlertStats returns alert statistics
func (ca *ConnectionAlerting) GetAlertStats() map[string]interface{} {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	stats := map[string]interface{}{
		"active_alerts":    len(ca.activeAlerts),
		"total_rules":      len(ca.rules),
		"enabled_rules":    0,
		"alert_queue_size": len(ca.alertQueue),
		"history_size":     len(ca.alertHistory),
		"suppressions":     len(ca.suppressions),
	}

	enabledRules := 0
	for _, rule := range ca.rules {
		if rule.Enabled {
			enabledRules++
		}
	}
	stats["enabled_rules"] = enabledRules

	// Count alerts by severity
	severityCounts := map[Severity]int{
		SeverityInfo:     0,
		SeverityWarning:  0,
		SeverityError:    0,
		SeverityCritical: 0,
	}

	for _, alert := range ca.activeAlerts {
		severityCounts[alert.Severity]++
	}

	stats["alerts_by_severity"] = severityCounts

	return stats
}

// Close stops the connection alerting system
func (ca *ConnectionAlerting) Close() error {
	close(ca.done)
	return nil
}
