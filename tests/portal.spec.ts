import { test, expect } from '@playwright/test'

test('enroll, monitor, upload, deploy, and view on mobile', async ({ page, request }, testInfo) => {
  const errors: string[] = []
  page.on('pageerror', (error) => errors.push(error.message))
  const unique = Date.now().toString()
  const name = `Lab gateway ${unique}`
  const target = `linux-e2e-${unique}`
  await page.goto('/')
  await page.getByLabel('Admin access token').fill('e2e-only-admin-token')
  await page.getByRole('button', { name: 'Open workspace' }).click()
  await expect(page.getByRole('heading', { name: 'Fleet overview' })).toBeVisible()

  await page.getByRole('button', { name: 'Add device', exact: true }).click()
  const dialog = page.getByRole('dialog')
  await dialog.getByLabel('Device name').fill(name)
  await dialog.getByLabel('Device type').selectOption('mini-pc')
  await dialog.getByLabel('Hardware target').fill(target)
  await dialog.getByLabel('Location').fill('Workshop')
  await dialog.getByRole('button', { name: 'Add device', exact: true }).click()
  await expect(
    dialog.getByRole('heading', { name: 'Your device is ready to connect' }),
  ).toBeVisible()
  const deviceId = await dialog.locator('.copy-field code').nth(0).innerText()
  const token = await dialog.locator('.copy-field code').nth(1).innerText()
  expect(token).toHaveLength(48)
  await dialog.getByRole('button', { name: 'I saved my token' }).click()
  const headers = { Authorization: `Bearer ${token}`, 'X-Device-ID': deviceId }
  const heartbeat = await request.post('/api/agent/heartbeat', {
    headers,
    data: { version: '1.0.0', metrics: { cpu: 24, memory: 38, temperature: 42, uptime: 3600 } },
  })
  expect(heartbeat.ok()).toBeTruthy()
  await page.getByRole('button', { name: 'Refresh dashboard' }).click()
  const row = page.getByRole('row').filter({ hasText: name })
  await expect(row.getByText('online', { exact: true })).toBeVisible()
  await expect(row.getByText('24%', { exact: true })).toBeVisible()
  await page.screenshot({ path: testInfo.outputPath('dashboard-desktop.png'), fullPage: true })

  await page.getByRole('button', { name: 'Firmware', exact: true }).click()
  await page.getByRole('button', { name: 'Upload firmware', exact: true }).click()
  await dialog
    .getByLabel('Firmware artifact')
    .setInputFiles({
      name: 'update.bin',
      mimeType: 'application/octet-stream',
      buffer: Buffer.from('browser integration test package'),
    })
  await dialog.getByLabel('Version', { exact: true }).fill('2.0.0')
  await dialog.getByLabel('Hardware target').fill(target)
  await dialog.getByRole('button', { name: 'Upload artifact', exact: true }).click()
  await expect(dialog).not.toBeVisible()
  const card = page.locator('.firmware-card').filter({ hasText: target })
  await card.getByRole('button', { name: 'Deploy', exact: true }).click()
  await dialog.locator('.selection-row').filter({ hasText: name }).getByRole('checkbox').check()
  await dialog.getByRole('button', { name: 'Deploy to 1 device', exact: true }).click()
  await expect(page.getByRole('heading', { name: 'Deployments', exact: true })).toBeVisible()
  const jobsResponse = await request.get('/api/agent/jobs', { headers })
  expect(jobsResponse.ok()).toBeTruthy()
  const jobs = await jobsResponse.json()
  expect(jobs).toHaveLength(1)
  for (const status of ['downloading', 'installing', 'succeeded']) {
    const result = await request.post(`/api/agent/jobs/${jobs[0].deployment.id}/status`, {
      headers,
      data: { status },
    })
    expect(result.status()).toBe(204)
  }
  await page.getByRole('button', { name: 'Refresh dashboard' }).click()
  await expect(
    page.getByRole('row').filter({ hasText: name }).getByText('succeeded', { exact: true }),
  ).toBeVisible()

  await page.getByRole('button', { name: /^Devices/ }).click()
  await page.getByRole('button', { name: `View ${name}` }).click()
  await expect(dialog.getByText('42.0°C')).toBeVisible()
  await expect(dialog.getByText('2.0.0', { exact: true })).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(dialog).not.toBeVisible()

  await page.setViewportSize({ width: 390, height: 844 })
  await expect(page.getByRole('heading', { name: 'Devices', exact: true })).toBeVisible()
  await page.screenshot({ path: testInfo.outputPath('dashboard-mobile.png'), fullPage: true })
  const layout = await page.evaluate(() => ({
    page: document.documentElement.scrollWidth,
    viewport: window.innerWidth,
    containers: ['.main-shell', '.content', '.table-scroll', '.sr-only'].map((selector) => {
      const rect = document.querySelector(selector)!.getBoundingClientRect()
      return { selector, x: rect.x, width: rect.width }
    }),
  }))
  expect(layout.page, JSON.stringify(layout)).toBeLessThanOrEqual(layout.viewport)
  await page.getByRole('button', { name: 'Sign out' }).click()
  await expect(page.getByRole('heading', { name: 'Welcome to Edgenet' })).toBeVisible()
  expect(errors).toEqual([])
  // Revoke the test identity after the successful terminal deployment.
  const removed = await request.delete(`/api/admin/devices/${deviceId}`, {
    headers: { Authorization: 'Bearer e2e-only-admin-token' },
  })
  expect(removed.status()).toBe(204)
})
