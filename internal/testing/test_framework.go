// Package testing provides comprehensive testing framework for the MCP Memory Server
package testing

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/stretchr/testify/suite"
)

// TestSuite provides base functionality for all test suites
type TestSuite struct {
	suite.Suite
	config      *TestConfig
	db          *sql.DB
	cleanup     []func()
	mutex       sync.Mutex
	testData    *TestDataManager
	mocks       *MockManager
	fixtures    *FixtureManager
	performance *PerformanceTracker
}

// TestConfig defines test configuration
type TestConfig struct {
	// Database settings
	DatabaseURL     string `json:"database_url"`
	TestDatabaseURL string `json:"test_database_url"`
	UseInMemoryDB   bool   `json:"use_in_memory_db"`
	MigrateOnSetup  bool   `json:"migrate_on_setup"`

	// Performance settings
	EnablePerformanceTracking bool                     `json:"enable_performance_tracking"`
	PerformanceThresholds     map[string]time.Duration `json:"performance_thresholds"`

	// Concurrency settings
	MaxConcurrentTests int           `json:"max_concurrent_tests"`
	TestTimeout        time.Duration `json:"test_timeout"`

	// Data settings
	SeedTestData      bool   `json:"seed_test_data"`
	CleanupAfterTests bool   `json:"cleanup_after_tests"`
	TestDataDirectory string `json:"test_data_directory"`

	// Coverage settings
	EnableCoverage    bool    `json:"enable_coverage"`
	CoverageThreshold float64 `json:"coverage_threshold"`

	// Integration settings
	EnableIntegrationTests bool              `json:"enable_integration_tests"`
	ExternalServiceURLs    map[string]string `json:"external_service_urls"`
}

// DefaultTestConfig returns optimized test configuration
func DefaultTestConfig() *TestConfig {
	return &TestConfig{
		DatabaseURL:               "postgres://localhost/mcp_memory_test",
		TestDatabaseURL:           "postgres://localhost/mcp_memory_test",
		UseInMemoryDB:             true,
		MigrateOnSetup:            true,
		EnablePerformanceTracking: true,
		PerformanceThresholds: map[string]time.Duration{
			"unit_test":        100 * time.Millisecond,
			"integration_test": 5 * time.Second,
			"load_test":        30 * time.Second,
		},
		MaxConcurrentTests:     10,
		TestTimeout:            30 * time.Second,
		SeedTestData:           true,
		CleanupAfterTests:      true,
		TestDataDirectory:      "./testdata",
		EnableCoverage:         true,
		CoverageThreshold:      80.0,
		EnableIntegrationTests: true,
		ExternalServiceURLs: map[string]string{
			"qdrant": "http://localhost:6333",
			"openai": "https://api.openai.com",
		},
	}
}

// TestDataManager manages test data and fixtures
type TestDataManager struct {
	config    *TestConfig
	fixtures  map[string]interface{}
	templates map[string]string
	mutex     sync.RWMutex
}

// MockManager manages mocks and stubs for testing
type MockManager struct {
	mocks map[string]interface{}
	mutex sync.RWMutex
}

// FixtureManager manages test fixtures and sample data
type FixtureManager struct {
	fixtures map[string]*Fixture
	loader   *FixtureLoader
	mutex    sync.RWMutex
}

// PerformanceTracker tracks test performance metrics
type PerformanceTracker struct {
	metrics   map[string]*TestMetrics
	threshold map[string]time.Duration
	mutex     sync.RWMutex
}

// TestMetrics represents performance metrics for tests
type TestMetrics struct {
	TestName     string        `json:"test_name"`
	Duration     time.Duration `json:"duration"`
	MemoryUsage  int64         `json:"memory_usage"`
	Allocations  int64         `json:"allocations"`
	StartTime    time.Time     `json:"start_time"`
	EndTime      time.Time     `json:"end_time"`
	Success      bool          `json:"success"`
	ErrorMessage string        `json:"error_message,omitempty"`
}

// Fixture represents a test data fixture
type Fixture struct {
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	Data         map[string]interface{} `json:"data"`
	Dependencies []string               `json:"dependencies"`
	CreatedAt    time.Time              `json:"created_at"`
}

// FixtureLoader loads test fixtures from various sources
type FixtureLoader struct {
	sources []FixtureSource
}

// FixtureSource defines interface for loading fixtures
type FixtureSource interface {
	Load(name string) (*Fixture, error)
	List() ([]string, error)
	Type() string
}

// NewTestSuite creates a new comprehensive test suite
func NewTestSuite(config *TestConfig) *TestSuite {
	if config == nil {
		config = DefaultTestConfig()
	}

	testSuite := &TestSuite{
		config:      config,
		cleanup:     make([]func(), 0),
		testData:    NewTestDataManager(config),
		mocks:       NewMockManager(),
		fixtures:    NewFixtureManager(config),
		performance: NewPerformanceTracker(config.PerformanceThresholds),
	}

	return testSuite
}

// SetupSuite initializes the test suite
func (ts *TestSuite) SetupSuite() {
	// Initialize database if needed
	if !ts.config.UseInMemoryDB {
		ts.setupDatabase()
	}

	// Seed test data if enabled
	if ts.config.SeedTestData {
		ts.seedTestData()
	}

	// Initialize fixtures
	_ = ts.fixtures.LoadAll()

	// Setup performance tracking
	if ts.config.EnablePerformanceTracking {
		ts.performance.Start()
	}
}

// TearDownSuite cleans up after all tests
func (ts *TestSuite) TearDownSuite() {
	// Run cleanup functions
	ts.mutex.Lock()
	for _, cleanup := range ts.cleanup {
		cleanup()
	}
	ts.cleanup = nil
	ts.mutex.Unlock()

	// Cleanup database
	if ts.db != nil {
		_ = ts.db.Close()
	}

	// Stop performance tracking
	if ts.config.EnablePerformanceTracking {
		ts.performance.Stop()
		ts.performance.Report()
	}

	// Cleanup test data
	if ts.config.CleanupAfterTests {
		ts.cleanupTestData()
	}
}

// SetupTest runs before each test
func (ts *TestSuite) SetupTest() {
	testName := ts.T().Name()

	// Start performance tracking
	if ts.config.EnablePerformanceTracking {
		ts.performance.StartTest(testName)
	}

	// Reset mocks
	ts.mocks.Reset()
}

// TearDownTest runs after each test
func (ts *TestSuite) TearDownTest() {
	testName := ts.T().Name()

	// Stop performance tracking
	if ts.config.EnablePerformanceTracking {
		success := !ts.T().Failed()
		ts.performance.EndTest(testName, success)
	}
}

// AddCleanup adds a cleanup function to run after tests
func (ts *TestSuite) AddCleanup(fn func()) {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()

	ts.cleanup = append(ts.cleanup, fn)
}

// GetTestData returns test data manager
func (ts *TestSuite) GetTestData() *TestDataManager {
	return ts.testData
}

// GetMocks returns mock manager
func (ts *TestSuite) GetMocks() *MockManager {
	return ts.mocks
}

// GetFixtures returns fixture manager
func (ts *TestSuite) GetFixtures() *FixtureManager {
	return ts.fixtures
}

// AssertPerformance asserts test performance meets thresholds
func (ts *TestSuite) AssertPerformance(testName string) {
	if !ts.config.EnablePerformanceTracking {
		return
	}

	metrics := ts.performance.GetMetrics(testName)
	if metrics == nil {
		return
	}

	if threshold, exists := ts.config.PerformanceThresholds[getTestType(testName)]; exists {
		ts.Assert().True(
			metrics.Duration <= threshold,
			fmt.Sprintf("Test %s took %v, exceeding threshold of %v",
				testName, metrics.Duration, threshold),
		)
	}
}

// RequireEnvironment skips test if environment requirements not met
func (ts *TestSuite) RequireEnvironment(requirements ...string) {
	for _, req := range requirements {
		switch req {
		case "integration":
			if !ts.config.EnableIntegrationTests {
				ts.T().Skip("Integration tests disabled")
			}
		case "database":
			if ts.config.UseInMemoryDB {
				ts.T().Skip("Database tests require real database")
			}
		case "external_services":
			for service, url := range ts.config.ExternalServiceURLs {
				if !ts.isServiceAvailable(service, url) {
					ts.T().Skipf("External service %s not available at %s", service, url)
				}
			}
		}
	}
}

// CreateTempDir creates a temporary directory for test files
func (ts *TestSuite) CreateTempDir(pattern string) string {
	dir, err := os.MkdirTemp("", pattern)
	ts.Require().NoError(err)

	ts.AddCleanup(func() {
		_ = os.RemoveAll(dir)
	})

	return dir
}

// Private methods

func (ts *TestSuite) setupDatabase() {
	if ts.config.TestDatabaseURL == "" {
		ts.T().Fatal("Test database URL not configured")
	}

	db, err := sql.Open("postgres", ts.config.TestDatabaseURL)
	ts.Require().NoError(err)

	ts.db = db

	if ts.config.MigrateOnSetup {
		ts.runMigrations()
	}
}

func (ts *TestSuite) runMigrations() {
	// In production, would run actual database migrations
	// For now, this is a placeholder
}

func (ts *TestSuite) seedTestData() {
	ts.testData.SeedAll()
}

func (ts *TestSuite) cleanupTestData() {
	ts.testData.CleanupAll()
}

func (ts *TestSuite) isServiceAvailable(service, url string) bool {
	// Simple availability check - in production would be more sophisticated
	return true
}

// TestDataManager implementation

// NewTestDataManager creates a new test data manager
func NewTestDataManager(config *TestConfig) *TestDataManager {
	return &TestDataManager{
		config:    config,
		fixtures:  make(map[string]interface{}),
		templates: make(map[string]string),
	}
}

// SeedAll seeds all test data
func (tdm *TestDataManager) SeedAll() {
	// Implement test data seeding
}

// CleanupAll cleans up all test data
func (tdm *TestDataManager) CleanupAll() {
	tdm.mutex.Lock()
	defer tdm.mutex.Unlock()

	tdm.fixtures = make(map[string]interface{})
	tdm.templates = make(map[string]string)
}

// CreateTestData creates test data of specified type
func (tdm *TestDataManager) CreateTestData(dataType string, count int) []interface{} {
	switch dataType {
	case "content":
		return tdm.createTestContent(count)
	case "users":
		return tdm.createTestUsers(count)
	case "projects":
		return tdm.createTestProjects(count)
	default:
		return nil
	}
}

func (tdm *TestDataManager) createTestContent(count int) []interface{} {
	content := make([]interface{}, count)
	for i := 0; i < count; i++ {
		content[i] = map[string]interface{}{
			"id":         fmt.Sprintf("content_%d", i),
			"title":      fmt.Sprintf("Test Content %d", i),
			"content":    fmt.Sprintf("This is test content number %d", i),
			"project_id": "test_project",
			"created_at": time.Now().Add(-time.Duration(i) * time.Hour),
		}
	}
	return content
}

func (tdm *TestDataManager) createTestUsers(count int) []interface{} {
	users := make([]interface{}, count)
	for i := 0; i < count; i++ {
		users[i] = map[string]interface{}{
			"id":     fmt.Sprintf("user_%d", i),
			"name":   fmt.Sprintf("Test User %d", i),
			"email":  fmt.Sprintf("user%d@test.com", i),
			"role":   "user",
			"active": true,
		}
	}
	return users
}

func (tdm *TestDataManager) createTestProjects(count int) []interface{} {
	projects := make([]interface{}, count)
	for i := 0; i < count; i++ {
		projects[i] = map[string]interface{}{
			"id":          fmt.Sprintf("project_%d", i),
			"name":        fmt.Sprintf("Test Project %d", i),
			"description": fmt.Sprintf("This is test project number %d", i),
			"owner_id":    "user_0",
			"created_at":  time.Now().Add(-time.Duration(i) * 24 * time.Hour),
		}
	}
	return projects
}

// MockManager implementation

// NewMockManager creates a new mock manager
func NewMockManager() *MockManager {
	return &MockManager{
		mocks: make(map[string]interface{}),
	}
}

// Register registers a mock object
func (mm *MockManager) Register(name string, mock interface{}) {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	mm.mocks[name] = mock
}

// Get retrieves a mock object
func (mm *MockManager) Get(name string) interface{} {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	return mm.mocks[name]
}

// Reset resets all mocks
func (mm *MockManager) Reset() {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	// Reset mock states - in production would call Reset() on each mock
	for _, mock := range mm.mocks {
		if resettable, ok := mock.(interface{ Reset() }); ok {
			resettable.Reset()
		}
	}
}

// FixtureManager implementation

// NewFixtureManager creates a new fixture manager
func NewFixtureManager(config *TestConfig) *FixtureManager {
	return &FixtureManager{
		fixtures: make(map[string]*Fixture),
		loader:   NewFixtureLoader(config.TestDataDirectory),
	}
}

// LoadAll loads all available fixtures
func (fm *FixtureManager) LoadAll() error {
	fixtures, err := fm.loader.LoadAll()
	if err != nil {
		return err
	}

	fm.mutex.Lock()
	defer fm.mutex.Unlock()

	for _, fixture := range fixtures {
		fm.fixtures[fixture.Name] = fixture
	}

	return nil
}

// Get retrieves a fixture by name
func (fm *FixtureManager) Get(name string) *Fixture {
	fm.mutex.RLock()
	defer fm.mutex.RUnlock()

	return fm.fixtures[name]
}

// NewFixtureLoader creates a new fixture loader
func NewFixtureLoader(dataDir string) *FixtureLoader {
	return &FixtureLoader{
		sources: []FixtureSource{
			&JSONFixtureSource{directory: dataDir},
			&YAMLFixtureSource{directory: dataDir},
		},
	}
}

// LoadAll loads all fixtures from all sources
func (fl *FixtureLoader) LoadAll() ([]*Fixture, error) {
	var allFixtures []*Fixture

	for _, source := range fl.sources {
		names, err := source.List()
		if err != nil {
			continue // Skip unavailable sources
		}

		for _, name := range names {
			fixture, err := source.Load(name)
			if err != nil {
				continue // Skip failed loads
			}
			allFixtures = append(allFixtures, fixture)
		}
	}

	return allFixtures, nil
}

// JSONFixtureSource loads fixtures from JSON files
type JSONFixtureSource struct {
	directory string
}

func (jfs *JSONFixtureSource) Load(name string) (*Fixture, error) {
	// Simplified fixture loading - in production would parse JSON
	return &Fixture{
		Name:      name,
		Type:      "json",
		Data:      make(map[string]interface{}),
		CreatedAt: time.Now(),
	}, nil
}

func (jfs *JSONFixtureSource) List() ([]string, error) {
	if jfs.directory == "" {
		return nil, errors.New("directory not set")
	}

	files, err := filepath.Glob(filepath.Join(jfs.directory, "*.json"))
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(files))
	for _, file := range files {
		base := filepath.Base(file)
		name := base[:len(base)-5] // Remove .json extension
		names = append(names, name)
	}

	return names, nil
}

func (jfs *JSONFixtureSource) Type() string {
	return "json"
}

// YAMLFixtureSource loads fixtures from YAML files
type YAMLFixtureSource struct {
	directory string
}

func (yfs *YAMLFixtureSource) Load(name string) (*Fixture, error) {
	// Simplified fixture loading - in production would parse YAML
	return &Fixture{
		Name:      name,
		Type:      "yaml",
		Data:      make(map[string]interface{}),
		CreatedAt: time.Now(),
	}, nil
}

func (yfs *YAMLFixtureSource) List() ([]string, error) {
	if yfs.directory == "" {
		return nil, errors.New("directory not set")
	}

	files, err := filepath.Glob(filepath.Join(yfs.directory, "*.yaml"))
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(files))
	for _, file := range files {
		base := filepath.Base(file)
		name := base[:len(base)-5] // Remove .yaml extension
		names = append(names, name)
	}

	return names, nil
}

func (yfs *YAMLFixtureSource) Type() string {
	return "yaml"
}

// PerformanceTracker implementation

// NewPerformanceTracker creates a new performance tracker
func NewPerformanceTracker(thresholds map[string]time.Duration) *PerformanceTracker {
	return &PerformanceTracker{
		metrics:   make(map[string]*TestMetrics),
		threshold: thresholds,
	}
}

// Start starts performance tracking
func (pt *PerformanceTracker) Start() {
	// Initialize performance tracking
}

// Stop stops performance tracking
func (pt *PerformanceTracker) Stop() {
	// Finalize performance tracking
}

// StartTest starts tracking for a specific test
func (pt *PerformanceTracker) StartTest(testName string) {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Safe conversion with overflow check
	var memUsage, allocations int64
	if m.Alloc > math.MaxInt64 {
		memUsage = math.MaxInt64
	} else {
		memUsage = int64(m.Alloc)
	}
	if m.Mallocs > math.MaxInt64 {
		allocations = math.MaxInt64
	} else {
		allocations = int64(m.Mallocs)
	}

	pt.metrics[testName] = &TestMetrics{
		TestName:    testName,
		StartTime:   time.Now(),
		MemoryUsage: memUsage,
		Allocations: allocations,
	}
}

// EndTest ends tracking for a specific test
func (pt *PerformanceTracker) EndTest(testName string, success bool) {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	metric, exists := pt.metrics[testName]
	if !exists {
		return
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	metric.EndTime = time.Now()
	metric.Duration = metric.EndTime.Sub(metric.StartTime)

	// Safe conversion with overflow check
	var currentMem, currentAllocs int64
	if m.Alloc > math.MaxInt64 {
		currentMem = math.MaxInt64
	} else {
		currentMem = int64(m.Alloc)
	}
	if m.Mallocs > math.MaxInt64 {
		currentAllocs = math.MaxInt64
	} else {
		currentAllocs = int64(m.Mallocs)
	}

	metric.MemoryUsage = currentMem - metric.MemoryUsage
	metric.Allocations = currentAllocs - metric.Allocations
	metric.Success = success
}

// GetMetrics returns metrics for a specific test
func (pt *PerformanceTracker) GetMetrics(testName string) *TestMetrics {
	pt.mutex.RLock()
	defer pt.mutex.RUnlock()

	return pt.metrics[testName]
}

// Report generates a performance report
func (pt *PerformanceTracker) Report() {
	pt.mutex.RLock()
	defer pt.mutex.RUnlock()

	fmt.Println("\n=== Performance Report ===")
	for testName, metrics := range pt.metrics {
		status := "PASS"
		if !metrics.Success {
			status = "FAIL"
		}

		fmt.Printf("%s: %s (%v, %d allocs, %d bytes)\n",
			testName, status, metrics.Duration, metrics.Allocations, metrics.MemoryUsage)
	}
	fmt.Println("=========================")
}

// Utility functions

func getTestType(testName string) string {
	if contains(testName, "Integration") {
		return "integration_test"
	}
	if contains(testName, "Load") || contains(testName, "Benchmark") {
		return "load_test"
	}
	return "unit_test"
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s[:len(substr)] == substr ||
		s[len(s)-len(substr):] == substr ||
		findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
