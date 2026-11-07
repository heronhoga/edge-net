export interface Metrics {
  cpu?: number
  memory?: number
  temperature?: number
  uptime: number
}
export interface Device {
  id: string
  name: string
  kind: string
  target: string
  location: string
  version: string
  status: string
  lastSeen: string | null
  createdAt: string
  metrics: Metrics
  history: (Metrics & { at: string })[]
}
export interface Firmware {
  id: string
  name: string
  version: string
  target: string
  sha256: string
  size: number
  createdAt: string
}
export interface Deployment {
  id: string
  deviceId: string
  deviceName: string
  firmwareId: string
  version: string
  status: string
  message: string
  createdAt: string
  updatedAt: string
}
export interface Snapshot {
  devices: Device[]
  firmware: Firmware[]
  deployments: Deployment[]
  events: { id: string; message: string; at: string }[]
  serverTime: string
}
