#!/usr/bin/env node

const http = require('http');
const readline = require('readline');

const rl = readline.createInterface({
  input: process.stdin,
  output: process.stdout,
  terminal: false
});

rl.on('line', (line) => {
  try {
    const request = JSON.parse(line);
    
    const postData = JSON.stringify(request);
    const options = {
      hostname: 'localhost',
      port: 9080,
      path: '/mcp',
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Content-Length': Buffer.byteLength(postData)
      }
    };

    const req = http.request(options, (res) => {
      let data = '';
      res.on('data', (chunk) => {
        data += chunk;
      });
      res.on('end', () => {
        console.log(data);
      });
    });

    req.on('error', (err) => {
      console.error(JSON.stringify({
        error: {
          code: -1,
          message: err.message
        },
        id: request.id
      }));
    });

    req.write(postData);
    req.end();
  } catch (err) {
    console.error(JSON.stringify({
      error: {
        code: -32700,
        message: 'Parse error'
      }
    }));
  }
});