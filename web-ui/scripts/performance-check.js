#!/usr/bin/env node

const fs = require('fs')
const path = require('path')
const { execSync } = require('child_process')

/**
 * Performance optimization check script
 * Analyzes bundle size, dependencies, and provides optimization recommendations
 */

const MAX_BUNDLE_SIZE = 500 * 1024 // 500KB
const MAX_CHUNK_SIZE = 250 * 1024 // 250KB
const MAX_DEPENDENCY_COUNT = 100

console.log('🚀 Running performance optimization check...\n')

// Check bundle size
function checkBundleSize() {
  console.log('📦 Checking bundle size...')
  
  try {
    const buildPath = path.join(__dirname, '../.next')
    if (!fs.existsSync(buildPath)) {
      console.log('⚠️  Build not found. Run "npm run build" first.')
      return
    }

    const staticPath = path.join(buildPath, 'static/chunks')
    if (!fs.existsSync(staticPath)) {
      console.log('⚠️  Static chunks not found.')
      return
    }

    const chunks = fs.readdirSync(staticPath)
      .filter(file => file.endsWith('.js'))
      .map(file => {
        const filePath = path.join(staticPath, file)
        const stats = fs.statSync(filePath)
        return {
          name: file,
          size: stats.size,
          sizeKB: Math.round(stats.size / 1024 * 100) / 100
        }
      })
      .sort((a, b) => b.size - a.size)

    const totalSize = chunks.reduce((sum, chunk) => sum + chunk.size, 0)
    const totalSizeKB = Math.round(totalSize / 1024 * 100) / 100

    console.log(`   Total bundle size: ${totalSizeKB} KB`)
    
    if (totalSize > MAX_BUNDLE_SIZE) {
      console.log(`   ❌ Bundle size exceeds ${MAX_BUNDLE_SIZE / 1024} KB limit`)
    } else {
      console.log(`   ✅ Bundle size within limits`)
    }

    console.log('\n   Largest chunks:')
    chunks.slice(0, 5).forEach(chunk => {
      const status = chunk.size > MAX_CHUNK_SIZE ? '❌' : '✅'
      console.log(`   ${status} ${chunk.name}: ${chunk.sizeKB} KB`)
    })

    console.log()
  } catch (error) {
    console.log('❌ Error checking bundle size:', error.message)
  }
}

// Check dependencies
function checkDependencies() {
  console.log('📚 Checking dependencies...')
  
  try {
    const packageJson = JSON.parse(fs.readFileSync(path.join(__dirname, '../package.json'), 'utf8'))
    const dependencies = Object.keys(packageJson.dependencies || {})
    const devDependencies = Object.keys(packageJson.devDependencies || {})
    
    console.log(`   Production dependencies: ${dependencies.length}`)
    console.log(`   Development dependencies: ${devDependencies.length}`)
    console.log(`   Total dependencies: ${dependencies.length + devDependencies.length}`)
    
    if (dependencies.length > MAX_DEPENDENCY_COUNT) {
      console.log(`   ⚠️  High number of production dependencies (${dependencies.length})`)
    } else {
      console.log(`   ✅ Dependency count reasonable`)
    }

    // Check for potential optimizations
    const heavyDependencies = [
      'lodash',
      'moment',
      'axios',
      'react-dom',
      '@apollo/client'
    ]

    const foundHeavy = dependencies.filter(dep => heavyDependencies.includes(dep))
    if (foundHeavy.length > 0) {
      console.log('\n   📋 Heavy dependencies found:')
      foundHeavy.forEach(dep => {
        console.log(`   ⚠️  ${dep} - consider lightweight alternatives`)
      })
    }

    console.log()
  } catch (error) {
    console.log('❌ Error checking dependencies:', error.message)
  }
}

// Check for unused dependencies
function checkUnusedDependencies() {
  console.log('🔍 Checking for unused dependencies...')
  
  try {
    // This is a simplified check - in production you'd use tools like depcheck
    const packageJson = JSON.parse(fs.readFileSync(path.join(__dirname, '../package.json'), 'utf8'))
    const dependencies = Object.keys(packageJson.dependencies || {})
    
    // Simple heuristic: check if dependency is imported in any .ts/.tsx files
    const srcPath = path.join(__dirname, '../')
    const usedDeps = new Set()
    
    function scanDirectory(dir) {
      if (!fs.existsSync(dir)) return
      
      const items = fs.readdirSync(dir)
      items.forEach(item => {
        const itemPath = path.join(dir, item)
        const stat = fs.statSync(itemPath)
        
        if (stat.isDirectory() && !item.startsWith('.') && item !== 'node_modules') {
          scanDirectory(itemPath)
        } else if (item.endsWith('.ts') || item.endsWith('.tsx') || item.endsWith('.js') || item.endsWith('.jsx')) {
          try {
            const content = fs.readFileSync(itemPath, 'utf8')
            dependencies.forEach(dep => {
              if (content.includes(`from '${dep}'`) || content.includes(`require('${dep}')`)) {
                usedDeps.add(dep)
              }
            })
          } catch (error) {
            // Ignore file read errors
          }
        }
      })
    }
    
    scanDirectory(srcPath)
    
    const unusedDeps = dependencies.filter(dep => !usedDeps.has(dep))
    
    if (unusedDeps.length > 0) {
      console.log('   ⚠️  Potentially unused dependencies:')
      unusedDeps.forEach(dep => {
        console.log(`   📦 ${dep}`)
      })
      console.log('\n   💡 Consider removing unused dependencies to reduce bundle size')
    } else {
      console.log('   ✅ No obviously unused dependencies found')
    }
    
    console.log()
  } catch (error) {
    console.log('❌ Error checking unused dependencies:', error.message)
  }
}

// Performance recommendations
function showRecommendations() {
  console.log('💡 Performance Optimization Recommendations:\n')
  
  const recommendations = [
    {
      category: 'Bundle Optimization',
      items: [
        'Use dynamic imports for heavy components',
        'Implement code splitting at route level',
        'Tree shake unused code',
        'Use webpack-bundle-analyzer to identify large dependencies'
      ]
    },
    {
      category: 'Image Optimization',
      items: [
        'Use Next.js Image component for automatic optimization',
        'Implement lazy loading for below-the-fold images',
        'Use WebP format with fallbacks',
        'Optimize images before uploading'
      ]
    },
    {
      category: 'Caching Strategy',
      items: [
        'Implement service worker for offline capabilities',
        'Use Apollo Client cache effectively',
        'Set up proper HTTP caching headers',
        'Use browser storage for user preferences'
      ]
    },
    {
      category: 'Runtime Performance',
      items: [
        'Memoize expensive computations with useMemo',
        'Use React.memo for component optimization',
        'Implement virtual scrolling for large lists',
        'Debounce user inputs and API calls'
      ]
    },
    {
      category: 'Monitoring',
      items: [
        'Set up Web Vitals monitoring',
        'Track bundle size in CI/CD',
        'Monitor runtime performance',
        'Set performance budgets'
      ]
    }
  ]
  
  recommendations.forEach(category => {
    console.log(`📊 ${category.category}:`)
    category.items.forEach(item => {
      console.log(`   • ${item}`)
    })
    console.log()
  })
}

// Generate performance report
function generateReport() {
  const reportPath = path.join(__dirname, '../performance-report.json')
  const report = {
    timestamp: new Date().toISOString(),
    bundles: {},
    dependencies: {},
    recommendations: [],
    tools: {
      'Bundle Analyzer': 'npm run analyze',
      'Lighthouse': 'npm run lighthouse',
      'Size Limit': 'npm run size-limit'
    }
  }
  
  fs.writeFileSync(reportPath, JSON.stringify(report, null, 2))
  console.log(`📄 Performance report generated: ${reportPath}`)
}

// Main execution
function main() {
  checkBundleSize()
  checkDependencies()
  checkUnusedDependencies()
  showRecommendations()
  generateReport()
  
  console.log('✨ Performance check complete!')
  console.log('\n🔧 Next steps:')
  console.log('   1. Run "npm run analyze" to see detailed bundle analysis')
  console.log('   2. Run "npm run lighthouse" for full performance audit')
  console.log('   3. Check performance-report.json for detailed findings')
}

if (require.main === module) {
  main()
}

module.exports = {
  checkBundleSize,
  checkDependencies,
  checkUnusedDependencies,
  showRecommendations,
  generateReport
}