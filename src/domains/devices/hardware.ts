import { Code2, Cpu } from 'lucide-vue-next'

export const kinds: Record<string, string> = {
  esp32: 'ESP32',
  arduino: 'Arduino',
}

export const targets: Record<string, string> = {
  esp32: 'esp32-devkit-v1',
  arduino: 'arduino-uno-r4-wifi',
}

export function kindIcon(kind: string) {
  return kind === 'arduino' ? Code2 : Cpu
}
