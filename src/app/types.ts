import type { Device } from '../domains/devices/types'
import type { Firmware } from '../domains/firmware/types'
import type { Deployment } from '../domains/deployments/types'
import type { ActivityEvent } from '../domains/overview/types'
import type { User } from '../domains/auth/types'

export type Page = 'overview' | 'devices' | 'firmware' | 'deployments' | 'connect'
export type Dialog = '' | 'enroll' | 'upload' | 'deploy' | 'credentials' | 'device'

export interface Snapshot {
  devices: Device[]
  firmware: Firmware[]
  deployments: Deployment[]
  events: ActivityEvent[]
  serverTime: string
}

export interface WorkspaceSnapshot extends Snapshot {
  user: User
  csrfToken: string
}
