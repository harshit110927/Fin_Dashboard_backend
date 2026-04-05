#!/usr/bin/env node
/*
  End-to-end API test script for Finance Dashboard backend.
  Requires Node.js 18+ (global fetch).

  Usage:
    node scripts/e2e-test.js

  Override base URL:
    BASE_URL=http://localhost:8080/api/v1 node scripts/e2e-test.js

  Debug flags:
    DEBUG=1         — print every request + response body
    VERBOSE=1       — print step entry/exit with timing
    NO_COLOR=1      — disable ANSI color output
    STOP_ON_FAIL=1  — halt immediately on first failure instead of continuing

  Optional env vars:
    ADMIN_EMAIL       (default: admin@fin.local)
    ADMIN_PASSWORD    (default: Admin@12345)
    ANALYST_EMAIL     (default: analyst@fin.local)
    ANALYST_PASSWORD  (default: Analyst@12345)
    VIEWER_EMAIL      (default: viewer@fin.local)
    VIEWER_PASSWORD   (default: Viewer@12345)

  Test categories covered:
    AUTH        — register, login, refresh, logout, validation, negative cases
    RBAC        — role-based access control for every protected route
    USERS       — full admin user management flow
    RECORDS     — CRUD, filters, pagination, immutability, void, soft delete
    DASHBOARD   — summary, trends, categories, recent activity, limits
    CATEGORIES  — list, create, update, role enforcement
    VALIDATION  — bad input, missing fields, wrong types
    NEGATIVE    — already-voided, inactive user, garbage tokens, etc.
*/

// ─── ANSI colour helpers ──────────────────────────────────────────────────────
const NO_COLOR = Boolean(process.env.NO_COLOR);
const c   = (code) => NO_COLOR ? (s) => s : (s) => `\x1b[${code}m${s}\x1b[0m`;
const dim    = c('2');
const bold   = c('1');
const red    = c('31');
const green  = c('32');
const yellow = c('33');
const cyan   = c('36');
const grey   = c('90');
const blue   = c('34');
const magenta = c('35');

// ─── Config ───────────────────────────────────────────────────────────────────
const BASE_URL     = process.env.BASE_URL     || 'http://localhost:8080/api/v1';
const DEBUG        = Boolean(process.env.DEBUG);
const VERBOSE      = Boolean(process.env.VERBOSE);
const STOP_ON_FAIL = Boolean(process.env.STOP_ON_FAIL);

const cfg = {
  admin:   { email: process.env.ADMIN_EMAIL    || 'admin@fin.local',   password: process.env.ADMIN_PASSWORD    || 'Admin@12345'   },
  analyst: { email: process.env.ANALYST_EMAIL  || 'analyst@fin.local', password: process.env.ANALYST_PASSWORD  || 'Analyst@12345' },
  viewer:  { email: process.env.VIEWER_EMAIL   || 'viewer@fin.local',  password: process.env.VIEWER_PASSWORD   || 'Viewer@12345'  },
};

// ─── Global state ─────────────────────────────────────────────────────────────
const _requestLog  = [];
const _results     = [];   // { step, label, passed, error, durationMs }
let   _currentStep = 'init';
let   _globalStart = Date.now();
let   _passCount   = 0;
let   _failCount   = 0;

// ─── Call-site capture ────────────────────────────────────────────────────────
function callSite(depth = 2) {
  const err    = new Error();
  const frames = err.stack.split('\n').slice(1);
  for (let i = depth; i < frames.length; i++) {
    const frame = frames[i];
    if (frame.includes('callSite') || frame.includes('assert') ||
        frame.includes('request')  || frame.includes('runTest') ||
        frame.includes('login'))   continue;
    const match = frame.match(/\((.+?:\d+:\d+)\)/) || frame.match(/at (.+?:\d+:\d+)/);
    if (match) return match[1].replace(process.cwd() + '\\', '').replace(process.cwd() + '/', '');
  }
  return 'unknown';
}

// ─── Assertion ────────────────────────────────────────────────────────────────
function assert(condition, message) {
  if (!condition) {
    const site = callSite(1);
    throw Object.assign(new Error(message), { _site: site });
  }
}

// ─── Step tracker ─────────────────────────────────────────────────────────────
function beginStep(name) {
  _currentStep = name;
  if (VERBOSE) console.log(`\n${cyan('▶')} ${bold(name)}`);
}

// ─── Individual test runner ───────────────────────────────────────────────────
/**
 * Wraps a single test case. Catches errors, records result, continues unless STOP_ON_FAIL.
 * @param {string} label   — short description shown in output
 * @param {Function} fn    — async test body
 */
async function runTest(label, fn) {
  const t0 = Date.now();
  try {
    await fn();
    const ms = Date.now() - t0;
    _results.push({ step: _currentStep, label, passed: true, durationMs: ms });
    _passCount++;
    console.log(`  ${green('✓')}  ${label}  ${grey(`(${ms}ms)`)}`);
  } catch (err) {
    const ms = Date.now() - t0;
    _results.push({ step: _currentStep, label, passed: false, error: err, durationMs: ms });
    _failCount++;
    console.log(`  ${red('✗')}  ${label}  ${grey(`(${ms}ms)`)}`);
    console.log(`       ${yellow(err._site || '')}  ${red(err.message.split('\n')[0])}`);
    if (DEBUG) {
      const frames = (err.stack || '').split('\n').slice(1)
        .filter(f => !f.includes('node:internal') && !f.includes('node_modules'))
        .slice(0, 5);
      frames.forEach(f => console.log(dim('       ' + f.trim())));
    }
    if (STOP_ON_FAIL) {
      printSummary();
      process.exit(1);
    }
  }
}

// ─── Core request wrapper ─────────────────────────────────────────────────────
async function request(method, path, { token, body, expectedStatus } = {}) {
  const url  = `${BASE_URL}${path}`;
  const site = callSite(1);
  const t0   = Date.now();

  const headers = { 'Content-Type': 'application/json' };
  if (token) headers.Authorization = `Bearer ${token}`;

  const reqEntry = {
    step: _currentStep, site, method, path, url,
    body:           body   ? JSON.stringify(body) : null,
    token:          token  ? `${token.slice(0, 14)}…` : null,
    expectedStatus, status: null, responseBody: null,
    durationMs: null, error: null,
  };

  if (DEBUG) {
    console.log(grey(`\n     ┌─ ${method} ${path}  [${site}]`));
    if (body) console.log(grey(`     │  req  : ${JSON.stringify(body)}`));
  }

  let res, text, data;
  try {
    res  = await fetch(url, { method, headers, body: body ? JSON.stringify(body) : undefined });
    text = await res.text();
    try { data = text ? JSON.parse(text) : null; } catch { data = { raw: text }; }

    reqEntry.status       = res.status;
    reqEntry.responseBody = text.length > 2000 ? text.slice(0, 2000) + '…(truncated)' : text;
    reqEntry.durationMs   = Date.now() - t0;

    if (DEBUG) {
      const sc = res.status >= 400 ? red : green;
      console.log(grey(`     │  status: `) + sc(String(res.status)));
      console.log(grey(`     │  res   : ${reqEntry.responseBody}`));
      console.log(grey(`     │  time  : ${reqEntry.durationMs}ms`));
      console.log(grey(`     └─`));
    }

    if (typeof expectedStatus === 'number' && res.status !== expectedStatus) {
      reqEntry.error = `expected HTTP ${expectedStatus}, got ${res.status}`;
      _requestLog.push(reqEntry);
      throw Object.assign(
        new Error(
          `${method} ${path}\n` +
          `  expected : HTTP ${expectedStatus}\n` +
          `  got      : HTTP ${res.status}\n` +
          `  body     : ${reqEntry.responseBody}\n` +
          `  site     : ${site}`
        ),
        { _site: site, _reqEntry: reqEntry }
      );
    }

    _requestLog.push(reqEntry);
    return { status: res.status, data, raw: text };

  } catch (err) {
    reqEntry.durationMs = Date.now() - t0;
    if (!reqEntry.error) { reqEntry.error = err.message; _requestLog.push(reqEntry); }
    throw err;
  }
}

// ─── Login helper ─────────────────────────────────────────────────────────────
async function login(user) {
  const out = await request('POST', '/auth/login', {
    body: { email: user.email, password: user.password },
    expectedStatus: 200,
  });
  assert(out.data?.success === true,      'Login response success=false');
  assert(out.data?.data?.access_token,    'Missing access_token in login response');
  assert(out.data?.data?.refresh_token,   'Missing refresh_token in login response');
  assert(out.data?.data?.user?.id,        'Missing user.id in login response');
  assert(out.data?.data?.user?.role,      'Missing user.role in login response');
  return out.data.data;
}

// ─── Summary printer ──────────────────────────────────────────────────────────
function printSummary() {
  const dur     = Date.now() - _globalStart;
  const total   = _passCount + _failCount;
  const allPass = _failCount === 0;

  console.log('\n' + (allPass ? green : red)('━'.repeat(62)));
  console.log((allPass ? green : red)(bold('  TEST SUMMARY')));
  console.log((allPass ? green : red)('━'.repeat(62)));
  console.log(`  Total   : ${bold(String(total))}`);
  console.log(`  Passed  : ${green(bold(String(_passCount)))}`);
  console.log(`  Failed  : ${_failCount > 0 ? red(bold(String(_failCount))) : green('0')}`);
  console.log(`  Duration: ${dur}ms`);
  console.log(`  Requests: ${_requestLog.length}`);

  if (_failCount > 0) {
    console.log('\n' + red(bold('  Failed tests:')));
    _results
      .filter(r => !r.passed)
      .forEach((r, i) => {
        console.log(`\n  ${red(`${i + 1}.`)} ${bold(r.label)}`);
        console.log(`     Step   : ${r.step}`);
        console.log(`     Error  : ${red(r.error?.message?.split('\n')[0] || 'unknown')}`);
        if (r.error?._site) console.log(`     Site   : ${yellow(r.error._site)}`);
        // Show last request from this test
        const lastReq = _requestLog.slice().reverse().find(req => req.step === r.step);
        if (lastReq) {
          console.log(`     Request: ${lastReq.method} ${lastReq.path} → HTTP ${lastReq.status ?? '?'}`);
          if (lastReq.responseBody) console.log(`     Body   : ${grey(lastReq.responseBody.slice(0, 300))}`);
        }
      });
  }

  console.log('\n' + (allPass ? green : red)('━'.repeat(62)));
  if (allPass) {
    console.log(green(bold(`\n  🎉  All ${total} tests passed in ${dur}ms\n`)));
  } else {
    console.log(red(bold(`\n  ❌  ${_failCount} test(s) failed\n`)));
  }
}

// ─── Section header ───────────────────────────────────────────────────────────
function section(title) {
  console.log(`\n${blue('┌' + '─'.repeat(60))}`);
  console.log(`${blue('│')}  ${bold(title)}`);
  console.log(`${blue('└' + '─'.repeat(60))}`);
}

// ─── Main ─────────────────────────────────────────────────────────────────────
async function run() {
  console.log(`\n${bold(magenta('Finance Dashboard — Comprehensive E2E Test Suite'))}`);
  console.log(dim(`  Target  : ${BASE_URL}`));
  console.log(dim(`  Debug   : ${DEBUG ? 'ON' : 'off'}  |  Verbose: ${VERBOSE ? 'ON' : 'off'}  |  StopOnFail: ${STOP_ON_FAIL ? 'ON' : 'off'}`));
  console.log(dim(`  Users   : ${cfg.admin.email} / ${cfg.analyst.email} / ${cfg.viewer.email}\n`));

  // ═══════════════════════════════════════════════════════════════════════
  // SECTION 1 — AUTH: LOGIN
  // ═══════════════════════════════════════════════════════════════════════
  section('1 · AUTH — Login & Token flows');
  beginStep('1 · Auth — Login');

  // We need tokens to run all other tests, so login is non-negotiable
  // If this fails the entire suite is aborted
  let admin, analyst, viewer;
  try {
    admin   = await login(cfg.admin);
    analyst = await login(cfg.analyst);
    viewer  = await login(cfg.viewer);
    console.log(`  ${green('✓')}  Login passed for admin / analyst / viewer`);
  } catch (err) {
    console.log(red(bold('\n  FATAL: Cannot login as one or more users. Ensure seed data exists.')));
    console.log(red(`  ${err.message.split('\n')[0]}\n`));
    process.exit(1);
  }

  await runTest('Admin token has correct role=admin', async () => {
    assert(admin.user.role === 'admin', `Expected role=admin, got ${admin.user.role}`);
  });

  await runTest('Analyst token has correct role=analyst', async () => {
    assert(analyst.user.role === 'analyst', `Expected role=analyst, got ${analyst.user.role}`);
  });

  await runTest('Viewer token has correct role=viewer', async () => {
    assert(viewer.user.role === 'viewer', `Expected role=viewer, got ${viewer.user.role}`);
  });

  // ─── Refresh token ───────────────────────────────────────────────────
  beginStep('1 · Auth — Refresh token');

  await runTest('Refresh token returns new access_token', async () => {
    const res = await request('POST', '/auth/refresh', {
      body: { refresh_token: admin.refresh_token },
      expectedStatus: 200,
    });
    assert(res.data?.data?.access_token, 'Missing access_token in refresh response');
    // Update admin token to the freshest one
    admin.access_token = res.data.data.access_token;
  });

  await runTest('Invalid refresh token → 401', async () => {
    await request('POST', '/auth/refresh', {
      body: { refresh_token: 'this.is.a.garbage.refresh.token' },
      expectedStatus: 401,
    });
  });

  await runTest('Missing refresh token body → 400 or 401', async () => {
    const res = await request('POST', '/auth/refresh', {
      body: {},
    });
    assert(res.status === 400 || res.status === 401,
      `Expected 400 or 401, got ${res.status}`);
  });

  // ═══════════════════════════════════════════════════════════════════════
  // SECTION 2 — AUTH: VALIDATION & NEGATIVE CASES
  // ═══════════════════════════════════════════════════════════════════════
  section('2 · AUTH — Validation & Negative cases');
  beginStep('2 · Auth — Validation');

  await runTest('Register duplicate email → 400', async () => {
    await request('POST', '/auth/register', {
      body: { name: 'Duplicate Admin', email: cfg.admin.email, password: 'Admin@12345', role: 'viewer' },
      expectedStatus: 400,
    });
  });

  await runTest('Register with invalid email format → 400', async () => {
    await request('POST', '/auth/register', {
      body: { name: 'Bad Email', email: 'not-an-email-address', password: 'Admin@12345', role: 'viewer' },
      expectedStatus: 400,
    });
  });

  await runTest('Register with password too short (<8 chars) → 400', async () => {
    await request('POST', '/auth/register', {
      body: { name: 'Short Pass', email: `short-${Date.now()}@fin.local`, password: '123', role: 'viewer' },
      expectedStatus: 400,
    });
  });

  await runTest('Register with missing name → 400', async () => {
    await request('POST', '/auth/register', {
      body: { email: `noname-${Date.now()}@fin.local`, password: 'Admin@12345', role: 'viewer' },
      expectedStatus: 400,
    });
  });

  await runTest('Register with invalid role value → 400', async () => {
    await request('POST', '/auth/register', {
      body: { name: 'Bad Role', email: `badrole-${Date.now()}@fin.local`, password: 'Admin@12345', role: 'superadmin' },
      expectedStatus: 400,
    });
  });

  await runTest('Login with wrong password → 401', async () => {
    await request('POST', '/auth/login', {
      body: { email: cfg.admin.email, password: 'TotallyWrongPassword999' },
      expectedStatus: 401,
    });
  });

  await runTest('Login with non-existent email → 401', async () => {
    await request('POST', '/auth/login', {
      body: { email: `ghost-${Date.now()}@fin.local`, password: 'Admin@12345' },
      expectedStatus: 401,
    });
  });

  await runTest('Login with missing password → 400', async () => {
    await request('POST', '/auth/login', {
      body: { email: cfg.admin.email },
      expectedStatus: 400,
    });
  });

  await runTest('Protected route with NO token → 401', async () => {
    await request('GET', '/records', { expectedStatus: 401 });
  });

  await runTest('Protected route with garbage Bearer token → 401', async () => {
    await request('GET', '/records', {
      token: 'this.is.complete.garbage',
      expectedStatus: 401,
    });
  });

  await runTest('Protected route with malformed JWT (no signature) → 401', async () => {
    await request('GET', '/records', {
      token: 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiZmFrZSIsInJvbGUiOiJhZG1pbiJ9',
      expectedStatus: 401,
    });
  });

  // ═══════════════════════════════════════════════════════════════════════
  // SECTION 3 — RBAC: ROLE-BASED ACCESS CONTROL
  // ═══════════════════════════════════════════════════════════════════════
  section('3 · RBAC — Role-Based Access Control');
  beginStep('3 · RBAC — User routes');

  await runTest('Viewer cannot GET /users → 403', async () => {
    await request('GET', '/users', { token: viewer.access_token, expectedStatus: 403 });
  });

  await runTest('Analyst cannot GET /users → 403', async () => {
    await request('GET', '/users', { token: analyst.access_token, expectedStatus: 403 });
  });

  await runTest('Viewer cannot POST /users → 403', async () => {
    await request('POST', '/users', {
      token: viewer.access_token,
      body: { name: 'X', email: 'x@x.com', password: 'Admin@12345', role: 'viewer' },
      expectedStatus: 403,
    });
  });

  beginStep('3 · RBAC — Record routes');

  await runTest('Viewer cannot create record (POST /records) → 403', async () => {
    await request('POST', '/records', {
      token: viewer.access_token,
      body: { amount: 100, type: 'expense', category_id: 6, date: '2026-04-01', description: 'should fail' },
      expectedStatus: 403,
    });
  });

  await runTest('Analyst cannot create record (POST /records) → 403', async () => {
    await request('POST', '/records', {
      token: analyst.access_token,
      body: { amount: 100, type: 'expense', category_id: 6, date: '2026-04-01', description: 'should fail' },
      expectedStatus: 403,
    });
  });

  await runTest('Viewer can read records (GET /records) → 200', async () => {
    await request('GET', '/records', { token: viewer.access_token, expectedStatus: 200 });
  });

  await runTest('Analyst can read records (GET /records) → 200', async () => {
    await request('GET', '/records', { token: analyst.access_token, expectedStatus: 200 });
  });

  beginStep('3 · RBAC — Dashboard routes');

  await runTest('Viewer CANNOT access trends (GET /dashboard/trends) → 403', async () => {
    await request('GET', '/dashboard/trends', { token: viewer.access_token, expectedStatus: 403 });
  });

  await runTest('Analyst CAN access trends (GET /dashboard/trends) → 200', async () => {
    await request('GET', '/dashboard/trends', { token: analyst.access_token, expectedStatus: 200 });
  });

  await runTest('Viewer CAN access summary (GET /dashboard/summary) → 200', async () => {
    await request('GET', '/dashboard/summary', { token: viewer.access_token, expectedStatus: 200 });
  });

  await runTest('Viewer CAN access categories (GET /dashboard/categories) → 200', async () => {
    await request('GET', '/dashboard/categories', { token: viewer.access_token, expectedStatus: 200 });
  });

  await runTest('Viewer CAN access recent (GET /dashboard/recent) → 200', async () => {
    await request('GET', '/dashboard/recent', { token: viewer.access_token, expectedStatus: 200 });
  });

  beginStep('3 · RBAC — Category routes');

  await runTest('Viewer cannot create category (POST /categories) → 403', async () => {
    await request('POST', '/categories', {
      token: viewer.access_token,
      body: { name: 'BadCategory', type: 'expense' },
      expectedStatus: 403,
    });
  });

  await runTest('Analyst cannot create category (POST /categories) → 403', async () => {
    await request('POST', '/categories', {
      token: analyst.access_token,
      body: { name: 'BadCategory', type: 'expense' },
      expectedStatus: 403,
    });
  });

  // ═══════════════════════════════════════════════════════════════════════
  // SECTION 4 — CATEGORIES
  // ═══════════════════════════════════════════════════════════════════════
  section('4 · CATEGORIES');
  beginStep('4 · Categories — CRUD');

  await runTest('All roles can list categories → 200', async () => {
    const r1 = await request('GET', '/categories', { token: admin.access_token,   expectedStatus: 200 });
    const r2 = await request('GET', '/categories', { token: analyst.access_token, expectedStatus: 200 });
    const r3 = await request('GET', '/categories', { token: viewer.access_token,  expectedStatus: 200 });
    assert(Array.isArray(r1.data?.data), 'Categories not an array (admin)');
    assert(Array.isArray(r2.data?.data), 'Categories not an array (analyst)');
    assert(Array.isArray(r3.data?.data), 'Categories not an array (viewer)');
  });

  const unique = Date.now();
  let categoryID;

  await runTest('Admin creates category → 201 with id', async () => {
    const res = await request('POST', '/categories', {
      token: admin.access_token,
      body: { name: `E2E-Cat-${unique}`, type: 'expense' },
      expectedStatus: 201,
    });
    assert(res.data?.data?.id,   'Missing category id after create');
    assert(res.data?.data?.name, 'Missing category name after create');
    assert(res.data?.data?.type === 'expense', 'Category type mismatch');
    categoryID = res.data.data.id;
  });

  await runTest('Admin updates category name → 200', async () => {
    if (!categoryID) return;
    const res = await request('PATCH', `/categories/${categoryID}`, {
      token: admin.access_token,
      body: { name: `E2E-Cat-Updated-${unique}` },
      expectedStatus: 200,
    });
    assert(res.data?.success === true, 'Update category did not return success=true');
  });

  await runTest('Create category with missing name → 400', async () => {
    await request('POST', '/categories', {
      token: admin.access_token,
      body: { type: 'expense' },
      expectedStatus: 400,
    });
  });

  await runTest('Create category with invalid type → 400', async () => {
    await request('POST', '/categories', {
      token: admin.access_token,
      body: { name: `E2E-BadType-${unique}`, type: 'revenue' },
      expectedStatus: 400,
    });
  });

  await runTest('Create duplicate category name → 400 or 409', async () => {
    if (!categoryID) return;
    const res = await request('POST', '/categories', {
      token: admin.access_token,
      body: { name: `E2E-Cat-Updated-${unique}`, type: 'expense' },
    });
    assert(res.status === 400 || res.status === 409,
      `Expected 400 or 409 for duplicate category, got ${res.status}`);
  });

  // ═══════════════════════════════════════════════════════════════════════
  // SECTION 5 — FINANCIAL RECORDS: VALIDATION
  // ═══════════════════════════════════════════════════════════════════════
  section('5 · RECORDS — Validation');
  beginStep('5 · Records — Input validation');

  await runTest('Create record with amount=0 → 400', async () => {
    await request('POST', '/records', {
      token: admin.access_token,
      body: { amount: 0, type: 'expense', category_id: 6, date: '2026-04-01' },
      expectedStatus: 400,
    });
  });

  await runTest('Create record with negative amount → 400', async () => {
    await request('POST', '/records', {
      token: admin.access_token,
      body: { amount: -500, type: 'expense', category_id: 6, date: '2026-04-01' },
      expectedStatus: 400,
    });
  });

  await runTest('Create record with invalid type → 400', async () => {
    await request('POST', '/records', {
      token: admin.access_token,
      body: { amount: 100, type: 'invalid_type', category_id: 6, date: '2026-04-01' },
      expectedStatus: 400,
    });
  });

  await runTest('Create record with missing amount → 400', async () => {
    await request('POST', '/records', {
      token: admin.access_token,
      body: { type: 'expense', category_id: 6, date: '2026-04-01' },
      expectedStatus: 400,
    });
  });

  await runTest('Create record with missing date → 400', async () => {
    await request('POST', '/records', {
      token: admin.access_token,
      body: { amount: 100, type: 'expense', category_id: 6 },
      expectedStatus: 400,
    });
  });

  await runTest('Create record with non-existent category_id → 400 or 422', async () => {
    const res = await request('POST', '/records', {
      token: admin.access_token,
      body: { amount: 100, type: 'expense', category_id: 99999, date: '2026-04-01' },
    });
    assert(res.status === 400 || res.status === 422,
      `Expected 400 or 422 for bad category_id, got ${res.status}`);
  });

  // ═══════════════════════════════════════════════════════════════════════
  // SECTION 6 — FINANCIAL RECORDS: CRUD
  // ═══════════════════════════════════════════════════════════════════════
  section('6 · RECORDS — CRUD');
  beginStep('6 · Records — Create');

  let incomeRecordID, expenseRecordID, recordToDeleteID;

  await runTest('Admin creates income record → 201 with correct fields', async () => {
    const res = await request('POST', '/records', {
      token: admin.access_token,
      body: {
        amount: 75000.00,
        type: 'income',
        category_id: 1,             // Salary
        date: '2026-04-01',
        description: 'E2E April salary',
      },
      expectedStatus: 201,
    });
    assert(res.data?.data?.id,                   'Missing id');
    assert(res.data?.data?.amount === 75000.00,  'Amount mismatch');
    assert(res.data?.data?.type === 'income',    'Type mismatch');
    assert(res.data?.data?.status === 'active',  'Status should be active');
    assert(res.data?.data?.category_name,        'Missing category_name');
    assert(res.data?.data?.created_by_name,      'Missing created_by_name');
    incomeRecordID = res.data.data.id;
  });

  await runTest('Admin creates expense record → 201', async () => {
    if (!categoryID) return;
    const res = await request('POST', '/records', {
      token: admin.access_token,
      body: {
        amount: 999.55,
        type: 'expense',
        category_id: categoryID,
        date: '2026-04-02',
        description: 'E2E expense record',
      },
      expectedStatus: 201,
    });
    assert(res.data?.data?.id, 'Missing id');
    expenseRecordID = res.data.data.id;
  });

  await runTest('Admin creates record to be deleted → 201', async () => {
    const res = await request('POST', '/records', {
      token: admin.access_token,
      body: {
        amount: 123.45,
        type: 'expense',
        category_id: 4,             // Rent
        date: '2026-04-03',
        description: 'E2E record for soft-delete test',
      },
      expectedStatus: 201,
    });
    assert(res.data?.data?.id, 'Missing id');
    recordToDeleteID = res.data.data.id;
  });

  beginStep('6 · Records — Read');

  await runTest('GET /records returns paginated response with meta → 200', async () => {
    const res = await request('GET', '/records?page=1&per_page=10', {
      token: admin.access_token,
      expectedStatus: 200,
    });
    assert(res.data?.success === true,               'success=false');
    assert(Array.isArray(res.data?.data),            'data is not array');
    assert(typeof res.data?.meta?.page === 'number', 'missing meta.page');
    assert(typeof res.data?.meta?.total === 'number','missing meta.total');
    assert(typeof res.data?.meta?.per_page === 'number', 'missing meta.per_page');
  });

  await runTest('Viewer can GET /records → 200', async () => {
    const res = await request('GET', '/records', { token: viewer.access_token, expectedStatus: 200 });
    assert(Array.isArray(res.data?.data), 'data not array');
  });

  await runTest('Analyst can GET /records → 200', async () => {
    const res = await request('GET', '/records', { token: analyst.access_token, expectedStatus: 200 });
    assert(Array.isArray(res.data?.data), 'data not array');
  });

  await runTest('GET /records/:id returns full record → 200', async () => {
    if (!incomeRecordID) return;
    const res = await request('GET', `/records/${incomeRecordID}`, {
      token: admin.access_token,
      expectedStatus: 200,
    });
    assert(res.data?.data?.id === incomeRecordID, 'Returned wrong record');
    assert(res.data?.data?.category_name,         'Missing category_name on single record');
    assert(res.data?.data?.created_by_name,       'Missing created_by_name on single record');
  });

  await runTest('GET /records/:id with non-existent id → 404', async () => {
    await request('GET', '/records/00000000-0000-0000-0000-000000000000', {
      token: admin.access_token,
      expectedStatus: 404,
    });
  });

  beginStep('6 · Records — Filtering');

  await runTest('Filter by type=income returns only income records', async () => {
    const res = await request('GET', '/records?type=income', {
      token: admin.access_token,
      expectedStatus: 200,
    });
    const records = res.data?.data || [];
    assert(records.every(r => r.type === 'income'),
      `Found non-income records when filtering type=income`);
  });

  await runTest('Filter by type=expense returns only expense records', async () => {
    const res = await request('GET', '/records?type=expense', {
      token: admin.access_token,
      expectedStatus: 200,
    });
    const records = res.data?.data || [];
    assert(records.every(r => r.type === 'expense'),
      `Found non-expense records when filtering type=expense`);
  });

  await runTest('Filter by date_from and date_to returns records in range', async () => {
    const res = await request('GET', '/records?date_from=2026-04-01&date_to=2026-04-30', {
      token: admin.access_token,
      expectedStatus: 200,
    });
    const records = res.data?.data || [];
    // All returned records should have date in April 2026
    assert(records.every(r => r.date >= '2026-04-01' && r.date <= '2026-04-30'),
      'Records outside date range were returned');
  });

  await runTest('Filter by category_id returns only matching category', async () => {
    if (!categoryID) return;
    const res = await request('GET', `/records?category_id=${categoryID}`, {
      token: admin.access_token,
      expectedStatus: 200,
    });
    const records = res.data?.data || [];
    assert(records.every(r => r.category_id === categoryID),
      'Records with wrong category_id returned');
  });

  await runTest('Pagination: page=1 per_page=1 returns exactly 1 record', async () => {
    const res = await request('GET', '/records?page=1&per_page=1', {
      token: admin.access_token,
      expectedStatus: 200,
    });
    assert(res.data?.data?.length === 1,    'Expected exactly 1 record');
    assert(res.data?.meta?.per_page === 1,  'meta.per_page should be 1');
    assert(res.data?.meta?.page === 1,      'meta.page should be 1');
  });

  await runTest('Sort by amount_desc returns records in descending amount order', async () => {
    const res = await request('GET', '/records?sort=amount_desc&per_page=50', {
      token: admin.access_token,
      expectedStatus: 200,
    });
    const records = res.data?.data || [];
    for (let i = 1; i < records.length; i++) {
      assert(records[i - 1].amount >= records[i].amount,
        `Records not sorted by amount DESC at index ${i}`);
    }
  });

  beginStep('6 · Records — Update');

  await runTest('Admin updates record description and category → 200', async () => {
    if (!expenseRecordID || !categoryID) return;
    const res = await request('PATCH', `/records/${expenseRecordID}`, {
      token: admin.access_token,
      body: { description: 'E2E updated description', category_id: categoryID },
      expectedStatus: 200,
    });
    assert(res.data?.success === true, 'Update returned success=false');
  });

  await runTest('Attempt to update amount → 400 IMMUTABLE_FIELD', async () => {
    if (!expenseRecordID) return;
    const res = await request('PATCH', `/records/${expenseRecordID}`, {
      token: admin.access_token,
      body: { amount: 9999999.99 },
      expectedStatus: 400,
    });
    assert(
      res.data?.error?.code === 'IMMUTABLE_FIELD' || res.data?.error?.code?.includes('IMMUTABLE'),
      `Expected error code IMMUTABLE_FIELD, got: ${res.data?.error?.code}`
    );
  });

  await runTest('Attempt to update type → 400 IMMUTABLE_FIELD', async () => {
    if (!expenseRecordID) return;
    const res = await request('PATCH', `/records/${expenseRecordID}`, {
      token: admin.access_token,
      body: { type: 'income' },
      expectedStatus: 400,
    });
    assert(
      res.data?.error?.code === 'IMMUTABLE_FIELD' || res.data?.error?.code?.includes('IMMUTABLE'),
      `Expected error code IMMUTABLE_FIELD, got: ${res.data?.error?.code}`
    );
  });

  await runTest('Viewer cannot update record → 403', async () => {
    if (!expenseRecordID) return;
    await request('PATCH', `/records/${expenseRecordID}`, {
      token: viewer.access_token,
      body: { description: 'viewer tries to update' },
      expectedStatus: 403,
    });
  });

  beginStep('6 · Records — Void');

  await runTest('Admin voids a record with valid reason → 200, status=void', async () => {
    if (!expenseRecordID) return;
    const res = await request('POST', `/records/${expenseRecordID}/void`, {
      token: admin.access_token,
      body: { reason: 'Voiding this record as part of automated E2E test run' },
      expectedStatus: 200,
    });
    assert(res.data?.data?.status === 'void', `Expected status=void, got ${res.data?.data?.status}`);
  });

  await runTest('Void already-voided record → 400', async () => {
    if (!expenseRecordID) return;
    await request('POST', `/records/${expenseRecordID}/void`, {
      token: admin.access_token,
      body: { reason: 'Trying to void again, should be rejected by business logic' },
      expectedStatus: 400,
    });
  });

  await runTest('Void with reason too short (<10 chars) → 400', async () => {
    if (!incomeRecordID) return;
    await request('POST', `/records/${incomeRecordID}/void`, {
      token: admin.access_token,
      body: { reason: 'short' },
      expectedStatus: 400,
    });
  });

  await runTest('Viewer cannot void record → 403', async () => {
    if (!incomeRecordID) return;
    await request('POST', `/records/${incomeRecordID}/void`, {
      token: viewer.access_token,
      body: { reason: 'Viewer attempts void, should be rejected' },
      expectedStatus: 403,
    });
  });

  beginStep('6 · Records — Soft Delete');

  await runTest('Admin soft deletes a record → 200', async () => {
    if (!recordToDeleteID) return;
    const res = await request('DELETE', `/records/${recordToDeleteID}`, {
      token: admin.access_token,
      expectedStatus: 200,
    });
    assert(res.data?.success === true, 'Delete returned success=false');
  });

  await runTest('Soft-deleted record does NOT appear in list', async () => {
    if (!recordToDeleteID) return;
    const res = await request('GET', '/records?per_page=100', {
      token: admin.access_token,
      expectedStatus: 200,
    });
    const ids = (res.data?.data || []).map(r => r.id);
    assert(!ids.includes(recordToDeleteID),
      'Soft-deleted record still appears in GET /records list');
  });

  await runTest('Soft-deleted record returns 404 on GET /records/:id', async () => {
    if (!recordToDeleteID) return;
    await request('GET', `/records/${recordToDeleteID}`, {
      token: admin.access_token,
      expectedStatus: 404,
    });
  });

  await runTest('Viewer cannot delete record → 403', async () => {
    if (!incomeRecordID) return;
    await request('DELETE', `/records/${incomeRecordID}`, {
      token: viewer.access_token,
      expectedStatus: 403,
    });
  });


  // ─── Step 5.5 · New endpoints ─────────────────────────────────────────────
  beginStep('5.5 · Health check & record history');

  await runTest('GET /health returns 200 with db=ok', async () => {
    // Note: call without /api/v1 prefix
    const res = await fetch(`${BASE_URL.replace('/api/v1', '')}/health`);
    const data = await res.json();
    assert(res.status === 200,        `Expected 200, got ${res.status}`);
    assert(data.status === 'ok',      'Missing status=ok');
    assert(data.db === 'ok',          'DB status not ok');
    assert(data.timestamp,            'Missing timestamp');
    assert(data.request_id,           'Missing request_id in health response');
  });

  await runTest('GET /health returns X-Request-ID header', async () => {
    const res = await fetch(`${BASE_URL.replace('/api/v1', '')}/health`);
    assert(res.headers.get('x-request-id'), 'Missing X-Request-ID response header');
  });

  await runTest('GET /records/:id/history returns audit trail → 200 (admin)', async () => {
    if (!incomeRecordID) return;
    const res = await request('GET', `/records/${incomeRecordID}/history`, {
      token: admin.access_token,
      expectedStatus: 200,
    });
    assert(Array.isArray(res.data?.data), 'history data not array');
    if (res.data.data.length > 0) {
      const entry = res.data.data[0];
      assert(entry.action,     'Audit entry missing action');
      assert(entry.actor_id,   'Audit entry missing actor_id');
      assert(entry.created_at, 'Audit entry missing created_at');
    }
  });

  await runTest('GET /records/:id/history → 403 for viewer', async () => {
    if (!incomeRecordID) return;
    await request('GET', `/records/${incomeRecordID}/history`, {
      token: viewer.access_token,
      expectedStatus: 403,
    });
  });

  await runTest('GET /records/:id/history → 403 for analyst', async () => {
    if (!incomeRecordID) return;
    await request('GET', `/records/${incomeRecordID}/history`, {
      token: analyst.access_token,
      expectedStatus: 403,
    });
  });

  // ═══════════════════════════════════════════════════════════════════════
  // SECTION 7 — DASHBOARD
  // ═══════════════════════════════════════════════════════════════════════
  section('7 · DASHBOARD');
  beginStep('7 · Dashboard — Summary');

  const today   = new Date();
  const year    = today.getUTCFullYear();
  const month   = today.getUTCMonth();   // 0-indexed
  const from    = `${year}-${String(month + 1).padStart(2, '0')}-01`;
  const lastDay = new Date(Date.UTC(year, month + 1, 0)).getUTCDate();
  const to      = `${year}-${String(month + 1).padStart(2, '0')}-${String(lastDay).padStart(2, '0')}`;

  await runTest('Admin gets summary with date range → 200 with correct shape', async () => {
    const res = await request('GET', `/dashboard/summary?from=${from}&to=${to}`, {
      token: admin.access_token,
      expectedStatus: 200,
    });
    const d = res.data?.data;
    assert(typeof d?.total_income   === 'number', 'Missing total_income');
    assert(typeof d?.total_expenses === 'number', 'Missing total_expenses');
    assert(typeof d?.net_balance    === 'number', 'Missing net_balance');
    assert(d?.period?.from,                       'Missing period.from');
    assert(d?.period?.to,                         'Missing period.to');
    assert(
      Math.abs(d.net_balance - (d.total_income - d.total_expenses)) < 0.01,
      `net_balance (${d.net_balance}) !== total_income (${d.total_income}) - total_expenses (${d.total_expenses})`
    );
  });

  await runTest('Viewer gets summary → 200', async () => {
    const res = await request('GET', '/dashboard/summary', {
      token: viewer.access_token,
      expectedStatus: 200,
    });
    assert(typeof res.data?.data?.net_balance === 'number', 'Missing net_balance for viewer');
  });

  await runTest('Analyst gets summary → 200', async () => {
    await request('GET', '/dashboard/summary', {
      token: analyst.access_token,
      expectedStatus: 200,
    });
  });

  await runTest('Summary defaults to current month when no params given → 200', async () => {
    const res = await request('GET', '/dashboard/summary', {
      token: admin.access_token,
      expectedStatus: 200,
    });
    assert(res.data?.data?.period?.from, 'Missing period.from when no params');
    assert(res.data?.data?.period?.to,   'Missing period.to when no params');
  });

  beginStep('7 · Dashboard — Trends');

  await runTest('Admin gets trends → 200 with array of monthly data', async () => {
    const res = await request('GET', '/dashboard/trends', {
      token: admin.access_token,
      expectedStatus: 200,
    });
    assert(Array.isArray(res.data?.data), 'trends data is not array');
    if (res.data.data.length > 0) {
      const first = res.data.data[0];
      assert(typeof first.month    === 'string', 'trend item missing month');
      assert(typeof first.income   === 'number', 'trend item missing income');
      assert(typeof first.expenses === 'number', 'trend item missing expenses');
      assert(typeof first.net      === 'number', 'trend item missing net');
    }
  });

  await runTest('Analyst gets trends → 200', async () => {
    await request('GET', '/dashboard/trends', {
      token: analyst.access_token,
      expectedStatus: 200,
    });
  });

  await runTest('Viewer gets trends → 403', async () => {
    await request('GET', '/dashboard/trends', {
      token: viewer.access_token,
      expectedStatus: 403,
    });
  });

  beginStep('7 · Dashboard — Categories');

  await runTest('All roles get category breakdown → 200 with income[] and expenses[]', async () => {
    for (const [label, token] of [['admin', admin.access_token], ['analyst', analyst.access_token], ['viewer', viewer.access_token]]) {
      const res = await request('GET', '/dashboard/categories', {
        token,
        expectedStatus: 200,
      });
      assert(Array.isArray(res.data?.data?.income),   `${label}: income not array`);
      assert(Array.isArray(res.data?.data?.expenses), `${label}: expenses not array`);
    }
  });

  beginStep('7 · Dashboard — Recent Activity');

  await runTest('GET /dashboard/recent returns array → 200', async () => {
    const res = await request('GET', '/dashboard/recent', {
      token: admin.access_token,
      expectedStatus: 200,
    });
    assert(Array.isArray(res.data?.data), 'recent data is not array');
  });

  await runTest('GET /dashboard/recent?limit=5 returns max 5 items', async () => {
    const res = await request('GET', '/dashboard/recent?limit=5', {
      token: admin.access_token,
      expectedStatus: 200,
    });
    assert((res.data?.data?.length ?? 0) <= 5,
      `Expected ≤5 records, got ${res.data?.data?.length}`);
  });

  await runTest('GET /dashboard/recent?limit=10 → 200', async () => {
    await request('GET', '/dashboard/recent?limit=10', {
      token: admin.access_token,
      expectedStatus: 200,
    });
  });

  await runTest('GET /dashboard/recent?limit=100 exceeds max → 400', async () => {
    await request('GET', '/dashboard/recent?limit=100', {
      token: admin.access_token,
      expectedStatus: 400,
    });
  });

  // ═══════════════════════════════════════════════════════════════════════
  // SECTION 8 — USER MANAGEMENT
  // ═══════════════════════════════════════════════════════════════════════
  section('8 · USERS — Admin management flow');
  beginStep('8 · Users — List & Read');

  await runTest('Admin lists users → 200 with paginated meta', async () => {
    const res = await request('GET', '/users?page=1&per_page=10', {
      token: admin.access_token,
      expectedStatus: 200,
    });
    assert(Array.isArray(res.data?.data),            'users data not array');
    assert(typeof res.data?.meta?.total === 'number','missing meta.total');
    assert(typeof res.data?.meta?.page  === 'number','missing meta.page');
  });

  let createdUserID;
  const newUserEmail = `e2e-created-${unique}@fin.local`;

  beginStep('8 · Users — Create');

  await runTest('Admin creates new viewer user → 201 with id', async () => {
    const res = await request('POST', '/users', {
      token: admin.access_token,
      body: { name: 'E2E Created User', email: newUserEmail, password: 'E2eUser@12345', role: 'viewer' },
      expectedStatus: 201,
    });
    assert(res.data?.data?.id,                     'Missing id after create');
    assert(res.data?.data?.email === newUserEmail, 'Email mismatch after create');
    assert(res.data?.data?.role === 'viewer',      'Role should be viewer');
    createdUserID = res.data.data.id;
  });

  await runTest('Admin gets user by ID → 200', async () => {
    if (!createdUserID) return;
    const res = await request('GET', `/users/${createdUserID}`, {
      token: admin.access_token,
      expectedStatus: 200,
    });
    assert(res.data?.data?.id === createdUserID, 'Returned wrong user');
  });

  await runTest('Create user with duplicate email → 400', async () => {
    await request('POST', '/users', {
      token: admin.access_token,
      body: { name: 'Dupe', email: newUserEmail, password: 'Admin@12345', role: 'viewer' },
      expectedStatus: 400,
    });
  });

  await runTest('Create user with invalid role → 400', async () => {
    await request('POST', '/users', {
      token: admin.access_token,
      body: { name: 'Bad', email: `bad-role-${unique}@fin.local`, password: 'Admin@12345', role: 'god' },
      expectedStatus: 400,
    });
  });

  beginStep('8 · Users — Update');

  await runTest('Admin updates user name → 200', async () => {
    if (!createdUserID) return;
    const res = await request('PATCH', `/users/${createdUserID}`, {
      token: admin.access_token,
      body: { name: 'E2E Updated Name' },
      expectedStatus: 200,
    });
    assert(res.data?.success === true, 'Update user returned success=false');
  });

  await runTest('Admin updates user role to analyst → 200', async () => {
    if (!createdUserID) return;
    const res = await request('PATCH', `/users/${createdUserID}/role`, {
      token: admin.access_token,
      body: { role: 'analyst' },
      expectedStatus: 200,
    });
    assert(res.data?.success === true, 'Role update returned success=false');
  });

  await runTest('Update role with invalid value → 400', async () => {
    if (!createdUserID) return;
    await request('PATCH', `/users/${createdUserID}/role`, {
      token: admin.access_token,
      body: { role: 'superuser' },
      expectedStatus: 400,
    });
  });

  beginStep('8 · Users — Status & Inactive login');

  await runTest('Admin deactivates user → 200', async () => {
    if (!createdUserID) return;
    const res = await request('PATCH', `/users/${createdUserID}/status`, {
      token: admin.access_token,
      body: { is_active: false },
      expectedStatus: 200,
    });
    assert(res.data?.success === true, 'Status update returned success=false');
  });

  await runTest('Inactive user cannot login → 401', async () => {
    await request('POST', '/auth/login', {
      body: { email: newUserEmail, password: 'E2eUser@12345' },
      expectedStatus: 401,
    });
  });

  await runTest('Admin reactivates user → 200', async () => {
    if (!createdUserID) return;
    await request('PATCH', `/users/${createdUserID}/status`, {
      token: admin.access_token,
      body: { is_active: true },
      expectedStatus: 200,
    });
  });

  beginStep('8 · Users — Delete');

  await runTest('Admin soft-deletes user → 200', async () => {
    if (!createdUserID) return;
    const res = await request('DELETE', `/users/${createdUserID}`, {
      token: admin.access_token,
      expectedStatus: 200,
    });
    assert(res.data?.success === true, 'Delete user returned success=false');
  });

  await runTest('Soft-deleted user does not appear in user list', async () => {
    if (!createdUserID) return;
    const res = await request('GET', '/users?per_page=100', {
      token: admin.access_token,
      expectedStatus: 200,
    });
    const ids = (res.data?.data || []).map(u => u.id);
    assert(!ids.includes(createdUserID),
      'Soft-deleted user still appears in GET /users list');
  });

  await runTest('Soft-deleted user cannot login → 401', async () => {
    await request('POST', '/auth/login', {
      body: { email: newUserEmail, password: 'E2eUser@12345' },
      expectedStatus: 401,
    });
  });

  // ═══════════════════════════════════════════════════════════════════════
  // SECTION 9 — LOGOUT
  // ═══════════════════════════════════════════════════════════════════════
  section('9 · AUTH — Logout');
  beginStep('9 · Auth — Logout');

  await runTest('Logout invalidates refresh token → 200', async () => {
    await request('POST', '/auth/logout', {
      token: analyst.access_token,
      body: { refresh_token: analyst.refresh_token },
      expectedStatus: 200,
    });
  });

  await runTest('Using revoked refresh token after logout → 401', async () => {
    await request('POST', '/auth/refresh', {
      body: { refresh_token: analyst.refresh_token },
      expectedStatus: 401,
    });
  });

  await runTest('Admin logout → 200', async () => {
    await request('POST', '/auth/logout', {
      token: admin.access_token,
      body: { refresh_token: admin.refresh_token },
      expectedStatus: 200,
    });
  });

  // ═══════════════════════════════════════════════════════════════════════
  // SECTION 10 — RESPONSE SHAPE VALIDATION
  // ═══════════════════════════════════════════════════════════════════════
  section('10 · RESPONSE SHAPE — Envelope consistency');
  beginStep('10 · Response shape');

  // Login again as viewer for final shape checks (admin token might be spent)
  let freshViewer;
  try {
    freshViewer = await login(cfg.viewer);
  } catch (_) {
    freshViewer = null;
  }

  await runTest('Every error response has success=false and error.code + error.message', async () => {
    const res = await request('GET', '/records', {
      token: 'garbage-token',
    });
    assert(res.data?.success === false,       'Error response success should be false');
    assert(res.data?.error?.code,             'Error response missing error.code');
    assert(res.data?.error?.message,          'Error response missing error.message');
  });

  await runTest('Every success response has success=true and data field', async () => {
    if (!freshViewer) return;
    const res = await request('GET', '/records', {
      token: freshViewer.access_token,
      expectedStatus: 200,
    });
    assert(res.data?.success === true, 'Success response success should be true');
    assert('data' in res.data,         'Success response missing data field');
  });

  await runTest('Paginated response has meta with page, per_page, total', async () => {
    if (!freshViewer) return;
    const res = await request('GET', '/records?page=1&per_page=5', {
      token: freshViewer.access_token,
      expectedStatus: 200,
    });
    const meta = res.data?.meta;
    assert(typeof meta?.page     === 'number', 'meta.page not a number');
    assert(typeof meta?.per_page === 'number', 'meta.per_page not a number');
    assert(typeof meta?.total    === 'number', 'meta.total not a number');
    assert(meta.per_page === 5,                `meta.per_page should be 5, got ${meta.per_page}`);
    assert(meta.page     === 1,                `meta.page should be 1, got ${meta.page}`);
  });

  // ─── Final summary ────────────────────────────────────────────────────────
  printSummary();
  process.exit(_failCount > 0 ? 1 : 0);
}

run().catch((err) => {
  console.error(red(bold('\n  FATAL UNHANDLED ERROR')));
  console.error(red(err.message));
  if (err.stack) {
    err.stack.split('\n').slice(1, 6).forEach(f => console.error(dim('  ' + f.trim())));
  }
  printSummary();
  process.exit(1);
});