import client from './client'
import type { ApiResponse, ListResponse, Channel, ChannelType, ChannelStatus, ChannelSettings, ChannelExtra } from '@/types'

export interface CreateChannelRequest {
  type: ChannelType
  name: string
  baseURL: string
  apiKey: string
  weight?: number
  remark?: string
  tag?: string
  model?: string
  customModel?: string
  priceRate?: number
  expiredAt?: string
  groupIDs?: string[]
}

export interface UpdateChannelRequest {
  type?: ChannelType
  name?: string
  baseURL?: string
  apiKey?: string
  weight?: number
  status?: ChannelStatus
  remark?: string
  tag?: string
  model?: string
  customModel?: string
  priceRate?: number
  expiredAt?: string | null
  groupIDs?: string[]
}

export interface ListChannelsParams {
  page?: number
  pageSize?: number
  orderBy?: string
  keyword?: string
  groupID?: string
  type?: ChannelType
  status?: ChannelStatus
  onlyAvailable?: boolean
  preloadGroups?: boolean
  preloadStats?: boolean
}

export async function listChannels(params: ListChannelsParams = {}) {
  const response = await client.get<ApiResponse<ListResponse<Channel>>>('/channel', {
    params: {
      page: params.page ?? 1,
      pageSize: params.pageSize ?? 20,
      ...params,
    },
  })
  return response.data.data
}

export async function getChannel(id: string) {
  const response = await client.get<ApiResponse<Channel>>(`/channel/${id}`)
  return response.data.data
}

export async function createChannel(data: CreateChannelRequest) {
  const response = await client.post<ApiResponse<Channel>>('/channel', data)
  return response.data.data
}

export async function updateChannel(id: string, data: UpdateChannelRequest) {
  const response = await client.put<ApiResponse<Channel>>(`/channel/${id}`, data)
  return response.data.data
}

export async function deleteChannel(id: string) {
  await client.delete(`/channel/${id}`)
}

export async function setChannelStatus(id: string, status: ChannelStatus) {
  const response = await client.patch<ApiResponse<Channel>>(`/channel/${id}/status`, { status })
  return response.data.data
}

export interface FetchModelsRequest {
  type: ChannelType
  baseURL: string
  apiKey: string
}

export async function fetchModels(data: FetchModelsRequest) {
  const response = await client.post<ApiResponse<string[]>>('/channel/fetch-model', data)
  return response.data.data
}

// Channel type labels
export const CHANNEL_TYPE_LABELS: Record<ChannelType, string> = {
  openai: 'OpenAI',
  claude: 'Anthropic Claude',
  azure: 'Azure OpenAI',
  custom: '自定义',
}

// Channel status labels
export const CHANNEL_STATUS_LABELS: Record<ChannelStatus, { label: string; color: string }> = {
  enabled: { label: '启用', color: 'success' },
  disabled: { label: '禁用', color: 'warning' },
  expired: { label: '过期', color: 'danger' },
}
