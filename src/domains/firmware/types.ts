export interface Firmware {
  id: string
  name: string
  version: string
  target: string
  sha256: string
  size: number
  createdAt: string
}

export interface FirmwareUpload {
  file: File
  version: string
  target: string
}
