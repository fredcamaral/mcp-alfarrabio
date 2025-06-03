#!/usr/bin/env node

const fs = require('fs');
const { execSync } = require('child_process');
const path = require('path');

class ArchitectureMonitor {
  constructor() {
    this.components = ['mcp', 'storage', 'intelligence', 'workflow', 'embeddings'];
    this.metrics = {
      componentHealth: new Map(),
      dependencyCount: 0,
      apiEndpoints: 0,
      cycleComplexity: 0
    };
  }

  async analyzeArchitecture() {
    console.log('🔍 Starting MCP Memory Server architecture analysis...');

    const analysis = {
      timestamp: new Date().toISOString(),
      project: 'MCP Memory Server',
      language: 'Go',
      architecture: await this.detectArchitectureStyle(),
      components: await this.analyzeComponents(),
      dependencies: await this.analyzeDependencies(),
      endpoints: await this.countEndpoints(),
      codeMetrics: await this.analyzeCodeMetrics(),
      layerCompliance: await this.checkLayerCompliance(),
      healthScore: 0
    };

    analysis.healthScore = this.calculateHealthScore(analysis);

    console.log('📊 Architecture Analysis Results:');
    console.log(`   Architecture Style: ${analysis.architecture.style}`);
    console.log(`   Components Found: ${Object.keys(analysis.components).length}`);
    console.log(`   Go Dependencies: ${analysis.dependencies.go}`);
    console.log(`   Code Quality Score: ${analysis.healthScore}/100`);

    // Save detailed report
    const reportPath = '.claude/architecture-health.json';
    fs.writeFileSync(reportPath, JSON.stringify(analysis, null, 2));
    console.log(`💾 Detailed report saved to: ${reportPath}`);

    return analysis;
  }

  async detectArchitectureStyle() {
    try {
      // Check for DDD patterns
      const hasDomainLayer = fs.existsSync('pkg/types');
      const hasApplicationLayer = fs.existsSync('internal/mcp');
      const hasInfrastructureLayer = fs.existsSync('internal/storage');
      
      // Check for service boundaries
      const serviceCount = execSync('ls internal/ | wc -l', { encoding: 'utf8' }).trim();
      
      let style = 'Unknown';
      let confidence = 0;
      
      if (hasDomainLayer && hasApplicationLayer && hasInfrastructureLayer) {
        style = 'Layered Architecture with DDD';
        confidence = 0.95;
      } else if (parseInt(serviceCount) > 10) {
        style = 'Modular Monolith';
        confidence = 0.8;
      }
      
      return {
        style,
        confidence,
        evidence: {
          domainLayer: hasDomainLayer,
          applicationLayer: hasApplicationLayer,
          infrastructureLayer: hasInfrastructureLayer,
          serviceCount: parseInt(serviceCount)
        }
      };
    } catch (error) {
      return { style: 'Unknown', confidence: 0, error: error.message };
    }
  }

  async analyzeComponents() {
    const components = {};

    try {
      // Analyze internal packages
      const internalDirs = execSync('ls internal/', { encoding: 'utf8' })
        .trim().split('\n').filter(d => d);

      for (const dir of internalDirs) {
        const dirPath = `internal/${dir}`;
        if (fs.statSync(dirPath).isDirectory()) {
          components[dir] = {
            path: dirPath,
            files: this.countGoFiles(dirPath),
            linesOfCode: this.countLOC(dirPath),
            hasTests: this.hasTests(dirPath),
            dependencies: this.getPackageDependencies(dirPath),
            complexity: this.calculateComplexity(dirPath)
          };
        }
      }

      // Analyze cmd packages (entry points)
      if (fs.existsSync('cmd')) {
        const cmdDirs = execSync('ls cmd/', { encoding: 'utf8' })
          .trim().split('\n').filter(d => d);
        
        for (const dir of cmdDirs) {
          const dirPath = `cmd/${dir}`;
          components[`cmd-${dir}`] = {
            path: dirPath,
            files: this.countGoFiles(dirPath),
            linesOfCode: this.countLOC(dirPath),
            hasTests: this.hasTests(dirPath),
            isEntryPoint: true
          };
        }
      }
    } catch (error) {
      console.warn('Could not analyze components:', error.message);
    }

    return components;
  }

  async analyzeDependencies() {
    try {
      // Analyze go.mod for Go dependencies
      const goMod = fs.readFileSync('go.mod', 'utf8');
      const goRequires = (goMod.match(/require\s+\(/s) ? 
        goMod.split('require (')[1].split(')')[0] : 
        goMod.split('require')[1] || '').split('\n').filter(line => line.trim());
      
      // Count internal dependencies
      const internalImports = execSync(
        'grep -r "mcp-memory/internal" --include="*.go" . 2>/dev/null | wc -l', 
        { encoding: 'utf8' }
      ).trim();

      // Analyze external dependencies
      const externalDeps = goRequires.filter(dep => 
        dep.trim() && !dep.includes('mcp-memory') && !dep.includes('//'));

      return {
        go: externalDeps.length,
        internal: parseInt(internalImports) || 0,
        total: externalDeps.length + parseInt(internalImports),
        frameworks: this.identifyGoFrameworks(goRequires),
        externalDeps: externalDeps.slice(0, 10) // Top 10 for brevity
      };
    } catch (error) {
      return { error: 'Could not analyze Go dependencies: ' + error.message };
    }
  }

  async countEndpoints() {
    try {
      // Count MCP tools
      const mcpTools = execSync(
        'grep -r "AddTool\\|NewTool" internal/mcp/ --include="*.go" | wc -l', 
        { encoding: 'utf8' }
      ).trim();

      // Count HTTP handlers
      const httpHandlers = execSync(
        'grep -r "HandleFunc\\|mux\\.Handle" cmd/ --include="*.go" | wc -l', 
        { encoding: 'utf8' }
      ).trim();

      return {
        mcpTools: parseInt(mcpTools) || 0,
        httpHandlers: parseInt(httpHandlers) || 0,
        total: (parseInt(mcpTools) || 0) + (parseInt(httpHandlers) || 0)
      };
    } catch (error) {
      return { error: 'Could not count endpoints: ' + error.message };
    }
  }

  async analyzeCodeMetrics() {
    try {
      const goFiles = execSync('find . -name "*.go" | grep -v vendor | wc -l', { encoding: 'utf8' });
      const totalLOC = execSync('find . -name "*.go" | grep -v vendor | xargs wc -l | tail -1', { encoding: 'utf8' });
      
      // Calculate average function size
      const functions = execSync('grep -r "^func " --include="*.go" . | wc -l', { encoding: 'utf8' });
      const avgFunctionSize = Math.round((parseInt(totalLOC.trim().split(' ')[0]) || 0) / (parseInt(functions.trim()) || 1));

      return {
        totalFiles: parseInt(goFiles.trim()) || 0,
        totalLOC: parseInt(totalLOC.trim().split(' ')[0]) || 0,
        totalFunctions: parseInt(functions.trim()) || 0,
        avgLOCPerFile: Math.round((parseInt(totalLOC.trim().split(' ')[0]) || 0) / (parseInt(goFiles.trim()) || 1)),
        avgFunctionSize: avgFunctionSize,
        complexity: avgFunctionSize > 50 ? 'High' : avgFunctionSize > 25 ? 'Medium' : 'Low'
      };
    } catch (error) {
      return { error: 'Could not analyze code metrics: ' + error.message };
    }
  }

  async checkLayerCompliance() {
    try {
      // Check for layer violations (infrastructure importing domain)
      const violations = execSync(
        'grep -r "pkg/types" internal/storage/ internal/embeddings/ 2>/dev/null | wc -l', 
        { encoding: 'utf8' }
      ).trim();

      // Check circular dependencies
      const circularDeps = execSync(
        'go mod graph 2>/dev/null | grep -E "mcp-memory.*->.*mcp-memory" | wc -l', 
        { encoding: 'utf8' }
      ).trim();

      return {
        layerViolations: parseInt(violations) || 0,
        circularDependencies: parseInt(circularDeps) || 0,
        compliance: (parseInt(violations) === 0 && parseInt(circularDeps) === 0) ? 'Excellent' : 'Needs Improvement'
      };
    } catch (error) {
      return { error: 'Could not check layer compliance: ' + error.message };
    }
  }

  countGoFiles(dir) {
    try {
      const count = execSync(`find ${dir} -name "*.go" | wc -l`, { encoding: 'utf8' });
      return parseInt(count.trim()) || 0;
    } catch (error) {
      return 0;
    }
  }

  countLOC(dir) {
    try {
      const loc = execSync(`find ${dir} -name "*.go" | xargs wc -l | tail -1`, { encoding: 'utf8' });
      return parseInt(loc.trim().split(' ')[0]) || 0;
    } catch (error) {
      return 0;
    }
  }

  hasTests(dir) {
    try {
      const testFiles = execSync(`find ${dir} -name "*_test.go" | wc -l`, { encoding: 'utf8' });
      return parseInt(testFiles.trim()) > 0;
    } catch (error) {
      return false;
    }
  }

  getPackageDependencies(dir) {
    try {
      const imports = execSync(`grep -r "import" ${dir} | wc -l`, { encoding: 'utf8' });
      return parseInt(imports.trim()) || 0;
    } catch (error) {
      return 0;
    }
  }

  calculateComplexity(dir) {
    try {
      const functions = execSync(`grep -r "^func " ${dir} | wc -l`, { encoding: 'utf8' });
      const ifStatements = execSync(`grep -r "if " ${dir} | wc -l`, { encoding: 'utf8' });
      const complexity = (parseInt(ifStatements.trim()) || 0) / Math.max(parseInt(functions.trim()) || 1, 1);
      return complexity > 5 ? 'High' : complexity > 3 ? 'Medium' : 'Low';
    } catch (error) {
      return 'Unknown';
    }
  }

  identifyGoFrameworks(deps) {
    const frameworks = [];
    const known = {
      'github.com/gorilla/mux': 'Gorilla Mux (HTTP routing)',
      'github.com/gorilla/websocket': 'Gorilla WebSocket',
      'github.com/qdrant/go-client': 'Qdrant Vector Database',
      'github.com/sashabaranov/go-openai': 'OpenAI API Client',
      'github.com/fredcamaral/gomcp-sdk': 'MCP Protocol SDK',
      'github.com/stretchr/testify': 'Testify Testing Framework'
    };

    deps.forEach(dep => {
      const depName = dep.trim().split(' ')[0];
      if (known[depName]) {
        frameworks.push(known[depName]);
      }
    });

    return frameworks;
  }

  calculateHealthScore(analysis) {
    let score = 100;

    // Test coverage penalty
    const componentsWithTests = Object.values(analysis.components).filter(c => c.hasTests).length;
    const totalComponents = Object.keys(analysis.components).length;
    if (totalComponents > 0) {
      const testCoverage = componentsWithTests / totalComponents;
      score -= (1 - testCoverage) * 30; // Up to -30 for no tests
    }

    // Complexity penalty
    if (analysis.codeMetrics.avgFunctionSize > 50) {
      score -= 20; // Large functions indicate complexity
    } else if (analysis.codeMetrics.avgFunctionSize > 25) {
      score -= 10;
    }

    // Dependencies penalty
    if (analysis.dependencies.go > 30) {
      score -= 15; // Many dependencies increase maintenance burden
    } else if (analysis.dependencies.go > 20) {
      score -= 5;
    }

    // Layer compliance bonus/penalty
    if (analysis.layerCompliance?.compliance === 'Excellent') {
      score += 10; // Bonus for good architecture
    } else {
      score -= 20; // Penalty for architecture violations
    }

    // Architecture style bonus
    if (analysis.architecture?.confidence > 0.9) {
      score += 5; // Bonus for clear architecture
    }

    return Math.max(0, Math.round(score));
  }
}

// Health monitoring functionality
class HealthMonitor {
  constructor() {
    this.monitor = new ArchitectureMonitor();
  }

  async checkSystemHealth() {
    console.log('🏥 Checking MCP Memory Server health...');
    
    const health = {
      timestamp: new Date().toISOString(),
      components: await this.checkComponentHealth(),
      services: await this.checkServiceHealth(),
      dependencies: await this.checkDependencyHealth(),
      overall: 'unknown'
    };

    // Calculate overall health
    const componentHealthy = Object.values(health.components).every(status => status === 'healthy');
    const servicesHealthy = Object.values(health.services).every(status => status === 'healthy');
    
    if (componentHealthy && servicesHealthy) {
      health.overall = 'healthy';
    } else if (componentHealthy || servicesHealthy) {
      health.overall = 'degraded';
    } else {
      health.overall = 'unhealthy';
    }

    console.log(`🎯 Overall system health: ${health.overall.toUpperCase()}`);
    return health;
  }

  async checkComponentHealth() {
    const components = {};
    const criticalPaths = ['internal/mcp', 'internal/storage', 'internal/intelligence', 'cmd/server'];
    
    for (const path of criticalPaths) {
      try {
        if (fs.existsSync(path)) {
          const hasGoFiles = execSync(`find ${path} -name "*.go" | head -1`, { encoding: 'utf8' }).trim();
          components[path] = hasGoFiles ? 'healthy' : 'warning';
        } else {
          components[path] = 'missing';
        }
      } catch (error) {
        components[path] = 'error';
      }
    }
    
    return components;
  }

  async checkServiceHealth() {
    const services = {};
    
    // Check if project builds
    try {
      execSync('go build -o /tmp/mcp-test ./cmd/server', { stdio: 'pipe' });
      services.build = 'healthy';
    } catch (error) {
      services.build = 'error';
    }

    // Check if tests pass
    try {
      execSync('go test -short ./...', { stdio: 'pipe' });
      services.tests = 'healthy';
    } catch (error) {
      services.tests = 'warning';
    }

    return services;
  }

  async checkDependencyHealth() {
    try {
      execSync('go mod verify', { stdio: 'pipe' });
      return { goMod: 'healthy' };
    } catch (error) {
      return { goMod: 'error' };
    }
  }
}

// Main execution
if (require.main === module) {
  const monitor = new ArchitectureMonitor();
  const healthMonitor = new HealthMonitor();
  
  const command = process.argv[2];
  
  if (command === 'health') {
    healthMonitor.checkSystemHealth()
      .then(health => {
        console.log('\n📋 Health Summary:');
        console.log(`   Components: ${Object.keys(health.components).length} checked`);
        console.log(`   Services: ${Object.keys(health.services).length} checked`);
        console.log(`   Status: ${health.overall.toUpperCase()}`);
      })
      .catch(console.error);
  } else {
    monitor.analyzeArchitecture()
      .then(result => {
        console.log(`\n🏆 Architecture health score: ${result.healthScore}/100`);
        console.log(`📦 Components analyzed: ${Object.keys(result.components).length}`);
        console.log(`🔗 Dependencies: ${result.dependencies.go} external`);
        console.log(`🏗️  Architecture: ${result.architecture.style}`);
        console.log(`✅ Layer compliance: ${result.layerCompliance?.compliance || 'Unknown'}`);
      })
      .catch(console.error);
  }
}

module.exports = { ArchitectureMonitor, HealthMonitor };