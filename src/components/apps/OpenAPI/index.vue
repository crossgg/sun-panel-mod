<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { NAlert, NButton, NCard, NInput, NPopconfirm, NTag, useMessage } from 'naive-ui'
import { SvgIcon } from '@/components/common'
import { t } from '@/locales'
import {
  type OpenAPIConfig,
  disableOpenAPI,
  enableOpenAPI,
  getOpenAPIConfig,
  refreshOpenAPIToken,
} from '@/api/system/openApi'

const message = useMessage()
const loading = ref(false)
const config = ref<OpenAPIConfig>({
  enabled: false,
  token: '',
  endpoint: '',
})
const endpoint = computed(() => config.value.endpoint || `${window.location.origin}/openapi/v1`)

async function loadConfig() {
  loading.value = true
  try {
    const response = await getOpenAPIConfig()
    if (response.code === 0)
      config.value = response.data
  }
  finally {
    loading.value = false
  }
}

async function updateConfig(request: () => ReturnType<typeof enableOpenAPI>, successText: string) {
  loading.value = true
  try {
    const response = await request()
    if (response.code === 0) {
      config.value = response.data
      message.success(successText)
    }
    else {
      message.error(response.msg)
    }
  }
  catch {
    message.error(t('common.networkError'))
  }
  finally {
    loading.value = false
  }
}

function handleEnable() {
  updateConfig(enableOpenAPI, t('apps.openAPI.enableSuccess'))
}

function handleDisable() {
  updateConfig(disableOpenAPI, t('apps.openAPI.disableSuccess'))
}

function handleRefreshToken() {
  updateConfig(refreshOpenAPIToken, t('apps.openAPI.refreshSuccess'))
}

async function copyText(value: string) {
  try {
    await navigator.clipboard.writeText(value)
    message.success(t('apps.openAPI.copySuccess'))
  }
  catch {
    message.error(t('apps.openAPI.copyFailed'))
  }
}

onMounted(loadConfig)
</script>

<template>
  <div class="h-full overflow-auto rounded-lg bg-slate-200 p-2 dark:bg-zinc-900">
    <NCard size="small" class="open-api-card">
      <NAlert type="info" :show-icon="true">
        {{ t('apps.openAPI.description') }}
        <a
          class="ml-1 text-blue-600 hover:underline dark:text-blue-400"
          href="https://github.com/hslr-s/sun-panel-browser-extension"
          target="_blank"
          rel="noopener noreferrer"
        >{{ t('apps.openAPI.extensionLink') }}</a>
      </NAlert>

      <section class="py-5">
        <div class="mb-2 text-base font-medium">
          {{ t('apps.openAPI.status') }}
        </div>
        <NTag :type="config.enabled ? 'success' : 'error'" size="small">
          {{ config.enabled ? t('apps.openAPI.enabled') : t('apps.openAPI.disabled') }}
        </NTag>
      </section>

      <section class="border-y border-slate-200 py-5 dark:border-zinc-700">
        <div class="mb-2 flex flex-wrap items-baseline gap-1">
          <span class="font-medium">{{ t('apps.openAPI.endpoint') }}</span>
          <span class="text-xs text-slate-400">{{ t('apps.openAPI.endpointHint') }}</span>
        </div>
        <NInput :value="endpoint" readonly>
          <template #suffix>
            <NButton
              quaternary
              circle
              size="small"
              :aria-label="t('apps.openAPI.copyEndpoint')"
              :title="t('apps.openAPI.copyEndpoint')"
              @click="copyText(endpoint)"
            >
              <template #icon>
                <SvgIcon icon="ion-copy" />
              </template>
            </NButton>
          </template>
        </NInput>

        <div class="mb-2 mt-5 font-medium">
          Token
        </div>
        <NInput :value="config.token" readonly :placeholder="t('apps.openAPI.noToken')">
          <template #suffix>
            <NButton
              quaternary
              circle
              size="small"
              :disabled="!config.token"
              :aria-label="t('apps.openAPI.copyToken')"
              :title="t('apps.openAPI.copyToken')"
              @click="copyText(config.token)"
            >
              <template #icon>
                <SvgIcon icon="ion-copy" />
              </template>
            </NButton>
          </template>
        </NInput>
      </section>

      <div class="flex flex-wrap justify-center gap-2 pt-5">
        <NPopconfirm
          v-if="config.enabled"
          :positive-text="t('common.confirm')"
          :negative-text="t('common.cancel')"
          @positive-click="handleRefreshToken"
        >
          <template #trigger>
            <NButton type="info" secondary :loading="loading">
              <template #icon>
                <SvgIcon icon="shuaxin" />
              </template>
              {{ t('apps.openAPI.refreshToken') }}
            </NButton>
          </template>
          {{ t('apps.openAPI.refreshWarning') }}
        </NPopconfirm>

        <NButton v-if="!config.enabled" type="success" :loading="loading" @click="handleEnable">
          {{ t('apps.openAPI.enable') }}
        </NButton>

        <NPopconfirm
          v-else
          :positive-text="t('common.confirm')"
          :negative-text="t('common.cancel')"
          @positive-click="handleDisable"
        >
          <template #trigger>
            <NButton type="error" secondary :loading="loading">
              {{ t('apps.openAPI.disable') }}
            </NButton>
          </template>
          {{ t('apps.openAPI.disableWarning') }}
        </NPopconfirm>
      </div>
    </NCard>
  </div>
</template>

<style scoped>
.open-api-card {
  min-height: 100%;
  border-radius: 8px;
}
</style>
