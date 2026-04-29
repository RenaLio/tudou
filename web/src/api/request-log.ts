import type { ApiResponse, ListResponse, ModelPricing, } from '@/types';
import client from './client';

export type RequestLogStatus = 'success' | 'fail';

export interface RequestLogExtra {
  requestBody?: string;
  responseBody?: string;
  headers?: Record<string, string>;
  ip?: string;
  userAgent?: string;
  requestPath?: string;
  retryTrace?: Array<{
    channelID?: string;
    channelName?: string;
    upstreamModel?: string;
    statusCode?: number;
    statusBody?: string;
  }>;
}

export interface ProviderDetail {
  provider?: string;
  requestFormat?: string;
  transFormat?: string;
}

export interface RequestLogResponse {
  id: string;
  requestId: string;
  userId: string;
  tokenId: string;
  tokenName: string;
  groupId: string;
  groupName: string;
  channelId: string;
  channelName: string;
  channelPriceRate: number;
  model: string;
  upstreamModel: string;
  inputToken: number;
  outputToken: number;
  cachedCreationInputTokens: number;
  cachedReadInputTokens: number;
  pricing: ModelPricing;
  costMicros: number;
  status: RequestLogStatus;
  ttft: number;
  transferTime: number;
  errorCode?: string;
  errorMsg?: string;
  isStream: boolean;
  extra: RequestLogExtra;
  providerDetail: ProviderDetail;
  createdAt: string;
}

export interface ListRequestLogsParams {
  page?: number;
  pageSize?: number;
  orderBy?: string;
  keyword?: string;
  requestId?: string;
  userId?: string;
  tokenId?: string;
  groupId?: string;
  channelId?: string;
  model?: string;
  upstreamModel?: string;
  status?: RequestLogStatus;
  isStream?: boolean;
  dateFrom?: string;
  dateTo?: string;
}

export async function listRequestLogs(params: ListRequestLogsParams = {},) {
  const response = await client.get<ApiResponse<ListResponse<RequestLogResponse>>>('/request-log', {
    params: {
      page: params.page ?? 1,
      pageSize: params.pageSize ?? 20,
      ...params,
    },
  },);
  return response.data.data;
}

export async function getRequestLogById(id: string,) {
  const response = await client.get<ApiResponse<RequestLogResponse>>(`/request-log/${id}`,);
  return response.data.data;
}

export function formatMicrosCost(micros: number,): string {
  return `$${(micros / 1_000_000).toFixed(4,)}`;
}
