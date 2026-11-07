import type { AdminApi } from '../../shared/api/admin'
import type { Device, Enrollment } from './types'

export function createDevicesApi(api: AdminApi) {
  return {
    enroll: (input: Enrollment) =>
      api<{ device: Device; token: string }>('/devices', {
        method: 'POST',
        body: JSON.stringify(input),
      }),
    rotateToken: (id: string) =>
      api<{ token: string }>(`/devices/${id}/rotate-token`, { method: 'POST' }),
    remove: (id: string) => api<void>(`/devices/${id}`, { method: 'DELETE' }),
  }
}
