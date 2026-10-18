// network_guard.cjs — best-effort network isolation for Browser-Code projects.
//
// The plugin loads this file with `node --require`. It blocks Node-side DNS,
// TCP and HTTP(S) access outside loopback. Chrome is restricted separately by
// HIMIND_CHROME_ARGS (host-resolver-rules).
//
// This is a guardrail, not a hard OS sandbox. A Style Pack must still be
// treated as trusted local code; hard isolation belongs in a dedicated
// renderer process/container.

const dns = require('node:dns');
const net = require('node:net');
const tls = require('node:tls');
const http = require('node:http');
const https = require('node:https');
const fs = require('node:fs');
const {URL} = require('node:url');

const mode = (process.env.HIMIND_NETWORK_MODE || 'none').toLowerCase();
const allowed = new Set(
  (process.env.HIMIND_ALLOWED_HOSTS || '')
    .split(',')
    .map((item) => item.trim().toLowerCase())
    .filter(Boolean),
);

function isLoopback(host) {
  const value = String(host || '').trim().toLowerCase().replace(/^\[|\]$/g, '');
  return value === '' || value === 'localhost' || value === '127.0.0.1' || value === '::1';
}

function isAllowed(host) {
  if (mode === 'full') return true;
  if (isLoopback(host)) return true;
  return mode === 'declared' && allowed.has(String(host || '').trim().toLowerCase());
}

function blocked(host) {
  const error = new Error(`HIMIND_NETWORK_BLOCKED: ${host || 'unknown host'} (mode=${mode})`);
  error.code = 'ERR_HIMIND_NETWORK_BLOCKED';
  try {
    const log = process.env.HIMIND_NETWORK_LOG;
    if (log) fs.appendFileSync(log, `${new Date().toISOString()} blocked ${host || 'unknown'} mode=${mode}\n`);
  } catch {}
  return error;
}

function hostFromOptions(options) {
  if (typeof options === 'string') {
    try { return new URL(options).hostname; } catch { return options; }
  }
  if (options instanceof URL) return options.hostname;
  if (!options || typeof options !== 'object') return '';
  return options.hostname || options.host || '';
}

function guardCallback(args, host) {
  const callback = args[args.length - 1];
  if (typeof callback === 'function') {
    process.nextTick(() => callback(blocked(host)));
    return true;
  }
  return false;
}

const originalLookup = dns.lookup;
dns.lookup = function guardedLookup(hostname, options, callback) {
  if (!isAllowed(hostname)) {
    if (!guardCallback([options, callback], hostname)) {
      throw blocked(hostname);
    }
    return;
  }
  return originalLookup.apply(this, arguments);
};

for (const name of ['resolve', 'resolve4', 'resolve6', 'resolveAny', 'resolveCname', 'resolveMx', 'resolveTxt']) {
  if (typeof dns[name] !== 'function') continue;
  const original = dns[name];
  dns[name] = function guardedResolve(hostname) {
    if (!isAllowed(hostname)) {
      guardCallback(Array.from(arguments), hostname);
      return;
    }
    return original.apply(this, arguments);
  };
}

if (dns.promises && typeof dns.promises.lookup === 'function') {
  const originalPromiseLookup = dns.promises.lookup;
  dns.promises.lookup = async function guardedPromiseLookup(hostname) {
    if (!isAllowed(hostname)) throw blocked(hostname);
    return originalPromiseLookup.apply(this, arguments);
  };
}

const originalConnect = net.connect;
const originalCreateConnection = net.createConnection;
function guardedConnect(options) {
  const host = hostFromOptions(options);
  if (!isAllowed(host)) throw blocked(host);
  return originalConnect.apply(this, arguments);
}
net.connect = guardedConnect;
net.createConnection = function guardedCreateConnection(options) {
  const host = hostFromOptions(options);
  if (!isAllowed(host)) throw blocked(host);
  return originalCreateConnection.apply(this, arguments);
};

if (typeof tls.connect === 'function') {
  const originalTlsConnect = tls.connect;
  tls.connect = function guardedTlsConnect(options) {
    const host = hostFromOptions(options);
    if (!isAllowed(host)) throw blocked(host);
    return originalTlsConnect.apply(this, arguments);
  };
}

function guardRequest(module, original) {
  module.request = function guardedRequest(options) {
    const host = hostFromOptions(options);
    if (!isAllowed(host)) {
      const callback = typeof arguments[arguments.length - 1] === 'function' ? arguments[arguments.length - 1] : null;
      const req = new (require('node:events').EventEmitter)();
      req.setTimeout = () => req;
      req.write = () => true;
      req.end = () => {
        if (callback) process.nextTick(() => callback(blocked(host)));
        process.nextTick(() => req.emit('error', blocked(host)));
      };
      req.abort = () => req;
      req.destroy = () => req;
      return req;
    }
    return original.apply(this, arguments);
  };
}
guardRequest(http, http.request);
guardRequest(https, https.request);

if (typeof globalThis.fetch === 'function') {
  const originalFetch = globalThis.fetch;
  globalThis.fetch = function guardedFetch(input) {
    const url = typeof input === 'string' || input instanceof URL ? String(input) : input && input.url;
    let host = '';
    try { host = new URL(url).hostname; } catch {}
    if (!isAllowed(host)) return Promise.reject(blocked(host));
    return originalFetch.apply(this, arguments);
  };
}

if (typeof globalThis.WebSocket === 'function') {
  const OriginalWebSocket = globalThis.WebSocket;
  globalThis.WebSocket = function GuardedWebSocket(url) {
    let host = '';
    try { host = new URL(url).hostname; } catch {}
    if (!isAllowed(host)) throw blocked(host);
    return new OriginalWebSocket(url);
  };
  globalThis.WebSocket.prototype = OriginalWebSocket.prototype;
}
