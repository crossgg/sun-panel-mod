import { post } from '@/utils/request'

export interface OpenAPIConfig {
  enabled: boolean
  token: string
  endpoint: string
}

export function getOpenAPIConfig() {
  return post<OpenAPIConfig>({
    url: '/openApi/getConfig',
  })
}

export function enableOpenAPI() {
  return post<OpenAPIConfig>({
    url: '/openApi/enable',
  })
}

export function disableOpenAPI() {
  return post<OpenAPIConfig>({
    url: '/openApi/disable',
  })
}

export function refreshOpenAPIToken() {
  return post<OpenAPIConfig>({
    url: '/openApi/refreshToken',
  })
}
