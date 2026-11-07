import type { AdminApi } from '../../shared/api/admin'
import type { FirmwareUpload } from './types'

export const MAX_ARTIFACT_SIZE = 64 * 1024 * 1024

export function createFirmwareApi(api: AdminApi) {
  return {
    upload(input: FirmwareUpload) {
      if (input.file.size > MAX_ARTIFACT_SIZE) {
        throw new Error('The artifact must be 64 MiB or smaller.')
      }
      const body = new FormData()
      body.set('file', input.file)
      body.set('version', input.version)
      body.set('target', input.target)
      return api('/firmware', { method: 'POST', body })
    },
  }
}
