#!/usr/bin/env node
/*
  End-to-end API test script for Finance Dashboard backend.
  Requires Node.js 18+ (global fetch).

  Usage:
    BASE_URL=http://localhost:8080/api/v1 node scripts/e2e-test.js

  Optional env vars:
    ADMIN_EMAIL (default: admin@fin.local)
    ADMIN_PASSWORD (default: Admin@12345)
    ANALYST_EMAIL (default: analyst@fin.local)
    ANALYST_PASSWORD (default: Analyst@12345)
    VIEWER_EMAIL (default: viewer@fin.local)
    VIEWER_PASSWORD (default: Viewer@12345)
*/

const BASE_URL = process.env.BASE_URL || 'http://localhost:8080/api/v1';

const cfg = {
  admin: {
    email: process.env.ADMIN_EMAIL || 'admin@fin.local',
    password: process.env.ADMIN_PASSWORD || 'Admin@12345',
  },
  analyst: {
    email: process.env.ANALYST_EMAIL || 'analyst@fin.local',
    password: process.env.ANALYST_PASSWORD || 'Analyst@12345',
  },
  viewer: {
    email: process.env.VIEWER_EMAIL || 'viewer@fin.local',
    password: process.env.VIEWER_PASSWORD || 'Viewer@12345',
  },
};

function assert(condition, message) {
  if (!condition) throw new Error(message);
}

async function request(method, path, { token, body, expectedStatus } = {}) {
  const headers = { 'Content-Type': 'application/json' };
  if (token) headers.Authorization = `Bearer ${token}`;

  const res = await fetch(`${BASE_URL}${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });

  let data = null;
  const text = await res.text();
  try {
    data = text ? JSON.parse(text) : null;
  } catch {
    data = { raw: text };
  }

  if (typeof expectedStatus === 'number') {
    assert(
      res.status === expectedStatus,
      `${method} ${path} expected ${expectedStatus}, got ${res.status}: ${text}`,
    );
  }

  return { status: res.status, data, raw: text };
}

async function login(user) {
  const out = await request('POST', '/auth/login', {
    body: { email: user.email, password: user.password },
    expectedStatus: 200,
  });

  assert(out.data?.success === true, 'Login response success=false');
  assert(out.data?.data?.access_token, 'Missing access_token in login response');
  assert(out.data?.data?.refresh_token, 'Missing refresh_token in login response');
  return out.data.data;
}

async function run() {
  console.log(`Running E2E tests against ${BASE_URL}`);

  // 1) Login all roles
  const admin = await login(cfg.admin);
  const analyst = await login(cfg.analyst);
  const viewer = await login(cfg.viewer);
  console.log('✓ Login passed for admin/analyst/viewer');

  // 2) Refresh token
  const refresh = await request('POST', '/auth/refresh', {
    body: { refresh_token: admin.refresh_token },
    expectedStatus: 200,
  });
  assert(refresh.data?.data?.access_token, 'Missing refreshed access_token');
  console.log('✓ Refresh token passed');

  // 3) RBAC checks
  await request('POST', '/records', {
    token: viewer.access_token,
    body: {
      amount: 11,
      type: 'expense',
      category_id: 6,
      date: '2026-04-01',
      description: 'viewer should fail',
    },
    expectedStatus: 403,
  });

  await request('GET', '/dashboard/trends', {
    token: analyst.access_token,
    expectedStatus: 200,
  });

  await request('GET', '/dashboard/trends', {
    token: viewer.access_token,
    expectedStatus: 403,
  });
  console.log('✓ RBAC checks passed');

  // 4) Category create + update
  const unique = Date.now();
  const createdCategory = await request('POST', '/categories', {
    token: admin.access_token,
    body: { name: `E2E-Category-${unique}`, type: 'expense' },
    expectedStatus: 201,
  });
  const categoryID = createdCategory.data?.data?.id;
  assert(categoryID, 'Missing category id after create');

  await request('PATCH', `/categories/${categoryID}`, {
    token: admin.access_token,
    body: { name: `E2E-Category-Updated-${unique}` },
    expectedStatus: 200,
  });
  console.log('✓ Category create/update passed');

  // 5) Record create, list, get, update, void
  const createRecord = await request('POST', '/records', {
    token: admin.access_token,
    body: {
      amount: 999.55,
      type: 'expense',
      category_id: categoryID,
      date: '2026-04-01',
      description: 'E2E created record',
    },
    expectedStatus: 201,
  });
  const recordID = createRecord.data?.data?.id;
  assert(recordID, 'Missing record id after create');

  await request('GET', '/records?page=1&per_page=20', {
    token: admin.access_token,
    expectedStatus: 200,
  });

  await request('GET', `/records/${recordID}`, {
    token: admin.access_token,
    expectedStatus: 200,
  });

  await request('PATCH', `/records/${recordID}`, {
    token: admin.access_token,
    body: { description: 'E2E updated description', category_id: categoryID },
    expectedStatus: 200,
  });

  await request('PATCH', `/records/${recordID}`, {
    token: admin.access_token,
    body: { amount: 1234.56 },
    expectedStatus: 400,
  });

  await request('POST', `/records/${recordID}/void`, {
    token: admin.access_token,
    body: { reason: 'Duplicate record in automated end-to-end test run' },
    expectedStatus: 200,
  });
  console.log('✓ Record flow passed');

  // 6) Dashboard endpoints
  const today = new Date();
  const from = `${today.getUTCFullYear()}-${String(today.getUTCMonth() + 1).padStart(2, '0')}-01`;
  const to = `${today.getUTCFullYear()}-${String(today.getUTCMonth() + 1).padStart(2, '0')}-31`;

  await request('GET', `/dashboard/summary?from=${from}&to=${to}`, {
    token: admin.access_token,
    expectedStatus: 200,
  });
  await request('GET', '/dashboard/trends', {
    token: admin.access_token,
    expectedStatus: 200,
  });
  await request('GET', '/dashboard/categories', {
    token: admin.access_token,
    expectedStatus: 200,
  });
  await request('GET', '/dashboard/recent?limit=10', {
    token: admin.access_token,
    expectedStatus: 200,
  });
  console.log('✓ Dashboard flow passed');

  // 7) User admin flow
  const userEmail = `e2e-user-${unique}@fin.local`;
  const createUser = await request('POST', '/users', {
    token: admin.access_token,
    body: {
      name: 'E2E User',
      email: userEmail,
      password: 'E2eUser@12345',
      role: 'viewer',
    },
    expectedStatus: 201,
  });
  const userID = createUser.data?.data?.id;
  assert(userID, 'Missing user id after user create');

  await request('PATCH', `/users/${userID}/role`, {
    token: admin.access_token,
    body: { role: 'analyst' },
    expectedStatus: 200,
  });

  await request('PATCH', `/users/${userID}/status`, {
    token: admin.access_token,
    body: { is_active: false },
    expectedStatus: 200,
  });

  await request('DELETE', `/users/${userID}`, {
    token: admin.access_token,
    expectedStatus: 200,
  });
  console.log('✓ User admin flow passed');

  // 8) Logout
  await request('POST', '/auth/logout', {
    token: admin.access_token,
    body: { refresh_token: admin.refresh_token },
    expectedStatus: 200,
  });
  console.log('✓ Logout passed');

  console.log('\n🎉 E2E test suite completed successfully.');
}

run().catch((err) => {
  console.error('\n❌ E2E test suite failed');
  console.error(err.message || err);
  process.exit(1);
});
