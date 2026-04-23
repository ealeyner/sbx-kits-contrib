#!/usr/bin/env node
// anthropic-bridge.js — Routes Anthropic API requests through the sandbox proxy.
//
// Node.js's built-in fetch() and many HTTP libraries do not respect
// HTTP_PROXY / HTTPS_PROXY environment variables. This bridge explicitly
// tunnels HTTPS requests to api.anthropic.com through the sandbox proxy
// using HTTP CONNECT tunneling.
//
// Usage: Set ANTHROPIC_BASE_URL=http://127.0.0.1:54321 so the Anthropic
// SDK sends requests here instead of directly to api.anthropic.com.
//
// Based on the bridge pattern from:
// https://www.docker.com/blog/run-openclaw-securely-in-docker-sandboxes/

const http = require('http');
const https = require('https');

// Parse proxy host/port from HTTP_PROXY env var (set by sandbox runtime),
// falling back to PROXY_HOST/PROXY_PORT or defaults.
const httpProxyUrl = process.env.HTTP_PROXY || process.env.http_proxy || '';
let PROXY_HOST = process.env.PROXY_HOST || 'host.docker.internal';
let PROXY_PORT = parseInt(process.env.PROXY_PORT || '3128', 10);
if (httpProxyUrl) {
  try {
    const url = new URL(httpProxyUrl);
    PROXY_HOST = url.hostname;
    PROXY_PORT = parseInt(url.port || '3128', 10);
  } catch (_) {
    // fall back to defaults
  }
}
const TARGET_HOST = process.env.BRIDGE_TARGET || 'api.anthropic.com';
const TARGET_PORT = 443;
const BRIDGE_PORT = parseInt(process.env.BRIDGE_PORT || '54321', 10);

// Create an HTTP CONNECT tunnel through the sandbox proxy to the target.
function createTunnel() {
  return new Promise((resolve, reject) => {
    const req = http.request({
      host: PROXY_HOST,
      port: PROXY_PORT,
      method: 'CONNECT',
      path: `${TARGET_HOST}:${TARGET_PORT}`,
    });
    req.on('connect', (res, socket) => {
      if (res.statusCode === 200) resolve(socket);
      else reject(new Error(`CONNECT tunnel failed with status ${res.statusCode}`));
    });
    req.on('error', reject);
    req.end();
  });
}

const server = http.createServer(async (req, res) => {
  try {
    const socket = await createTunnel();

    // Rewrite headers for the real target
    const headers = { ...req.headers, host: TARGET_HOST };
    delete headers['connection'];
    delete headers['keep-alive'];

    const proxyReq = https.request(
      {
        socket,
        hostname: TARGET_HOST,
        port: TARGET_PORT,
        path: req.url,
        method: req.method,
        headers,
      },
      (proxyRes) => {
        res.writeHead(proxyRes.statusCode, proxyRes.headers);
        proxyRes.pipe(res);
      }
    );

    proxyReq.on('error', (e) => {
      if (!res.headersSent) {
        res.writeHead(502);
        res.end(JSON.stringify({ error: 'bridge_error', message: e.message }));
      }
    });

    req.pipe(proxyReq);
  } catch (e) {
    res.writeHead(502);
    res.end(JSON.stringify({ error: 'tunnel_error', message: e.message }));
  }
});

server.listen(BRIDGE_PORT, '127.0.0.1', () => {
  console.log(
    `Anthropic API bridge: 127.0.0.1:${BRIDGE_PORT} -> ${TARGET_HOST}:${TARGET_PORT} via ${PROXY_HOST}:${PROXY_PORT}`
  );
});
