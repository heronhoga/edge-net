import type { AdminApi } from '../../shared/api/admin'
import type { DeploymentRequest } from './types'

export function createDeploymentsApi(api: AdminApi) {
  return {
    create: (input: DeploymentRequest) =>
      api('/deployments', { method: 'POST', body: JSON.stringify(input) }),
    cancel: (id: string) => api<void>(`/deployments/${id}/cancel`, { method: 'POST' }),
  }
}
