import type {
  ApiResponse,
  Channel,
  ChannelExtra,
  ChannelSettings,
  ChannelStatus,
  ChannelType,
  ListResponse,
} from '@/types';

export type { ChannelExtra, };
import client from './client';

export interface CreateChannelRequest {
  type: ChannelType;
  name: string;
  baseURL: string;
  apiKey: string;
  weight?: number;
  remark?: string;
  tag?: string;
  model?: string;
  customModel?: string;
  priceRate?: number;
  expiredAt?: string;
  groupIDs?: string[];
  extra?: ChannelExtra;
}

export interface UpdateChannelRequest {
  type?: ChannelType;
  name?: string;
  baseURL?: string;
  apiKey?: string;
  weight?: number;
  status?: ChannelStatus;
  remark?: string;
  tag?: string;
  model?: string;
  customModel?: string;
  priceRate?: number;
  expiredAt?: string | null;
  groupIDs?: string[];
  extra?: ChannelExtra;
}

export interface ListChannelsParams {
  page?: number;
  pageSize?: number;
  orderBy?: string;
  keyword?: string;
  groupID?: string;
  type?: ChannelType;
  status?: ChannelStatus;
  onlyAvailable?: boolean;
  preloadGroups?: boolean;
  preloadStats?: boolean;
}

export async function listChannels(params: ListChannelsParams = {},) {
  const response = await client.get<ApiResponse<ListResponse<Channel>>>('/channel', {
    params: {
      page: params.page ?? 1,
      pageSize: params.pageSize ?? 20,
      ...params,
    },
  },);
  return response.data.data;
}

export async function getChannel(id: string,) {
  const response = await client.get<ApiResponse<Channel>>(`/channel/${id}`,);
  return response.data.data;
}

export async function createChannel(data: CreateChannelRequest,) {
  const response = await client.post<ApiResponse<Channel>>('/channel', data,);
  return response.data.data;
}

export async function updateChannel(id: string, data: UpdateChannelRequest,) {
  const response = await client.put<ApiResponse<Channel>>(`/channel/${id}`, data,);
  return response.data.data;
}

export async function deleteChannel(id: string,) {
  await client.delete(`/channel/${id}`,);
}

export async function setChannelStatus(id: string, status: ChannelStatus,) {
  const response = await client.patch<ApiResponse<Channel>>(`/channel/${id}/status`, { status, },);
  return response.data.data;
}

export interface FetchModelsRequest {
  type: ChannelType;
  baseURL: string;
  apiKey: string;
}

export async function fetchModels(data: FetchModelsRequest,) {
  const response = await client.post<ApiResponse<string[]>>('/channel/fetch-model', data,);
  return response.data.data;
}

// Platform options from backend
export interface PlatformOptionExtra {
  exampleBaseUrl?: string;
  paths?: Record<string, string>;
}

export interface PlatformOption {
  key: string;
  value: string;
  extra?: PlatformOptionExtra;
}

export interface PlatformOptionsResponse {
  options: PlatformOption[];
}

export async function getPlatformOptions() {
  const response = await client.get<ApiResponse<PlatformOptionsResponse>>('/select-option/platform_options',);
  return response.data.data;
}

// Default channel type labels (for backward compatibility)
export const CHANNEL_TYPE_LABELS: Record<string, string> = {
  openai: 'OpenAI',
  claude: 'Anthropic Claude',
  azure: 'Azure OpenAI',
  custom: '自定义',
};

// Get type label from platform options or fallback to defaults
export function getChannelTypeLabel(type: string, platformOptions?: PlatformOption[],): string {
  if (platformOptions) {
    const opt = platformOptions.find(o => o.value === type);
    if (opt) return opt.key;
  }
  return CHANNEL_TYPE_LABELS[type] ?? type;
}

// Preset colors for known channel types
const CHANNEL_TYPE_COLORS: Record<string, { bg: string; text: string; badge: string; }> = {
  openai: { bg: 'bg-[#10a37f]', text: 'text-[#10a37f]', badge: 'bg-[rgba(16,163,127,0.1)]', },
  claude: { bg: 'bg-[#d97706]', text: 'text-[#d97706]', badge: 'bg-[rgba(217,119,6,0.1)]', },
  azure: { bg: 'bg-[#0078d4]', text: 'text-[#0078d4]', badge: 'bg-[rgba(0,120,212,0.1)]', },
  custom: { bg: 'bg-gray-500', text: 'text-text-secondary', badge: 'bg-bg-tertiary', },
};

// Fallback color palette for dynamic types
const FALLBACK_COLORS = [
  { bg: 'bg-violet-500', text: 'text-violet-500', badge: 'bg-[rgba(139,92,246,0.1)]', },
  { bg: 'bg-cyan-500', text: 'text-cyan-500', badge: 'bg-[rgba(6,182,212,0.1)]', },
  { bg: 'bg-rose-500', text: 'text-rose-500', badge: 'bg-[rgba(244,63,94,0.1)]', },
  { bg: 'bg-amber-500', text: 'text-amber-500', badge: 'bg-[rgba(245,158,11,0.1)]', },
  { bg: 'bg-emerald-500', text: 'text-emerald-500', badge: 'bg-[rgba(16,185,129,0.1)]', },
  { bg: 'bg-sky-500', text: 'text-sky-500', badge: 'bg-[rgba(14,165,233,0.1)]', },
];

function hashStr(s: string,): number {
  let h = 0;
  for (let i = 0; i < s.length; i++) {
    h = ((h << 5) - h + s.charCodeAt(i,)) | 0;
  }
  return Math.abs(h,);
}

export function getChannelTypeColor(type: string,): { bg: string; text: string; badge: string; } {
  if (CHANNEL_TYPE_COLORS[type]) return CHANNEL_TYPE_COLORS[type]!;
  return FALLBACK_COLORS[hashStr(type,) % FALLBACK_COLORS.length]!;
}

// Channel status labels
export const CHANNEL_STATUS_LABELS: Record<ChannelStatus, { label: string; color: string; }> = {
  enabled: { label: '启用', color: 'success', },
  disabled: { label: '禁用', color: 'warning', },
  expired: { label: '过期', color: 'danger', },
};
