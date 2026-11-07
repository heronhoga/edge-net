import { test, expect } from '@playwright/test'

// Browser-side session/UI coverage. Signed OAuth exchange and ownership enforcement
// are exercised against a local OIDC provider by internal/portal/auth_test.go.
test('Google session restores, sends CSRF, switches accounts, and signs out', async ({
  page,
  context,
}) => {
  const errors: string[] = []
  page.on('pageerror', (error) => errors.push(error.message))
  let signedIn = false
  let user = { id: 'alice', name: 'Alice Example', email: 'alice@example.com', provider: 'google' }
  let csrf = 'alice-csrf'
  let devices: object[] = []

  await page.route('**/api/auth/session', (route) =>
    route.fulfill({
      json: {
        user: signedIn ? user : null,
        csrfToken: signedIn ? csrf : '',
        googleEnabled: true,
        adminEnabled: false,
      },
    }),
  )
  await page.route('**/api/auth/google', async (route) => {
    signedIn = true
    await route.fulfill({
      status: 302,
      headers: {
        location: '/',
        'set-cookie': 'edgenet_session=browser-test-session; Path=/; HttpOnly; SameSite=Lax',
      },
    })
  })
  await page.route('**/api/admin/state', (route) =>
    route.fulfill({
      json: {
        user,
        csrfToken: csrf,
        devices,
        firmware: [],
        deployments: [],
        events: [],
        serverTime: new Date().toISOString(),
      },
    }),
  )
  await page.route('**/api/admin/devices', async (route) => {
    const headers = route.request().headers()
    expect(headers['x-csrf-token']).toBe(csrf)
    expect(headers.authorization).toBeUndefined()
    expect(headers.cookie).toContain('edgenet_session=browser-test-session')
    const input = route.request().postDataJSON()
    const device = {
      ...input,
      id: 'alice-device',
      status: 'pending',
      version: 'unknown',
      lastSeen: null,
      createdAt: new Date().toISOString(),
      metrics: { uptime: 0 },
      history: [],
    }
    devices = [device]
    await route.fulfill({ status: 201, json: { device, token: 'device-token' } })
  })
  await page.route('**/api/auth/logout', async (route) => {
    expect(route.request().method()).toBe('POST')
    expect(route.request().headers()['x-csrf-token']).toBe('bob-csrf')
    signedIn = false
    await route.fulfill({
      status: 204,
      headers: { 'set-cookie': 'edgenet_session=; Path=/; Max-Age=0; HttpOnly; SameSite=Lax' },
    })
  })

  await page.goto('/')
  await expect(page.getByRole('button', { name: 'Continue with Google' })).toBeEnabled()
  await expect(page.getByText('Local administrator', { exact: true })).not.toBeVisible()
  await page.getByRole('button', { name: 'Continue with Google' }).click()
  await expect(page.getByRole('heading', { name: 'Fleet overview' })).toBeVisible()
  await expect(page.getByText('alice@example.com', { exact: true })).toBeVisible()
  expect(
    (await context.cookies()).find((cookie) => cookie.name === 'edgenet_session')?.httpOnly,
  ).toBe(true)
  expect(await page.evaluate(() => sessionStorage.getItem('edgenet-token'))).toBeNull()
  await page.reload()
  await expect(page.getByRole('heading', { name: 'Fleet overview' })).toBeVisible()

  await page.getByRole('button', { name: 'Add device', exact: true }).click()
  const dialog = page.getByRole('dialog')
  await dialog.getByLabel('Device name').fill('Alice sensor')
  await dialog.getByRole('button', { name: 'Add device', exact: true }).click()
  await expect(
    dialog.getByRole('heading', { name: 'Your device is ready to connect' }),
  ).toBeVisible()
  await dialog.getByRole('button', { name: 'I saved my token' }).click()
  await page.getByRole('button', { name: 'View Alice sensor' }).click()

  // Simulate the shared Google cookie changing in another browser tab.
  user = { id: 'bob', name: 'Bob Example', email: 'bob@example.com', provider: 'google' }
  csrf = 'bob-csrf'
  devices = []
  // The refresh button is behind the modal, so use the normal five-second poll.
  await expect(dialog).not.toBeVisible({ timeout: 10000 })
  await expect(page.getByText('bob@example.com', { exact: true })).toBeVisible()
  await expect(page.getByText('Alice sensor', { exact: true })).not.toBeVisible()
  await expect(page.getByRole('heading', { name: 'Your fleet starts here' })).toBeVisible()
  await page.getByRole('button', { name: 'Sign out' }).click()
  await expect(page.getByRole('heading', { name: 'Welcome to Edgenet' })).toBeVisible()
  await page.reload()
  await expect(page.getByRole('button', { name: 'Continue with Google' })).toBeEnabled()
  expect(errors).toEqual([])
})

test('Google callback failures are explained and removed from the URL', async ({ page }) => {
  await page.route('**/api/auth/session', (route) =>
    route.fulfill({
      json: {
        user: null,
        csrfToken: '',
        googleEnabled: true,
        adminEnabled: false,
      },
    }),
  )
  await page.goto('/?auth_error=expired')
  await expect(page.getByRole('alert')).toContainText('Your sign-in request expired')
  await expect(page.getByRole('button', { name: 'Continue with Google' })).toBeEnabled()
  expect(new URL(page.url()).searchParams.has('auth_error')).toBe(false)
})
