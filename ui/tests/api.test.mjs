import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { test } from 'node:test'
import { transformWithEsbuild } from 'vite'

// Exercise the real Vite client with isolated browser storage and HTTP responses.
const source = await readFile(new URL('../src/api.ts', import.meta.url), 'utf8')
const { code } = await transformWithEsbuild(source, 'api.ts', {
  define: { 'import.meta.env.VITE_API_BASE': '"http://api.test"' },
})
const { api, ApiError } = await import(`data:text/javascript;base64,${Buffer.from(code).toString('base64')}`)
Object.defineProperty(globalThis, 'localStorage', {
  configurable: true,
  value: { getItem: () => null },
})

function respond(t, body, status = 200) {
  t.mock.method(globalThis, 'fetch', async () => new Response(body, { status }))
}

test('unwraps object and list data without exposing the envelope to components', async (t) => {
  respond(t, JSON.stringify({ data: [{ id: 'wallet-1' }], meta: { has_more: false } }))
  assert.deepEqual(await api.listWallets(), [{ id: 'wallet-1' }])
  t.mock.restoreAll()
  respond(t, JSON.stringify({ data: { transaction: { id: 'tx-1' }, quote: {} } }), 201)
  assert.deepEqual(await api.createSwap('wallet-1', 'XLM', 'USDC', '1', 100), {
    transaction: { id: 'tx-1' }, quote: {},
  })
})

test('preserves machine-readable error codes and safe messages', async (t) => {
  respond(t, JSON.stringify({ error: { code: 'wallet_not_found', message: 'Wallet not found.' } }), 404)
  await assert.rejects(api.listWallets(), (err) => {
    assert.ok(err instanceof ApiError)
    assert.equal(err.status, 404)
    assert.equal(err.code, 'wallet_not_found')
    assert.equal(err.message, 'Wallet not found.')
    return true
  })
})

test('accepts 204 without attempting to parse JSON', async (t) => {
  respond(t, null, 204)
  assert.equal(await api.deleteWallet('wallet-1'), undefined)
})

test('uses a safe fallback for non-JSON and malformed error bodies', async (t) => {
  for (const body of ['<html>private gateway details</html>', 'null', '{}', '{"error":{"message":{}}}']) {
    respond(t, body, 502)
    await assert.rejects(api.listWallets(), (err) => {
      assert.equal(err.code, 'request_failed')
      assert.equal(err.message, 'Request failed (HTTP 502).')
      return true
    })
    t.mock.restoreAll()
  }
})

test('rejects successful responses that violate the envelope contract', async (t) => {
  for (const body of ['[]', '{}', 'null', '<html>not JSON</html>']) {
    respond(t, body)
    await assert.rejects(api.listWallets(), (err) => {
      assert.equal(err.code, 'invalid_response')
      return true
    })
    t.mock.restoreAll()
  }
})

test('auth uses credentialed cookies and CSRF headers without bearer tokens', async (t) => {
  t.mock.method(globalThis, 'fetch', async (url, options) => {
    assert.equal(options.credentials, 'include')
    assert.equal(options.headers['X-CSRF-Protection'], '1')
    assert.equal(options.headers.Authorization, undefined)
    assert.equal(url, 'http://api.test/api/v1/auth/login')
    return new Response(JSON.stringify({ data: { id: 'user-1', email: 'owner@example.com', has_password: true, identities: [] } }))
  })
  assert.equal(await api.login('owner@example.com', 'password123'), 'owner@example.com')
})

test('logout accepts an empty 204 response', async (t) => {
  respond(t, null, 204)
  assert.equal(await api.logout(), undefined)
})

test('returns either the sent transaction or the held payment request', async (t) => {
  respond(t, JSON.stringify({ data: { transaction: { id: 'tx-1', status: 'signing' } } }), 201)
  assert.deepEqual(await api.createTxn('wallet-1', 'G…', '5', ''), { transaction: { id: 'tx-1', status: 'signing' } })
  t.mock.restoreAll()
  respond(t, JSON.stringify({ data: { request: { id: 'request-1', status: 'pending', requiredApprovals: 1 } } }), 202)
  assert.deepEqual(await api.createTxn('wallet-1', 'G…', '50', ''), {
    request: { id: 'request-1', status: 'pending', requiredApprovals: 1 },
  })
})
