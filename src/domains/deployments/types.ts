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

export interface DeploymentRequest {
  firmwareId: string
  deviceIds: string[]
}
