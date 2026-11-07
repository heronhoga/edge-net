import { ArrowUpRight, Cpu, Layers3, LayoutDashboard } from 'lucide-vue-next'
import type { Page } from './types'

export const navigation = [
  { id: 'overview', label: 'Overview', icon: LayoutDashboard },
  { id: 'devices', label: 'Devices', icon: Cpu },
  { id: 'firmware', label: 'Firmware', icon: Layers3 },
  { id: 'deployments', label: 'Deployments', icon: ArrowUpRight },
] satisfies { id: Page; label: string; icon: typeof Cpu }[]

export const pageDescriptions: Record<Page, string> = {
  overview: 'A little visibility goes a long way. Here’s how your devices are doing.',
  devices: 'Every Arduino and ESP32 board. All in one place.',
  firmware: 'Versioned artifacts, ready for their next deployment.',
  deployments: 'Follow every update from queued to completed.',
  connect: 'Connect your Arduino and ESP32 boards with a simple HTTP protocol.',
}

export function pageTitle(page: Page) {
  return page === 'connect'
    ? 'Connect your devices'
    : navigation.find((item) => item.id === page)!.label
}
