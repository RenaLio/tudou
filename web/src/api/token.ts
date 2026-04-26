import client from './client'
import type { ApiResponse, ListResponse, Token, TokenWithRelations, TokenStatus, TokenSettings, LoadBalanceStrategy } from '@/types'

export interface CreateTokenRequest {
  groupID: string
  name: string
  limit?: number
  expiresAt?: string
  loadBalanceStrategy?: LoadBalanceStrategy
  settings?: TokenSettings
}

export interface UpdateTokenRequest {
  groupID?: string
  name?: string
  status?: TokenStatus
  limit?: number
  expiresAt?: string | null
  loadBalanceStrategy?: LoadBalanceStrategy
  settings?: TokenSettings
}

export interface ListTokensParams {
  page?: number
  pageSize?: number
  orderBy?: string
  keyword?: string
  groupID?: string
  status?: TokenStatus
  onlyAvailable?: boolean
  preloadGroup?: boolean
  preloadStats?: boolean
}

export async function listTokens(params: ListTokensParams = {}) {
  const response = await client.get<ApiResponse<ListResponse<TokenWithRelations>>>('/token', {
    params: {
      page: params.page ?? 1,
      pageSize: params.pageSize ?? 20,
      ...params,
    },
  })
  return response.data.data
}

export async function getToken(id: string) {
  const response = await client.get<ApiResponse<TokenWithRelations>>(`/token/${id}`)
  return response.data.data
}

export async function createToken(data: CreateTokenRequest) {
  const response = await client.post<ApiResponse<Token>>('/token', data)
  return response.data.data
}

export async function updateToken(id: string, data: UpdateTokenRequest) {
  const response = await client.put<ApiResponse<Token>>(`/token/${id}`, data)
  return response.data.data
}

export async function deleteToken(id: string) {
  await client.delete(`/token/${id}`)
}

export async function setTokenStatus(id: string, status: TokenStatus) {
  const response = await client.patch<ApiResponse<Token>>(`/token/${id}/status`, { status })
  return response.data.data
}

// Token status labels
export const TOKEN_STATUS_LABELS: Record<TokenStatus, { label: string; color: string }> = {
  enabled: { label: '启用', color: 'success' },
  disabled: { label: '禁用', color: 'warning' },
  expired: { label: '过期', color: 'danger' },
}

// Load balance strategy labels
export const LOAD_BALANCE_STRATEGY_LABELS: Record<LoadBalanceStrategy, string> = {
  random: '随机',
  performance: '综合性能优先',
  ttft_first: '响应时间优先',
  tps_first: 'TPS优先',
  success_first: '成功率优先',
  cost_first: '成本优先',
  weighted: '加权',
  least_conn: '最少连接',
}

// Format token for display (show first 8 and last 4 chars)
export function formatToken(token: string): string {
  if (!token || token.length < 12) return token
  return `${token.slice(0, 8)}...${token.slice(-4)}`
}

// Copy to clipboard
export async function copyToClipboard(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text)
    return true
  } catch {
    return false
  }
}
