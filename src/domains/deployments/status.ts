export function isActive(status: string) {
  return ['queued', 'downloading', 'installing'].includes(status)
}
