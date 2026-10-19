<template>
  <!-- PC端使用原组件，移动端使用新组件 -->
  <DesktopNotePad 
    v-if="!isMobile" 
    ref="desktopRef" 
    v-bind="$props" 
    @update:visible="$emit('update:visible', $event)" 
    @remind-status-changed="$emit('remindStatusChanged', $event)" 
  />
  <MobileNotePad 
    v-else 
    ref="mobileRef" 
    v-bind="$props" 
    @update:visible="$emit('update:visible', $event)" 
    @remind-status-changed="$emit('remindStatusChanged', $event)" 
  />
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import DesktopNotePad from './DesktopNotePad.vue'
import MobileNotePad from './MobileNotePad.vue'

defineProps<{
  visible: boolean
}>()

defineEmits<{
  (e: 'update:visible', visible: boolean): void
  (e: 'remindStatusChanged', noteId: number): void
}>()

const desktopRef = ref<any>(null)
const mobileRef = ref<any>(null)

// 检测设备类型（响应式）
const isMobile = ref(false)

const checkMobile = () => {
  if (typeof window !== 'undefined') {
    isMobile.value = window.innerWidth <= 768
  }
}

onMounted(() => {
  checkMobile()
  window.addEventListener('resize', checkMobile)
})

onUnmounted(() => {
  window.removeEventListener('resize', checkMobile)
})

const refreshData = () => {
  if (isMobile.value) {
    mobileRef.value?.refreshData?.()
  } else {
    desktopRef.value?.refreshData?.()
  }
}

const selectNote = (note: any) => {
  if (isMobile.value) {
    mobileRef.value?.selectNote?.(note)
  } else {
    desktopRef.value?.selectNote?.(note)
  }
}

defineExpose({
  refreshData,
  selectNote,
})
</script>
