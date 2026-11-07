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

export interface Enrollment {
  name: string
  kind: string
  target: string
  location: string
}

export interface DeviceCredentials {
  id: string
  token: string
}
