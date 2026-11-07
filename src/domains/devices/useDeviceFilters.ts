import { computed, ref, type Ref } from 'vue'
import { kinds } from './hardware'
import type { Device } from './types'

export function useDeviceFilters(devices: Ref<Device[]>) {
  const query = ref('')
  const filter = ref('all')
  const filteredDevices = computed(() =>
    devices.value.filter(
      (device) =>
        (filter.value === 'all' || device.status === filter.value) &&
        `${device.name} ${device.target} ${device.location} ${kinds[device.kind]}`
          .toLowerCase()
          .includes(query.value.toLowerCase()),
    ),
  )

  function clearFilters() {
    query.value = ''
    filter.value = 'all'
  }

  return { query, filter, filteredDevices, clearFilters }
}
