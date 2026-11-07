<script setup lang="ts">
import { computed } from 'vue'
import { Plus, Upload } from 'lucide-vue-next'
import { useWorkspace } from './app/useWorkspace'
import AppLayout from './components/layout/AppLayout.vue'
import Modal from './components/ui/Modal.vue'
import ToastMessage from './components/ui/ToastMessage.vue'
import LoginView from './domains/auth/components/LoginView.vue'
import ConnectionGuide from './domains/connection/components/ConnectionGuide.vue'
import DeploymentForm from './domains/deployments/components/DeploymentForm.vue'
import DeploymentsView from './domains/deployments/components/DeploymentsView.vue'
import { isActive } from './domains/deployments/status'
import DeviceCredentials from './domains/devices/components/DeviceCredentials.vue'
import DeviceDetails from './domains/devices/components/DeviceDetails.vue'
import DeviceList from './domains/devices/components/DeviceList.vue'
import EnrollmentForm from './domains/devices/components/EnrollmentForm.vue'
import FirmwareUploadForm from './domains/firmware/components/FirmwareUploadForm.vue'
import FirmwareView from './domains/firmware/components/FirmwareView.vue'
import FleetNote from './domains/overview/components/FleetNote.vue'
import FleetStats from './domains/overview/components/FleetStats.vue'
import RecentActivity from './domains/overview/components/RecentActivity.vue'

const {
  user,
  googleEnabled,
  adminEnabled,
  initializing,
  startGoogle,
  restoreSession,
  loading,
  busy,
  error,
  modalError,
  notice,
  page,
  dialog,
  state,
  selected,
  credentials,
  deploymentDefaults,
  login,
  logout,
  refresh,
  open,
  close,
  openDevice,
  openDeploy,
  enroll,
  upload,
  deploy,
  cancel,
  rotate,
  remove,
  copy,
} = useWorkspace()

const showDevices = computed(() => page.value === 'overview' || page.value === 'devices')
const activeJobCount = computed(
  () => state.value.deployments.filter((job) => isActive(job.status)).length,
)
const hardwareTargets = computed(() => [
  ...new Set(state.value.devices.map((device) => device.target)),
])
const dialogTitle = computed(() => {
  switch (dialog.value) {
    case 'enroll':
      return 'Add a device'
    case 'upload':
      return 'Upload firmware'
    case 'deploy':
      return 'Deploy an update'
    case 'credentials':
      return 'Your device is ready to connect'
    default:
      return selected.value?.name || 'Device details'
  }
})
</script>

<template>
  <LoginView
    v-if="!user"
    :loading="loading"
    :initializing="initializing"
    :error="error"
    :google-enabled="googleEnabled"
    :admin-enabled="adminEnabled"
    @login="login"
    @google="startGoogle"
    @retry="restoreSession"
  />
  <AppLayout
    v-else
    :key="user.id"
    :user="user"
    :page="page"
    :device-count="state.devices.length"
    :loading="loading"
    :error="error"
    @navigate="page = $event"
    @logout="logout"
    @refresh="refresh"
  >
    <template #actions>
      <button v-if="showDevices" class="button primary" @click="open('enroll')">
        <Plus :size="17" />Add device
      </button>
      <button v-else-if="page === 'firmware'" class="button primary" @click="open('upload')">
        <Upload :size="17" />Upload firmware
      </button>
      <button v-else-if="page === 'deployments'" class="button primary" @click="openDeploy()">
        <Plus :size="17" />New deployment
      </button>
    </template>

    <FleetStats
      v-if="showDevices"
      :devices="state.devices"
      :active-job-count="activeJobCount"
      :firmware-count="state.firmware.length"
    />
    <KeepAlive>
      <DeviceList
        v-if="showDevices"
        :devices="state.devices"
        @enroll="open('enroll')"
        @deploy="openDeploy()"
        @view="openDevice"
      />
    </KeepAlive>
    <div v-if="page === 'overview'" class="bottom-grid">
      <RecentActivity :events="state.events" />
      <FleetNote @guide="page = 'connect'" />
    </div>
    <FirmwareView
      v-if="page === 'firmware'"
      :firmware="state.firmware"
      @upload="open('upload')"
      @copy="copy"
      @deploy="openDeploy"
    />
    <DeploymentsView
      v-if="page === 'deployments'"
      :deployments="state.deployments"
      :busy="busy"
      @create="openDeploy()"
      @cancel="cancel"
    />
    <ConnectionGuide v-if="page === 'connect'" @enroll="open('enroll')" />
  </AppLayout>

  <Modal v-if="dialog" :title="dialogTitle" :wide="dialog === 'device'" @close="close">
    <p v-if="modalError" class="error-box modal-error" role="alert">{{ modalError }}</p>
    <EnrollmentForm v-if="dialog === 'enroll'" :busy="busy" @submit="enroll" @close="close" />
    <DeviceCredentials
      v-else-if="dialog === 'credentials'"
      :credentials="credentials"
      @copy="copy"
      @close="close"
    />
    <FirmwareUploadForm
      v-else-if="dialog === 'upload'"
      :busy="busy"
      :targets="hardwareTargets"
      @submit="upload"
      @close="close"
    />
    <DeploymentForm
      v-else-if="dialog === 'deploy'"
      :devices="state.devices"
      :firmware="state.firmware"
      :deployments="state.deployments"
      :defaults="deploymentDefaults"
      :busy="busy"
      @submit="deploy"
      @close="close"
      @upload="open('upload')"
    />
    <DeviceDetails
      v-else-if="dialog === 'device' && selected"
      :device="selected"
      :busy="busy"
      @remove="remove"
      @rotate="rotate"
      @deploy="openDeploy(undefined, $event)"
    />
  </Modal>
  <ToastMessage :message="notice" @dismiss="notice = ''" />
</template>
