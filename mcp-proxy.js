#!/usr/bin/env node

const http = require('http');
const readline = require('readline');
const fs = require('fs');
const path = require('path');

// Configuration
const CONFIG = {
  server: {
    hostname: process.env.MCP_SERVER_HOST || 'localhost',
    port: parseInt(process.env.MCP_SERVER_PORT || '9080'),
    path: process.env.MCP_SERVER_PATH || '/mcp',
    timeout: 30000,  // 30 second timeout
    maxRetries: 3,
    retryDelay: 1000  // 1 second between retries
  },
  logging: {
    enabled: process.env.MCP_PROXY_DEBUG === 'true',
    logFile: path.join(__dirname, 'mcp-proxy.log')
  },
  limits: {
    maxLineLength: 1024 * 1024,  // 1MB max line length
    maxResponseSize: 10 * 1024 * 1024  // 10MB max response
  }
};

// Logging utility
function log(level, message, data = null) {
  if (!CONFIG.logging.enabled) return;
  
  const timestamp = new Date().toISOString();
  const logEntry = {
    timestamp,
    level,
    message,
    data: data ? JSON.stringify(data, null, 2) : null
  };
  
  try {
    fs.appendFileSync(CONFIG.logFile, JSON.stringify(logEntry) + '\n');
  } catch (err) {
    // Silent fail on logging errors
  }
}

// Validate JSON-RPC request
function validateRequest(request) {
  if (typeof request !== 'object' || request === null) {
    throw new Error('Request must be an object');
  }
  
  if (request.jsonrpc !== '2.0') {
    throw new Error('Invalid JSON-RPC version');
  }
  
  if (typeof request.method !== 'string' || request.method.length === 0) {
    throw new Error('Method must be a non-empty string');
  }
  
  if (request.id !== undefined && 
      typeof request.id !== 'string' && 
      typeof request.id !== 'number' && 
      request.id !== null) {
    throw new Error('ID must be string, number, or null');
  }
  
  return true;
}

// Create error response
function createErrorResponse(code, message, id = null, data = null) {
  const error = {
    jsonrpc: '2.0',
    error: {
      code,
      message
    },
    id
  };
  
  if (data) {
    error.error.data = data;
  }
  
  return error;
}

// Health check function
async function checkServerHealth() {
  return new Promise((resolve) => {
    const healthOptions = {
      hostname: CONFIG.server.hostname,
      port: CONFIG.server.port,
      path: '/health',
      method: 'GET',
      timeout: 5000
    };

    const req = http.request(healthOptions, (res) => {
      res.on('data', () => {}); // Consume response
      res.on('end', () => {
        resolve(res.statusCode === 200);
      });
    });

    req.on('error', () => resolve(false));
    req.on('timeout', () => {
      req.destroy();
      resolve(false);
    });

    req.end();
  });
}

// Send HTTP request with retry logic
function sendHttpRequest(request, retryCount = 0) {
  return new Promise((resolve, reject) => {
    const postData = JSON.stringify(request);
    
    // Validate request size
    if (Buffer.byteLength(postData) > CONFIG.limits.maxResponseSize) {
      return reject(new Error('Request too large'));
    }
    
    const options = {
      hostname: CONFIG.server.hostname,
      port: CONFIG.server.port,
      path: CONFIG.server.path,
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Content-Length': Buffer.byteLength(postData),
        'User-Agent': 'mcp-proxy/VERSION_PLACEHOLDER'
      },
      timeout: CONFIG.server.timeout
    };

    log('debug', 'Sending HTTP request', { options, retryCount });

    const req = http.request(options, (res) => {
      let data = '';
      let totalSize = 0;

      res.on('data', (chunk) => {
        totalSize += chunk.length;
        
        // Prevent memory exhaustion
        if (totalSize > CONFIG.limits.maxResponseSize) {
          req.destroy();
          return reject(new Error('Response too large'));
        }
        
        data += chunk;
      });

      res.on('end', () => {
        log('debug', 'HTTP response received', { 
          statusCode: res.statusCode, 
          dataLength: data.length 
        });
        
        if (res.statusCode !== 200) {
          return reject(new Error(`HTTP ${res.statusCode}: ${data}`));
        }
        
        try {
          const response = JSON.parse(data);
          resolve(response);
        } catch (parseErr) {
          log('error', 'Failed to parse HTTP response', { data, error: parseErr.message });
          reject(new Error('Invalid JSON response from server'));
        }
      });
    });

    req.on('error', async (err) => {
      log('error', 'HTTP request error', { error: err.message, retryCount });
      
      // Retry logic for network errors
      if (retryCount < CONFIG.server.maxRetries && 
          (err.code === 'ECONNREFUSED' || err.code === 'ETIMEDOUT' || err.code === 'ENOTFOUND')) {
        
        // Check server health before retrying
        const isHealthy = await checkServerHealth();
        log('debug', 'Server health check', { isHealthy, retryCount });
        
        setTimeout(() => {
          sendHttpRequest(request, retryCount + 1)
            .then(resolve)
            .catch(reject);
        }, CONFIG.server.retryDelay * (retryCount + 1));
        
        return;
      }
      
      reject(err);
    });

    req.on('timeout', () => {
      req.destroy();
      reject(new Error('Request timeout'));
    });

    try {
      req.write(postData);
      req.end();
    } catch (writeErr) {
      reject(writeErr);
    }
  });
}

// Process incoming line
async function processLine(line) {
  let request = null;
  let requestId = null;
  
  try {
    // Validate line length
    if (line.length > CONFIG.limits.maxLineLength) {
      throw new Error('Line too long');
    }
    
    // Skip empty lines
    if (!line.trim()) {
      return;
    }
    
    log('debug', 'Processing line', { line });
    
    // Parse JSON-RPC request
    try {
      request = JSON.parse(line);
      requestId = request.id;
    } catch (parseErr) {
      throw new Error(`JSON parse error: ${parseErr.message}`);
    }
    
    // Validate request
    validateRequest(request);
    
    // Send to MCP server
    const response = await sendHttpRequest(request);
    
    // Validate response
    if (typeof response !== 'object' || response === null) {
      throw new Error('Invalid response from server');
    }
    
    // Ensure response has correct ID
    if (response.id === undefined && requestId !== undefined) {
      response.id = requestId;
    }
    
    // Output response
    console.log(JSON.stringify(response));
    log('debug', 'Response sent', { response });
    
  } catch (err) {
    log('error', 'Error processing line', { 
      line, 
      error: err.message, 
      stack: err.stack 
    });
    
    // Send appropriate error response
    let errorCode = -32603; // Internal error
    let errorMessage = err.message;
    
    if (err.message.includes('JSON parse error')) {
      errorCode = -32700; // Parse error
      errorMessage = 'Parse error';
    } else if (err.message.includes('Invalid JSON-RPC') || err.message.includes('Method must be')) {
      errorCode = -32600; // Invalid request
      errorMessage = 'Invalid Request';
    } else if (err.message.includes('Method not found')) {
      errorCode = -32601; // Method not found
    } else if (err.message.includes('Invalid params')) {
      errorCode = -32602; // Invalid params
    }
    
    const errorResponse = createErrorResponse(errorCode, errorMessage, requestId);
    console.log(JSON.stringify(errorResponse)); // Send errors to stdout for Claude to see
  }
}

// Set up readline interface with error handling
const rl = readline.createInterface({
  input: process.stdin,
  output: process.stdout,
  terminal: false,
  crlfDelay: Infinity
});

// Handle readline events
rl.on('line', (line) => {
  processLine(line).catch((err) => {
    log('error', 'Unhandled error in processLine', { error: err.message, stack: err.stack });
  });
});

rl.on('close', () => {
  log('info', 'MCP proxy closing');
  process.exit(0);
});

rl.on('error', (err) => {
  log('error', 'Readline error', { error: err.message });
  
  // Create a new readline interface if the current one fails
  setTimeout(() => {
    log('info', 'Attempting to restart readline interface');
    const newRl = readline.createInterface({
      input: process.stdin,
      output: process.stdout,
      terminal: false,
      crlfDelay: Infinity
    });
    
    newRl.on('line', processLine);
  }, 1000);
});

// Handle process-level errors
process.on('uncaughtException', (err) => {
  log('error', 'Uncaught exception', { error: err.message, stack: err.stack });
  
  // Try to send error response before exiting
  const errorResponse = createErrorResponse(-32603, 'Internal error', null);
  console.log(JSON.stringify(errorResponse)); // Send to stdout
  
  // Don't exit immediately, let the process continue
});

process.on('unhandledRejection', (reason, promise) => {
  log('error', 'Unhandled rejection', { reason: String(reason) });
  
  // Try to send error response
  const errorResponse = createErrorResponse(-32603, 'Internal error', null);
  console.log(JSON.stringify(errorResponse)); // Send to stdout
});

// Graceful shutdown
process.on('SIGINT', () => {
  log('info', 'Received SIGINT, shutting down gracefully');
  rl.close();
});

process.on('SIGTERM', () => {
  log('info', 'Received SIGTERM, shutting down gracefully');
  rl.close();
});

// Initial health check and startup
(async () => {
  log('info', 'MCP proxy starting', { 
    config: CONFIG,
    pid: process.pid,
    nodeVersion: process.version 
  });
  
  // Perform initial health check
  const isHealthy = await checkServerHealth();
  log('info', 'Initial server health check', { 
    isHealthy,
    endpoint: `${CONFIG.server.hostname}:${CONFIG.server.port}/health`
  });
  
  if (!isHealthy) {
    log('warn', 'Server not responding to health check - will retry on requests');
  }
  
  log('info', 'MCP proxy ready');
})();