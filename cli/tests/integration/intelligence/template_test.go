//go:build integration
// +build integration

package intelligence

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/services"
	"lerian-mcp-memory-cli/tests/testutils"
)

type TemplateTestSuite struct {
	suite.Suite
	templateStore   *testutils.MockTemplateStorage
	taskStore       *testutils.MockTaskStorage
	templateService services.TemplateService
	classifier      services.ProjectClassifier
	testData        *testutils.TestDataGenerator
	tempDirs        []string
}

func (s *TemplateTestSuite) SetupSuite() {
	s.templateStore = testutils.NewMockTemplateStorage()
	s.taskStore = testutils.NewMockTaskStorage()
	s.testData = testutils.NewTestDataGenerator()

	s.classifier = services.NewProjectClassifier(services.ProjectClassifierDependencies{
		AI:     testutils.NewMockAIService(),
		Logger: slog.Default(),
	})

	s.templateService = services.NewTemplateService(services.TemplateServiceDependencies{
		TemplateStore:     s.templateStore,
		TaskStore:         s.taskStore,
		ProjectClassifier: s.classifier,
		Logger:            slog.Default(),
	})
}

func (s *TemplateTestSuite) TearDownSuite() {
	// Clean up temporary directories
	for _, dir := range s.tempDirs {
		os.RemoveAll(dir)
	}
}

func (s *TemplateTestSuite) TearDownTest() {
	s.templateStore.Clear()
	s.taskStore.Clear()
}

func (s *TemplateTestSuite) TestProjectTypeDetection() {
	testCases := []struct {
		name          string
		projectFiles  map[string]string
		expectedType  entities.ProjectType
		minConfidence float64
	}{
		{
			name: "React Web Application",
			projectFiles: map[string]string{
				"package.json": `{
					"name": "my-react-app",
					"dependencies": {
						"react": "^18.2.0",
						"react-dom": "^18.2.0"
					},
					"scripts": {
						"start": "react-scripts start",
						"build": "react-scripts build"
					}
				}`,
				"src/App.js":        "import React from 'react';\nexport default function App() { return <div>Hello</div>; }",
				"src/index.js":      "import React from 'react';\nimport ReactDOM from 'react-dom/client';",
				"public/index.html": "<!DOCTYPE html><html><head><title>React App</title></head><body><div id=\"root\"></div></body></html>",
			},
			expectedType:  entities.ProjectTypeWebApp,
			minConfidence: 0.8,
		},
		{
			name: "Go CLI Tool",
			projectFiles: map[string]string{
				"main.go": `package main

import (
	"fmt"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "mycli",
	Short: "A CLI application",
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
	}
}`,
				"cmd/root.go": `package cmd

import "github.com/spf13/cobra"

var RootCmd = &cobra.Command{
	Use: "mycli",
}`,
				"go.mod": `module mycli

go 1.21

require github.com/spf13/cobra v1.7.0`,
				"README.md": "# My CLI Tool\n\nA command-line interface built with Go and Cobra.",
			},
			expectedType:  entities.ProjectTypeCLI,
			minConfidence: 0.7,
		},
		{
			name: "REST API Service",
			projectFiles: map[string]string{
				"main.go": `package main

import (
	"net/http"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.GET("/api/health", healthCheck)
	r.POST("/api/users", createUser)
	r.Run(":8080")
}`,
				"handlers/users.go": `package handlers

import (
	"net/http"
	"github.com/gin-gonic/gin"
)

func GetUsers(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"users": []string{}})
}`,
				"middleware/auth.go": `package middleware

import "github.com/gin-gonic/gin"

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Auth logic
		c.Next()
	}
}`,
				"models/user.go": `package models

type User struct {
	ID    uint   ` + "`" + `json:"id"` + "`" + `
	Name  string ` + "`" + `json:"name"` + "`" + `
	Email string ` + "`" + `json:"email"` + "`" + `
}`,
				"Dockerfile": `FROM golang:1.21-alpine
WORKDIR /app
COPY . .
RUN go build -o api
EXPOSE 8080
CMD ["./api"]`,
			},
			expectedType:  entities.ProjectTypeAPI,
			minConfidence: 0.8,
		},
		{
			name: "Python Data Science Project",
			projectFiles: map[string]string{
				"requirements.txt": `pandas==2.0.0
numpy==1.24.0
matplotlib==3.7.0
jupyter==1.0.0
scikit-learn==1.2.0`,
				"main.py": `import pandas as pd
import numpy as np
import matplotlib.pyplot as plt
from sklearn.linear_model import LinearRegression

def analyze_data():
    df = pd.read_csv('data.csv')
    # Analysis code here
    pass`,
				"notebooks/analysis.ipynb": `{
	"cells": [
		{
			"cell_type": "code",
			"source": ["import pandas as pd\\nimport numpy as np"]
		}
	]
}`,
				"data/sample.csv": "id,value\n1,10\n2,20\n3,30",
			},
			expectedType:  entities.ProjectTypeDataScience,
			minConfidence: 0.7,
		},
		{
			name: "Mobile App (React Native)",
			projectFiles: map[string]string{
				"package.json": `{
					"name": "MyMobileApp",
					"dependencies": {
						"react-native": "0.72.0",
						"react": "18.2.0"
					},
					"scripts": {
						"android": "react-native run-android",
						"ios": "react-native run-ios"
					}
				}`,
				"App.js": `import React from 'react';
import {SafeAreaView, Text} from 'react-native';

const App = () => {
	return (
		<SafeAreaView>
			<Text>Hello Mobile World!</Text>
		</SafeAreaView>
	);
};

export default App;`,
				"android/app/build.gradle":                  "apply plugin: 'com.android.application'",
				"ios/MyMobileApp.xcodeproj/project.pbxproj": "// Xcode project file",
			},
			expectedType:  entities.ProjectTypeMobile,
			minConfidence: 0.7,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Create temporary project directory
			tempDir := s.testData.CreateTempProject(tc.projectFiles)
			s.tempDirs = append(s.tempDirs, tempDir)

			// Classify project
			projectType, confidence, err := s.classifier.ClassifyProject(context.Background(), tempDir)

			s.Require().NoError(err)
			s.Assert().Equal(tc.expectedType, projectType, "Project type should match expected")
			s.Assert().GreaterOrEqual(confidence, tc.minConfidence, "Confidence should meet minimum threshold")

			s.T().Logf("Classified '%s' as %s with confidence %.2f", tc.name, projectType, confidence)
		})
	}
}

func (s *TemplateTestSuite) TestTemplateInstantiation() {
	ctx := context.Background()

	// Create a web app template
	template := &entities.ProjectTemplate{
		ID:          uuid.New().String(),
		Name:        "React TypeScript Web App",
		Description: "Full-stack React application with TypeScript",
		ProjectType: entities.ProjectTypeWebApp,
		Framework:   "react",
		Language:    "typescript",
		Tasks: []entities.TemplateTask{
			{
				ID:          uuid.New().String(),
				Name:        "Setup {{.Framework}} project with {{.Language}}",
				Description: "Initialize new {{.Framework}} project with {{.Language}} support",
				Type:        "setup",
				Priority:    entities.PriorityHigh,
				Order:       1,
				Metadata: map[string]interface{}{
					"category": "initialization",
					"tools":    []string{"create-react-app", "typescript"},
				},
			},
			{
				ID:           uuid.New().String(),
				Name:         "Configure {{.Database}} database connection",
				Description:  "Set up database configuration for {{.Database}}",
				Type:         "configuration",
				Priority:     entities.PriorityHigh,
				Order:        2,
				Dependencies: []string{}, // Will be filled with first task ID
				Metadata: map[string]interface{}{
					"category": "database",
					"optional": false,
				},
			},
			{
				ID:           uuid.New().String(),
				Name:         "Implement user authentication with {{.AuthMethod}}",
				Description:  "Set up user authentication using {{.AuthMethod}}",
				Type:         "feature",
				Priority:     entities.PriorityHigh,
				Order:        3,
				Dependencies: []string{}, // Will be filled with second task ID
				Metadata: map[string]interface{}{
					"category":  "security",
					"auth_type": "{{.AuthMethod}}",
				},
			},
			{
				ID:           uuid.New().String(),
				Name:         "Create responsive UI components",
				Description:  "Build reusable UI components with responsive design",
				Type:         "implementation",
				Priority:     entities.PriorityMedium,
				Order:        4,
				Dependencies: []string{}, // Will be filled with first task ID
				Metadata: map[string]interface{}{
					"category":   "frontend",
					"responsive": true,
				},
			},
			{
				ID:           uuid.New().String(),
				Name:         "Write unit tests for {{.Framework}} components",
				Description:  "Create comprehensive test suite",
				Type:         "testing",
				Priority:     entities.PriorityMedium,
				Order:        5,
				Dependencies: []string{}, // Will be filled with UI task ID
				Metadata: map[string]interface{}{
					"category":  "testing",
					"test_type": "unit",
				},
			},
		},
		Variables: []entities.TemplateVariable{
			{
				Name:        "Framework",
				Type:        "string",
				Default:     "react",
				Required:    true,
				Options:     []string{"react", "vue", "angular"},
				Description: "Frontend framework to use",
			},
			{
				Name:        "Language",
				Type:        "string",
				Default:     "typescript",
				Required:    true,
				Options:     []string{"typescript", "javascript"},
				Description: "Programming language for frontend",
			},
			{
				Name:        "Database",
				Type:        "string",
				Default:     "postgresql",
				Required:    true,
				Options:     []string{"postgresql", "mysql", "mongodb"},
				Description: "Database system to use",
			},
			{
				Name:        "AuthMethod",
				Type:        "string",
				Default:     "jwt",
				Required:    false,
				Options:     []string{"jwt", "oauth", "session"},
				Description: "Authentication method",
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Fix dependencies (reference actual task IDs)
	template.Tasks[1].Dependencies = []string{template.Tasks[0].ID}
	template.Tasks[2].Dependencies = []string{template.Tasks[1].ID}
	template.Tasks[3].Dependencies = []string{template.Tasks[0].ID}
	template.Tasks[4].Dependencies = []string{template.Tasks[3].ID}

	err := s.templateStore.Create(ctx, template)
	s.Require().NoError(err)

	// Instantiate template with variables
	variables := map[string]interface{}{
		"repository": "my-web-app",
		"Framework":  "react",
		"Language":   "typescript",
		"Database":   "postgresql",
		"AuthMethod": "jwt",
	}

	tasks, err := s.templateService.InstantiateTemplate(ctx, template.ID, variables)
	s.Require().NoError(err)
	s.Assert().Len(tasks, 5, "Should create all template tasks")

	// Verify variable substitution
	setupTask := s.findTaskByName(tasks, "Setup react project with typescript")
	s.Assert().NotNil(setupTask, "Should find setup task with substituted variables")

	dbTask := s.findTaskByName(tasks, "Configure postgresql database connection")
	s.Assert().NotNil(dbTask, "Should find database task with substituted variables")

	authTask := s.findTaskByName(tasks, "Implement user authentication with jwt")
	s.Assert().NotNil(authTask, "Should find auth task with substituted variables")

	testTask := s.findTaskByName(tasks, "Write unit tests for react components")
	s.Assert().NotNil(testTask, "Should find test task with substituted variables")

	// Verify dependencies are preserved
	s.Assert().Contains(dbTask.Metadata["depends_on"], setupTask.ID, "Database task should depend on setup")
	s.Assert().Contains(authTask.Metadata["depends_on"], dbTask.ID, "Auth task should depend on database")

	// Verify metadata substitution
	authMetadata := authTask.Metadata["auth_type"]
	s.Assert().Equal("jwt", authMetadata, "Should substitute variables in metadata")

	// Verify all tasks have repository set
	for _, task := range tasks {
		s.Assert().Equal("my-web-app", task.Repository, "All tasks should have correct repository")
		s.Assert().NotEmpty(task.ID, "All tasks should have IDs")
		s.Assert().NotEmpty(task.Content, "All tasks should have content")
	}

	s.T().Logf("Successfully instantiated template with %d tasks", len(tasks))
}

func (s *TemplateTestSuite) TestTemplateMatching() {
	ctx := context.Background()

	// Create templates for different project types
	templates := []*entities.ProjectTemplate{
		{
			ID:          uuid.New().String(),
			Name:        "React Web App",
			ProjectType: entities.ProjectTypeWebApp,
			Framework:   "react",
			Language:    "javascript",
			Tags:        []string{"frontend", "spa", "react"},
			CreatedAt:   time.Now(),
		},
		{
			ID:          uuid.New().String(),
			Name:        "Go CLI Tool",
			ProjectType: entities.ProjectTypeCLI,
			Framework:   "cobra",
			Language:    "go",
			Tags:        []string{"cli", "golang", "cobra"},
			CreatedAt:   time.Now(),
		},
		{
			ID:          uuid.New().String(),
			Name:        "REST API Service",
			ProjectType: entities.ProjectTypeAPI,
			Framework:   "gin",
			Language:    "go",
			Tags:        []string{"api", "rest", "golang", "microservice"},
			CreatedAt:   time.Now(),
		},
	}

	for _, template := range templates {
		err := s.templateStore.Create(ctx, template)
		s.Require().NoError(err)
	}

	// Test matching for React project
	reactProject := s.testData.CreateTempProject(map[string]string{
		"package.json": `{"dependencies": {"react": "^18.0.0"}}`,
		"src/App.js":   "import React from 'react';",
	})
	s.tempDirs = append(s.tempDirs, reactProject)

	matches, err := s.templateService.MatchTemplates(ctx, reactProject)
	s.Require().NoError(err)
	s.Assert().NotEmpty(matches, "Should find matching templates")

	// Should prioritize React template
	bestMatch := matches[0]
	s.Assert().Equal("React Web App", bestMatch.Template.Name)
	s.Assert().Greater(bestMatch.Confidence, 0.7, "Should have high confidence for React match")

	// Test matching for Go CLI project
	cliProject := s.testData.CreateTempProject(map[string]string{
		"main.go":     "package main\nimport \"github.com/spf13/cobra\"",
		"cmd/root.go": "cobra.Command",
		"go.mod":      "module mycli\nrequire github.com/spf13/cobra",
	})
	s.tempDirs = append(s.tempDirs, cliProject)

	cliMatches, err := s.templateService.MatchTemplates(ctx, cliProject)
	s.Require().NoError(err)
	s.Assert().NotEmpty(cliMatches, "Should find CLI template matches")

	bestCLIMatch := cliMatches[0]
	s.Assert().Equal("Go CLI Tool", bestCLIMatch.Template.Name)
	s.Assert().Greater(bestCLIMatch.Confidence, 0.6, "Should have good confidence for CLI match")

	s.T().Logf("React project matched %d templates, best: %s (%.2f)",
		len(matches), bestMatch.Template.Name, bestMatch.Confidence)
	s.T().Logf("CLI project matched %d templates, best: %s (%.2f)",
		len(cliMatches), bestCLIMatch.Template.Name, bestCLIMatch.Confidence)
}

func (s *TemplateTestSuite) TestBuiltInTemplates() {
	ctx := context.Background()

	// Get built-in templates
	builtInTemplates := s.templateService.GetBuiltInTemplates()
	s.Assert().NotEmpty(builtInTemplates, "Should have built-in templates")

	// Verify we have templates for major project types
	projectTypes := make(map[entities.ProjectType]bool)
	for _, template := range builtInTemplates {
		projectTypes[template.ProjectType] = true

		// Verify template structure
		s.Assert().NotEmpty(template.ID, "Template should have ID")
		s.Assert().NotEmpty(template.Name, "Template should have name")
		s.Assert().NotEmpty(template.Description, "Template should have description")
		s.Assert().NotEmpty(template.Tasks, "Template should have tasks")
		s.Assert().NotNil(template.Variables, "Template should have variables")

		// Verify tasks have proper structure
		for _, task := range template.Tasks {
			s.Assert().NotEmpty(task.ID, "Task should have ID")
			s.Assert().NotEmpty(task.Name, "Task should have name")
			s.Assert().NotEmpty(task.Type, "Task should have type")
			s.Assert().Greater(task.Order, 0, "Task should have valid order")
		}
	}

	// Check for essential project types
	expectedTypes := []entities.ProjectType{
		entities.ProjectTypeWebApp,
		entities.ProjectTypeAPI,
		entities.ProjectTypeCLI,
		entities.ProjectTypeMobile,
	}

	for _, expectedType := range expectedTypes {
		s.Assert().True(projectTypes[expectedType],
			"Should have built-in template for %s", expectedType)
	}

	s.T().Logf("Found %d built-in templates covering %d project types",
		len(builtInTemplates), len(projectTypes))
}

func (s *TemplateTestSuite) TestTemplateValidation() {
	ctx := context.Background()

	// Test valid template
	validTemplate := &entities.ProjectTemplate{
		ID:          uuid.New().String(),
		Name:        "Valid Template",
		Description: "A valid template for testing",
		ProjectType: entities.ProjectTypeWebApp,
		Tasks: []entities.TemplateTask{
			{
				ID:       uuid.New().String(),
				Name:     "Setup project",
				Type:     "setup",
				Priority: entities.PriorityHigh,
				Order:    1,
			},
		},
		Variables: []entities.TemplateVariable{
			{
				Name:     "ProjectName",
				Type:     "string",
				Required: true,
			},
		},
		CreatedAt: time.Now(),
	}

	err := s.templateService.ValidateTemplate(validTemplate)
	s.Assert().NoError(err, "Valid template should pass validation")

	// Test invalid template - missing required fields
	invalidTemplate := &entities.ProjectTemplate{
		ID:   uuid.New().String(),
		Name: "", // Missing name
		Tasks: []entities.TemplateTask{
			{
				ID:    uuid.New().String(),
				Name:  "", // Missing task name
				Type:  "",
				Order: 0, // Invalid order
			},
		},
	}

	err = s.templateService.ValidateTemplate(invalidTemplate)
	s.Assert().Error(err, "Invalid template should fail validation")

	// Test template with circular dependencies
	circularTemplate := &entities.ProjectTemplate{
		ID:   uuid.New().String(),
		Name: "Circular Dependencies",
		Tasks: []entities.TemplateTask{
			{
				ID:           uuid.New().String(),
				Name:         "Task 1",
				Type:         "setup",
				Order:        1,
				Dependencies: []string{"task-2"},
			},
			{
				ID:           "task-2",
				Name:         "Task 2",
				Type:         "config",
				Order:        2,
				Dependencies: []string{"task-1"}, // Circular dependency
			},
		},
	}

	err = s.templateService.ValidateTemplate(circularTemplate)
	s.Assert().Error(err, "Template with circular dependencies should fail validation")
}

func (s *TemplateTestSuite) TestTemplateCustomization() {
	ctx := context.Background()

	// Create base template
	baseTemplate := &entities.ProjectTemplate{
		ID:          uuid.New().String(),
		Name:        "Base Web App",
		ProjectType: entities.ProjectTypeWebApp,
		Tasks: []entities.TemplateTask{
			{
				ID:       uuid.New().String(),
				Name:     "Setup {{.Framework}} project",
				Type:     "setup",
				Priority: entities.PriorityHigh,
				Order:    1,
			},
			{
				ID:       uuid.New().String(),
				Name:     "Implement authentication",
				Type:     "feature",
				Priority: entities.PriorityMedium,
				Order:    2,
			},
		},
		Variables: []entities.TemplateVariable{
			{
				Name:     "Framework",
				Type:     "string",
				Default:  "react",
				Required: true,
				Options:  []string{"react", "vue", "angular"},
			},
		},
		CreatedAt: time.Now(),
	}

	err := s.templateStore.Create(ctx, baseTemplate)
	s.Require().NoError(err)

	// Customize template for specific needs
	customizations := map[string]interface{}{
		"additional_tasks": []map[string]interface{}{
			{
				"name":     "Setup testing framework",
				"type":     "testing",
				"priority": "medium",
				"order":    3,
			},
			{
				"name":     "Deploy to production",
				"type":     "deployment",
				"priority": "low",
				"order":    4,
			},
		},
		"remove_tasks": []string{"Implement authentication"},
		"modify_variables": map[string]interface{}{
			"Framework": map[string]interface{}{
				"default": "vue",
				"options": []string{"vue", "nuxt"},
			},
		},
	}

	customizedTemplate, err := s.templateService.CustomizeTemplate(ctx, baseTemplate.ID, customizations)
	s.Require().NoError(err)

	// Verify customizations applied
	s.Assert().Len(customizedTemplate.Tasks, 3, "Should have base + additional tasks minus removed")

	// Check that authentication task was removed
	authTask := s.findTaskByName(customizedTemplate.Tasks, "Implement authentication")
	s.Assert().Nil(authTask, "Authentication task should be removed")

	// Check that new tasks were added
	testingTask := s.findTaskByName(customizedTemplate.Tasks, "Setup testing framework")
	s.Assert().NotNil(testingTask, "Testing task should be added")

	deployTask := s.findTaskByName(customizedTemplate.Tasks, "Deploy to production")
	s.Assert().NotNil(deployTask, "Deploy task should be added")

	// Check variable modification
	frameworkVar := s.findVariable(customizedTemplate.Variables, "Framework")
	s.Assert().NotNil(frameworkVar, "Framework variable should exist")
	s.Assert().Equal("vue", frameworkVar.Default, "Framework default should be updated")
	s.Assert().Equal([]string{"vue", "nuxt"}, frameworkVar.Options, "Framework options should be updated")
}

// Helper methods

func (s *TemplateTestSuite) findTaskByName(tasks []*entities.Task, name string) *entities.Task {
	for _, task := range tasks {
		if task.Content == name {
			return task
		}
	}
	return nil
}

func (s *TemplateTestSuite) findVariable(variables []entities.TemplateVariable, name string) *entities.TemplateVariable {
	for _, variable := range variables {
		if variable.Name == name {
			return &variable
		}
	}
	return nil
}

func TestTemplate(t *testing.T) {
	suite.Run(t, new(TemplateTestSuite))
}
