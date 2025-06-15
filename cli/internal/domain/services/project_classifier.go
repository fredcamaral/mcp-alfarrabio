package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"lerian-mcp-memory-cli/internal/domain/constants"
	"lerian-mcp-memory-cli/internal/domain/entities"
)

// ProjectClassifier interface defines project classification capabilities
// ProjectClassifier interface is now defined in interfaces.go

// FileAnalyzer interface for analyzing file systems
type FileAnalyzer interface {
	AnalyzeDirectory(ctx context.Context, path string) (*DirectoryAnalysis, error)
	GetFilesByPattern(ctx context.Context, path string, patterns []string) (map[string][]string, error)
	ReadConfigFiles(ctx context.Context, path string) (map[string]interface{}, error)
}

// ProjectStructure represents the analyzed structure of a project
type ProjectStructure struct {
	RootPath       string                 `json:"root_path"`
	Directories    []string               `json:"directories"`
	Files          map[string][]string    `json:"files"`        // Extension -> file list
	ConfigFiles    map[string]interface{} `json:"config_files"` // File -> parsed content
	Dependencies   map[string][]string    `json:"dependencies"` // Package manager -> deps
	BuildSystems   []string               `json:"build_systems"`
	TestFrameworks []string               `json:"test_frameworks"`
	CIFiles        []string               `json:"ci_files"`
	DockerFiles    []string               `json:"docker_files"`
	DatabaseFiles  []string               `json:"database_files"`
	AnalyzedAt     time.Time              `json:"analyzed_at"`
}

// DirectoryAnalysis represents analysis of a directory
type DirectoryAnalysis struct {
	TotalFiles     int                    `json:"total_files"`
	FilesByExt     map[string]int         `json:"files_by_ext"`
	DirectoryDepth int                    `json:"directory_depth"`
	LargestFiles   []string               `json:"largest_files"`
	ConfigFiles    []string               `json:"config_files"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// ProjectClassifierConfig holds configuration for project classification
type ProjectClassifierConfig struct {
	MaxAnalysisDepth   int                 `json:"max_analysis_depth"`
	IgnorePatterns     []string            `json:"ignore_patterns"`
	ConfigFilePatterns []string            `json:"config_file_patterns"`
	MaxFilesToAnalyze  int                 `json:"max_files_to_analyze"`
	FrameworkDetectors map[string][]string `json:"framework_detectors"`
	LanguageExtensions map[string][]string `json:"language_extensions"`
}

// DefaultProjectClassifierConfig returns default configuration
func DefaultProjectClassifierConfig() *ProjectClassifierConfig {
	return &ProjectClassifierConfig{
		MaxAnalysisDepth:  10,
		MaxFilesToAnalyze: 1000,
		IgnorePatterns: []string{
			"node_modules", ".git", ".vscode", ".idea", "target", "dist", "build",
			"vendor", "__pycache__", ".pytest_cache", "coverage", ".coverage",
		},
		ConfigFilePatterns: []string{
			"package.json", "composer.json", "Gemfile", "requirements.txt", "Pipfile",
			"go.mod", "Cargo.toml", "pom.xml", "build.gradle", "CMakeLists.txt",
			"Dockerfile", "docker-compose.yml", ".github", ".gitlab-ci.yml",
			"tsconfig.json", "webpack.config.js", "next.config.js", "nuxt.config.js",
		},
		FrameworkDetectors: map[string][]string{
			"react":     {"react", "@types/react", "react-dom"},
			"vue":       {"vue", "@vue/cli", "nuxt"},
			"angular":   {"@angular/core", "@angular/cli"},
			"svelte":    {"svelte", "@sveltejs/kit"},
			"next":      {"next", "next.js"},
			"nuxt":      {"nuxt", "@nuxt/"},
			"express":   {"express", "express.js"},
			"fastapi":   {"fastapi", "uvicorn"},
			"django":    {"django", "Django"},
			"flask":     {"flask", "Flask"},
			"gin":       {"github.com/gin-gonic/gin"},
			"echo":      {"github.com/labstack/echo"},
			"spring":    {"spring-boot", "springframework"},
			"rails":     {"rails", "ruby on rails"},
			"laravel":   {"laravel/framework"},
			"cobra":     {"github.com/spf13/cobra"},
			"cli":       {"github.com/urfave/cli"},
			"commander": {"commander"},
		},
		LanguageExtensions: map[string][]string{
			"go":         {".go"},
			"javascript": {".js", ".mjs", ".cjs"},
			"typescript": {".ts", ".tsx"},
			"python":     {".py", ".pyx", ".pyi"},
			"java":       {".java"},
			"csharp":     {".cs"},
			"cpp":        {".cpp", ".cxx", ".cc", ".c++"},
			"c":          {".c", ".h"},
			"rust":       {".rs"},
			"php":        {".php"},
			"ruby":       {".rb"},
			"kotlin":     {".kt", ".kts"},
			"swift":      {".swift"},
			"dart":       {".dart"},
			"scala":      {".scala"},
			"clojure":    {".clj", ".cljs"},
			"elixir":     {".ex", ".exs"},
			"erlang":     {".erl"},
			"haskell":    {".hs"},
			"ocaml":      {".ml", ".mli"},
			"fsharp":     {".fs", ".fsx"},
			"lua":        {".lua"},
			"r":          {".r", ".R"},
			"matlab":     {".m"},
			"julia":      {".jl"},
		},
	}
}

// projectClassifierImpl implements the ProjectClassifier interface
type projectClassifierImpl struct {
	fileAnalyzer FileAnalyzer
	config       *ProjectClassifierConfig
	logger       *slog.Logger
}

// NewProjectClassifier creates a new project classifier
func NewProjectClassifier(
	fileAnalyzer FileAnalyzer,
	config *ProjectClassifierConfig,
	logger *slog.Logger,
) ProjectClassifier {
	if config == nil {
		config = DefaultProjectClassifierConfig()
	}

	return &projectClassifierImpl{
		fileAnalyzer: fileAnalyzer,
		config:       config,
		logger:       logger,
	}
}

// ClassifyProject analyzes a project and returns its type with confidence
func (c *projectClassifierImpl) ClassifyProject(
	ctx context.Context,
	path string,
) (entities.ProjectType, float64, error) {
	c.logger.Info("classifying project", slog.String("path", path))

	// Get project characteristics
	characteristics, err := c.GetProjectCharacteristics(ctx, path)
	if err != nil {
		return entities.ProjectTypeUnknown, 0.0, fmt.Errorf("failed to analyze project: %w", err)
	}

	// Apply classification rules
	projectType, confidence := c.SuggestProjectType(characteristics)

	c.logger.Info("project classification completed",
		slog.String("type", string(projectType)),
		slog.Float64("confidence", confidence))

	return projectType, confidence, nil
}

// GetProjectCharacteristics analyzes project and returns characteristics
func (c *projectClassifierImpl) GetProjectCharacteristics(
	ctx context.Context,
	path string,
) (*entities.ProjectCharacteristics, error) {
	// Analyze directory structure
	analysis, err := c.fileAnalyzer.AnalyzeDirectory(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze directory: %w", err)
	}

	// Get config files
	configFiles, err := c.fileAnalyzer.ReadConfigFiles(ctx, path)
	if err != nil {
		c.logger.Warn("failed to read config files", slog.Any("error", err))
		configFiles = make(map[string]interface{})
	}

	characteristics := &entities.ProjectCharacteristics{
		Languages:      c.detectLanguages(analysis.FilesByExt),
		Frameworks:     c.detectFrameworks(configFiles),
		Dependencies:   c.extractDependencies(configFiles),
		FilePatterns:   c.analyzeFilePatterns(analysis),
		HasTests:       c.detectTests(analysis.FilesByExt),
		HasCI:          c.detectCI(analysis.ConfigFiles),
		HasDocker:      c.detectDocker(analysis.ConfigFiles),
		HasDatabase:    c.detectDatabase(configFiles),
		HasAPI:         c.detectAPI(analysis.FilesByExt, configFiles),
		HasFrontend:    c.detectFrontend(analysis.FilesByExt, configFiles),
		HasBackend:     c.detectBackend(analysis.FilesByExt, configFiles),
		DirectoryDepth: analysis.DirectoryDepth,
		TotalFiles:     analysis.TotalFiles,
		ConfigFiles:    analysis.ConfigFiles,
		BuildFiles:     c.detectBuildFiles(analysis.ConfigFiles),
		Metadata:       make(map[string]interface{}),
	}

	// Add metadata
	characteristics.Metadata["primary_language"] = characteristics.GetPrimaryLanguage()
	characteristics.Metadata["complexity_score"] = characteristics.GetComplexityScore()
	characteristics.Metadata["is_monorepo"] = characteristics.IsMonorepo()

	return characteristics, nil
}

// SuggestProjectType analyzes characteristics and suggests project type
func (c *projectClassifierImpl) SuggestProjectType(
	chars *entities.ProjectCharacteristics,
) (entities.ProjectType, float64) {
	scores := make(map[entities.ProjectType]float64)

	// Calculate scores for each project type
	c.scoreWebApp(chars, scores)
	c.scoreCLI(chars, scores)
	c.scoreAPI(chars, scores)
	c.scoreLibrary(chars, scores)
	c.scoreMicroservice(chars, scores)
	c.scoreMobile(chars, scores)
	c.scoreDesktop(chars, scores)
	c.scoreDataPipeline(chars, scores)
	c.scoreGame(chars, scores)

	// Find best match
	bestType, bestScore := c.findBestMatch(scores)

	// Normalize confidence
	confidence := c.normalizeConfidence(bestScore)

	// Use fallback if confidence is low
	if confidence < 0.5 {
		bestType, confidence = c.fallbackClassification(chars)
	}

	return bestType, confidence
}

// scoreWebApp calculates web application indicators
func (c *projectClassifierImpl) scoreWebApp(chars *entities.ProjectCharacteristics, scores map[entities.ProjectType]float64) {
	if chars.HasFrontend && (chars.HasAPI || chars.HasBackend) {
		scores[entities.ProjectTypeWebApp] += 0.8
	}
	if chars.HasFramework("react", "vue", "angular", "next", "nuxt", "svelte") {
		scores[entities.ProjectTypeWebApp] += 0.3
	}
	if chars.HasFrontend && chars.HasDatabase {
		scores[entities.ProjectTypeWebApp] += 0.2
	}
}

// scoreCLI calculates command-line application indicators
func (c *projectClassifierImpl) scoreCLI(chars *entities.ProjectCharacteristics, scores map[entities.ProjectType]float64) {
	if chars.HasMainFile() && !chars.HasFrontend && !chars.HasAPI {
		scores[entities.ProjectTypeCLI] += 0.7
	}
	if chars.HasFramework("cobra", "cli", "commander") {
		scores[entities.ProjectTypeCLI] += 0.4
	}
	primaryLang := chars.GetPrimaryLanguage()
	if (primaryLang == "go" || primaryLang == "rust" || primaryLang == "c") && !chars.HasFrontend {
		scores[entities.ProjectTypeCLI] += 0.3
	}
}

// scoreAPI calculates API service indicators
func (c *projectClassifierImpl) scoreAPI(chars *entities.ProjectCharacteristics, scores map[entities.ProjectType]float64) {
	if chars.HasAPI && !chars.HasFrontend {
		scores[entities.ProjectTypeAPI] += 0.8
	}
	if chars.HasFramework("express", "fastapi", "gin", "echo", "django", "flask", "spring") {
		scores[entities.ProjectTypeAPI] += 0.4
	}
	if chars.HasDatabase && chars.HasBackend && !chars.HasFrontend {
		scores[entities.ProjectTypeAPI] += 0.3
	}
}

// scoreLibrary calculates library/package indicators
func (c *projectClassifierImpl) scoreLibrary(chars *entities.ProjectCharacteristics, scores map[entities.ProjectType]float64) {
	if chars.TotalFiles < 50 && !chars.HasMainFile() && !chars.HasFrontend {
		scores[entities.ProjectTypeLibrary] += 0.6
	}
	if len(chars.Dependencies) < 5 && chars.TotalFiles < 100 {
		scores[entities.ProjectTypeLibrary] += 0.3
	}
}

// scoreMicroservice calculates microservice indicators
func (c *projectClassifierImpl) scoreMicroservice(chars *entities.ProjectCharacteristics, scores map[entities.ProjectType]float64) {
	if chars.HasDocker && chars.HasAPI && chars.DirectoryDepth <= 4 && chars.TotalFiles < 200 {
		scores[entities.ProjectTypeMicroservice] += 0.7
	}
	if chars.HasKubernetes() || chars.HasServiceMesh() {
		scores[entities.ProjectTypeMicroservice] += 0.4
	}
}

// scoreMobile calculates mobile application indicators
func (c *projectClassifierImpl) scoreMobile(chars *entities.ProjectCharacteristics, scores map[entities.ProjectType]float64) {
	if chars.HasFramework("react-native", "flutter", "ionic", "xamarin") {
		scores[entities.ProjectTypeMobile] += 0.9
	}
	primaryLang := chars.GetPrimaryLanguage()
	if primaryLang == "swift" || primaryLang == "kotlin" || primaryLang == "dart" {
		scores[entities.ProjectTypeMobile] += 0.6
	}
}

// scoreDesktop calculates desktop application indicators
func (c *projectClassifierImpl) scoreDesktop(chars *entities.ProjectCharacteristics, scores map[entities.ProjectType]float64) {
	if chars.HasFramework("electron", "tauri", "qt", "gtk") {
		scores[entities.ProjectTypeDesktop] += 0.8
	}
	primaryLang := chars.GetPrimaryLanguage()
	if primaryLang == "cpp" || primaryLang == constants.LanguageCSharp || primaryLang == "java" {
		scores[entities.ProjectTypeDesktop] += 0.4
	}
}

// scoreDataPipeline calculates data pipeline indicators
func (c *projectClassifierImpl) scoreDataPipeline(chars *entities.ProjectCharacteristics, scores map[entities.ProjectType]float64) {
	if chars.HasFramework("airflow", "kafka", "spark", "flink") {
		scores[entities.ProjectTypeDataPipeline] += 0.8
	}
	primaryLang := chars.GetPrimaryLanguage()
	if primaryLang == constants.LanguagePython && chars.HasDatabase {
		scores[entities.ProjectTypeDataPipeline] += 0.4
	}
}

// scoreGame calculates game development indicators
func (c *projectClassifierImpl) scoreGame(chars *entities.ProjectCharacteristics, scores map[entities.ProjectType]float64) {
	if chars.HasFramework("unity", "unreal", "godot", "pygame") {
		scores[entities.ProjectTypeGame] += 0.9
	}
	primaryLang := chars.GetPrimaryLanguage()
	if primaryLang == constants.LanguageCSharp || primaryLang == "cpp" {
		scores[entities.ProjectTypeGame] += 0.3
	}
}

// findBestMatch finds the project type with highest score
func (c *projectClassifierImpl) findBestMatch(scores map[entities.ProjectType]float64) (entities.ProjectType, float64) {
	var bestType = entities.ProjectTypeUnknown
	var bestScore float64

	for pType, score := range scores {
		if score > bestScore {
			bestType = pType
			bestScore = score
		}
	}

	return bestType, bestScore
}

// normalizeConfidence ensures confidence is in 0-1 range
func (c *projectClassifierImpl) normalizeConfidence(score float64) float64 {
	if score > 1.0 {
		return 1.0
	}
	return score
}

// AnalyzeProjectStructure provides detailed project structure analysis
func (c *projectClassifierImpl) AnalyzeProjectStructure(
	ctx context.Context,
	path string,
) (*ProjectStructure, error) {
	analysis, err := c.fileAnalyzer.AnalyzeDirectory(ctx, path)
	if err != nil {
		return nil, err
	}

	configFiles, err := c.fileAnalyzer.ReadConfigFiles(ctx, path)
	if err != nil {
		configFiles = make(map[string]interface{})
	}

	// Convert file analysis to project structure
	structure := &ProjectStructure{
		RootPath:       path,
		Directories:    c.extractDirectories(analysis),
		Files:          c.convertFilesByExt(analysis.FilesByExt),
		ConfigFiles:    configFiles,
		Dependencies:   c.extractDependenciesByManager(configFiles),
		BuildSystems:   c.detectBuildSystems(configFiles),
		TestFrameworks: c.detectTestFrameworks(analysis.FilesByExt, configFiles),
		CIFiles:        c.filterCIFiles(analysis.ConfigFiles),
		DockerFiles:    c.filterDockerFiles(analysis.ConfigFiles),
		DatabaseFiles:  c.filterDatabaseFiles(analysis.ConfigFiles),
		AnalyzedAt:     time.Now(),
	}

	return structure, nil
}

// Helper methods

func (c *projectClassifierImpl) detectLanguages(filesByExt map[string]int) map[string]int {
	languages := make(map[string]int)

	for lang, extensions := range c.config.LanguageExtensions {
		count := 0
		for _, ext := range extensions {
			if files, exists := filesByExt[ext]; exists {
				count += files
			}
		}
		if count > 0 {
			languages[lang] = count
		}
	}

	return languages
}

func (c *projectClassifierImpl) detectFrameworks(configFiles map[string]interface{}) []string {
	var frameworks []string
	frameworkSet := make(map[string]bool)

	for _, detectors := range c.config.FrameworkDetectors {
		for _, detector := range detectors {
			if c.findInConfigFiles(configFiles, detector) {
				for framework, patterns := range c.config.FrameworkDetectors {
					for _, pattern := range patterns {
						if pattern == detector && !frameworkSet[framework] {
							frameworks = append(frameworks, framework)
							frameworkSet[framework] = true
						}
					}
				}
			}
		}
	}

	return frameworks
}

func (c *projectClassifierImpl) extractDependencies(configFiles map[string]interface{}) []string {
	var dependencies []string
	depSet := make(map[string]bool)

	// Extract from different config file types
	c.extractPackageJSONDeps(configFiles, &dependencies, depSet)
	c.extractGoModDeps(configFiles, &dependencies, depSet)
	c.extractPythonDeps(configFiles, &dependencies, depSet)

	return dependencies
}

// extractPackageJSONDeps extracts dependencies from package.json
func (c *projectClassifierImpl) extractPackageJSONDeps(configFiles map[string]interface{}, dependencies *[]string, depSet map[string]bool) {
	packageJSON, exists := configFiles["package.json"]
	if !exists {
		return
	}

	packageData, ok := packageJSON.(map[string]interface{})
	if !ok {
		return
	}

	// Extract regular dependencies
	c.extractJSONDepsSection(packageData, "dependencies", dependencies, depSet)

	// Extract dev dependencies
	c.extractJSONDepsSection(packageData, "devDependencies", dependencies, depSet)
}

// extractJSONDepsSection extracts a specific dependency section from package.json
func (c *projectClassifierImpl) extractJSONDepsSection(packageData map[string]interface{}, section string, dependencies *[]string, depSet map[string]bool) {
	deps, exists := packageData[section]
	if !exists {
		return
	}

	depsMap, ok := deps.(map[string]interface{})
	if !ok {
		return
	}

	for dep := range depsMap {
		c.addDependency(dep, dependencies, depSet)
	}
}

// extractGoModDeps extracts dependencies from go.mod
func (c *projectClassifierImpl) extractGoModDeps(configFiles map[string]interface{}, dependencies *[]string, depSet map[string]bool) {
	goMod, exists := configFiles["go.mod"]
	if !exists {
		return
	}

	goModStr, ok := goMod.(string)
	if !ok {
		return
	}

	lines := strings.Split(goModStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "require") || strings.HasPrefix(line, "//") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		dep := strings.TrimPrefix(parts[1], "require")
		dep = strings.TrimSpace(dep)
		if dep != "" {
			c.addDependency(dep, dependencies, depSet)
		}
	}
}

// extractPythonDeps extracts dependencies from requirements.txt
func (c *projectClassifierImpl) extractPythonDeps(configFiles map[string]interface{}, dependencies *[]string, depSet map[string]bool) {
	reqTxt, exists := configFiles["requirements.txt"]
	if !exists {
		return
	}

	reqStr, ok := reqTxt.(string)
	if !ok {
		return
	}

	lines := strings.Split(reqStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split on version specifiers (==, >=, <=, etc.)
		dep := strings.FieldsFunc(line, func(r rune) bool {
			return r == '=' || r == '>' || r == '<' || r == '!' || r == '~'
		})[0]

		dep = strings.TrimSpace(dep)
		c.addDependency(dep, dependencies, depSet)
	}
}

// addDependency adds a dependency if it's not already present
func (c *projectClassifierImpl) addDependency(dep string, dependencies *[]string, depSet map[string]bool) {
	if dep != "" && !depSet[dep] {
		*dependencies = append(*dependencies, dep)
		depSet[dep] = true
	}
}

func (c *projectClassifierImpl) analyzeFilePatterns(analysis *DirectoryAnalysis) map[string]int {
	patterns := make(map[string]int)

	// Count by extension
	for ext, count := range analysis.FilesByExt {
		patterns[ext] = count
	}

	// Count specific patterns
	configPatterns := []string{"config", "env", "docker", "k8s", "kubernetes", "helm"}
	for _, pattern := range configPatterns {
		count := 0
		for _, configFile := range analysis.ConfigFiles {
			if strings.Contains(strings.ToLower(configFile), pattern) {
				count++
			}
		}
		if count > 0 {
			patterns[pattern] = count
		}
	}

	return patterns
}

func (c *projectClassifierImpl) detectTests(filesByExt map[string]int) bool {
	testIndicators := []string{".test.js", ".spec.js", ".test.ts", ".spec.ts", "_test.go", "_test.py"}

	for _, indicator := range testIndicators {
		if count, exists := filesByExt[indicator]; exists && count > 0 {
			return true
		}
	}

	// Check for test directories or files
	for ext := range filesByExt {
		if strings.Contains(ext, "test") || strings.Contains(ext, "spec") {
			return true
		}
	}

	return false
}

func (c *projectClassifierImpl) detectCI(configFiles []string) bool {
	ciIndicators := []string{".github", ".gitlab-ci", "jenkins", "ci.yml", "pipeline"}

	for _, configFile := range configFiles {
		lowerFile := strings.ToLower(configFile)
		for _, indicator := range ciIndicators {
			if strings.Contains(lowerFile, indicator) {
				return true
			}
		}
	}

	return false
}

func (c *projectClassifierImpl) detectDocker(configFiles []string) bool {
	dockerIndicators := []string{"dockerfile", "docker-compose", ".dockerignore"}

	for _, configFile := range configFiles {
		lowerFile := strings.ToLower(configFile)
		for _, indicator := range dockerIndicators {
			if strings.Contains(lowerFile, indicator) {
				return true
			}
		}
	}

	return false
}

func (c *projectClassifierImpl) detectDatabase(configFiles map[string]interface{}) bool {
	dbIndicators := []string{
		"mysql", "postgresql", "postgres", "mongodb", "redis", "sqlite", "cassandra",
		"database", "db", "sql", "orm", "sequelize", "mongoose", "gorm",
	}

	for _, configFile := range configFiles {
		configStr := fmt.Sprintf("%v", configFile)
		lowerConfig := strings.ToLower(configStr)

		for _, indicator := range dbIndicators {
			if strings.Contains(lowerConfig, indicator) {
				return true
			}
		}
	}

	return false
}

func (c *projectClassifierImpl) detectAPI(filesByExt map[string]int, configFiles map[string]interface{}) bool {
	// Check for API-related files
	apiIndicators := []string{"router", "handler", "controller", "endpoint", "api"}

	for ext := range filesByExt {
		for _, indicator := range apiIndicators {
			if strings.Contains(strings.ToLower(ext), indicator) {
				return true
			}
		}
	}

	// Check config files for API frameworks
	apiFrameworks := []string{"express", "fastapi", "gin", "echo", "django", "flask", "spring"}
	for _, framework := range apiFrameworks {
		if c.findInConfigFiles(configFiles, framework) {
			return true
		}
	}

	return false
}

func (c *projectClassifierImpl) detectFrontend(filesByExt map[string]int, configFiles map[string]interface{}) bool {
	// Check for frontend files
	frontendExts := []string{".jsx", ".tsx", ".vue", ".svelte", ".html", ".css", ".scss", ".sass"}
	for _, ext := range frontendExts {
		if count, exists := filesByExt[ext]; exists && count > 0 {
			return true
		}
	}

	// Check for frontend frameworks
	frontendFrameworks := []string{"react", "vue", "angular", "svelte", "next", "nuxt"}
	for _, framework := range frontendFrameworks {
		if c.findInConfigFiles(configFiles, framework) {
			return true
		}
	}

	return false
}

func (c *projectClassifierImpl) detectBackend(filesByExt map[string]int, configFiles map[string]interface{}) bool {
	// Check for backend languages
	backendLangs := []string{".go", ".py", ".java", ".cs", ".php", ".rb"}
	backendCount := 0

	for _, ext := range backendLangs {
		if count, exists := filesByExt[ext]; exists {
			backendCount += count
		}
	}

	// If significant backend code exists
	if backendCount > 5 {
		return true
	}

	// Check for backend frameworks
	backendFrameworks := []string{"express", "django", "flask", "spring", "gin", "echo", "rails", "laravel"}
	for _, framework := range backendFrameworks {
		if c.findInConfigFiles(configFiles, framework) {
			return true
		}
	}

	return false
}

func (c *projectClassifierImpl) detectBuildFiles(configFiles []string) []string {
	var buildFiles []string
	buildIndicators := []string{"makefile", "dockerfile", "build", "webpack", "rollup", "vite", "gulpfile", "gruntfile"}

	for _, configFile := range configFiles {
		lowerFile := strings.ToLower(configFile)
		for _, indicator := range buildIndicators {
			if strings.Contains(lowerFile, indicator) {
				buildFiles = append(buildFiles, configFile)
			}
		}
	}

	return buildFiles
}

func (c *projectClassifierImpl) findInConfigFiles(configFiles map[string]interface{}, searchTerm string) bool {
	searchTerm = strings.ToLower(searchTerm)

	for _, configContent := range configFiles {
		contentStr := strings.ToLower(fmt.Sprintf("%v", configContent))
		if strings.Contains(contentStr, searchTerm) {
			return true
		}
	}

	return false
}

func (c *projectClassifierImpl) fallbackClassification(chars *entities.ProjectCharacteristics) (entities.ProjectType, float64) {
	primaryLang := chars.GetPrimaryLanguage()

	// Language-based fallback
	switch primaryLang {
	case constants.LanguageJavaScript, constants.LanguageTypeScript:
		if chars.HasFrontend {
			return entities.ProjectTypeWebApp, 0.4
		}
		return entities.ProjectTypeAPI, 0.3
	case "go", "rust":
		return entities.ProjectTypeCLI, 0.4
	case constants.LanguagePython:
		return entities.ProjectTypeAPI, 0.3
	case "java", constants.LanguageCSharp:
		return entities.ProjectTypeAPI, 0.3
	}

	return entities.ProjectTypeUnknown, 0.0
}

// Additional helper methods for ProjectStructure analysis

func (c *projectClassifierImpl) extractDirectories(analysis *DirectoryAnalysis) []string {
	// This would be implemented based on the actual DirectoryAnalysis structure
	// For now, return common directories
	return []string{"src", "lib", "test", "docs", "config"}
}

func (c *projectClassifierImpl) convertFilesByExt(filesByExt map[string]int) map[string][]string {
	result := make(map[string][]string)
	for ext, count := range filesByExt {
		files := make([]string, count)
		for i := 0; i < count; i++ {
			files[i] = fmt.Sprintf("file%d%s", i+1, ext)
		}
		result[ext] = files
	}
	return result
}

func (c *projectClassifierImpl) extractDependenciesByManager(configFiles map[string]interface{}) map[string][]string {
	deps := make(map[string][]string)

	if _, exists := configFiles["package.json"]; exists {
		deps["npm"] = c.extractNpmDependencies(configFiles["package.json"])
	}
	if _, exists := configFiles["go.mod"]; exists {
		deps["go"] = c.extractGoDependencies(configFiles["go.mod"])
	}
	if _, exists := configFiles["requirements.txt"]; exists {
		deps["pip"] = c.extractPipDependencies(configFiles["requirements.txt"])
	}

	return deps
}

func (c *projectClassifierImpl) extractNpmDependencies(packageJSON interface{}) []string {
	var deps []string
	if packageData, ok := packageJSON.(map[string]interface{}); ok {
		if dependencies, exists := packageData["dependencies"]; exists {
			if depsMap, ok := dependencies.(map[string]interface{}); ok {
				for dep := range depsMap {
					deps = append(deps, dep)
				}
			}
		}
	}
	return deps
}

func (c *projectClassifierImpl) extractGoDependencies(goMod interface{}) []string {
	var deps []string
	if goModStr, ok := goMod.(string); ok {
		lines := strings.Split(goModStr, "\n")
		for _, line := range lines {
			if strings.Contains(line, "require") && !strings.HasPrefix(strings.TrimSpace(line), "//") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					deps = append(deps, parts[1])
				}
			}
		}
	}
	return deps
}

func (c *projectClassifierImpl) extractPipDependencies(reqTxt interface{}) []string {
	var deps []string
	if reqStr, ok := reqTxt.(string); ok {
		lines := strings.Split(reqStr, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				dep := strings.FieldsFunc(line, func(r rune) bool {
					return r == '=' || r == '>' || r == '<'
				})[0]
				deps = append(deps, strings.TrimSpace(dep))
			}
		}
	}
	return deps
}

func (c *projectClassifierImpl) detectBuildSystems(configFiles map[string]interface{}) []string {
	var buildSystems []string

	if _, exists := configFiles["Makefile"]; exists {
		buildSystems = append(buildSystems, "make")
	}
	if _, exists := configFiles["package.json"]; exists {
		buildSystems = append(buildSystems, "npm")
	}
	if _, exists := configFiles["go.mod"]; exists {
		buildSystems = append(buildSystems, "go")
	}
	if _, exists := configFiles["Cargo.toml"]; exists {
		buildSystems = append(buildSystems, "cargo")
	}
	if _, exists := configFiles["pom.xml"]; exists {
		buildSystems = append(buildSystems, "maven")
	}
	if _, exists := configFiles["build.gradle"]; exists {
		buildSystems = append(buildSystems, "gradle")
	}

	return buildSystems
}

func (c *projectClassifierImpl) detectTestFrameworks(filesByExt map[string]int, configFiles map[string]interface{}) []string {
	var frameworks []string

	// Check for test files
	if filesByExt[".test.js"] > 0 || filesByExt[".spec.js"] > 0 {
		frameworks = append(frameworks, "jest")
	}
	if filesByExt["_test.go"] > 0 {
		frameworks = append(frameworks, "go test")
	}
	if filesByExt["_test.py"] > 0 {
		frameworks = append(frameworks, "pytest")
	}

	// Check config files for test frameworks
	testFrameworks := []string{"jest", "mocha", "karma", "jasmine", "cypress", "playwright"}
	for _, framework := range testFrameworks {
		if c.findInConfigFiles(configFiles, framework) {
			frameworks = append(frameworks, framework)
		}
	}

	return frameworks
}

func (c *projectClassifierImpl) filterCIFiles(configFiles []string) []string {
	var ciFiles []string
	ciIndicators := []string{".github", ".gitlab-ci", "jenkins", "ci.yml"}

	for _, file := range configFiles {
		for _, indicator := range ciIndicators {
			if strings.Contains(strings.ToLower(file), indicator) {
				ciFiles = append(ciFiles, file)
			}
		}
	}

	return ciFiles
}

func (c *projectClassifierImpl) filterDockerFiles(configFiles []string) []string {
	var dockerFiles []string
	dockerIndicators := []string{"dockerfile", "docker-compose"}

	for _, file := range configFiles {
		for _, indicator := range dockerIndicators {
			if strings.Contains(strings.ToLower(file), indicator) {
				dockerFiles = append(dockerFiles, file)
			}
		}
	}

	return dockerFiles
}

func (c *projectClassifierImpl) filterDatabaseFiles(configFiles []string) []string {
	var dbFiles []string
	dbIndicators := []string{"migration", "schema", "seed", ".sql"}

	for _, file := range configFiles {
		for _, indicator := range dbIndicators {
			if strings.Contains(strings.ToLower(file), indicator) {
				dbFiles = append(dbFiles, file)
			}
		}
	}

	return dbFiles
}
