import type { ApiResponse, ListResponse, } from '@/types';
import client from './client';

// Types
export interface ObservationBucket15M {
  startAt: string;
  endAt: string;
  inputToken: number;
  outputToken: number;
  cachedCreationInputTokens: number;
  cachedReadInputTokens: number;
  requestSuccess: number;
  requestFailed: number;
  totalCostMicros: number;
  totalCost: number;
  avgTTFT: number;
  avgTPS: number;
}

export interface ObservationWindow3H {
  windowMinutes: number;
  bucketMinutes: number;
  buckets: ObservationBucket15M[];
}

export interface ChannelStatsResponse {
  channelID: string;
  channelName: string;
  inputToken: number;
  outputToken: number;
  cachedCreationInputTokens: number;
  cachedReadInputTokens: number;
  requestSuccess: number;
  requestFailed: number;
  totalCostMicros: number;
  totalCost: number;
  avgTTFT: number;
  avgTPS: number;
  window3h: ObservationWindow3H;
}

export interface ChannelModelStatsResponse {
  channelID: string;
  model: string;
  inputToken: number;
  outputToken: number;
  cachedCreationInputTokens: number;
  cachedReadInputTokens: number;
  requestSuccess: number;
  requestFailed: number;
  totalCostMicros: number;
  totalCost: number;
  avgTTFT: number;
  avgTPS: number;
  window3h: ObservationWindow3H;
}

export interface TokenStatsResponse {
  tokenID: string;
  tokenName: string;
  inputToken: number;
  outputToken: number;
  cachedCreationInputTokens: number;
  cachedReadInputTokens: number;
  requestSuccess: number;
  requestFailed: number;
  totalCostMicros: number;
  totalCost: number;
}

export interface UserStatsResponse {
  userID: string;
  inputToken: number;
  outputToken: number;
  cachedCreationInputTokens: number;
  cachedReadInputTokens: number;
  requestSuccess: number;
  requestFailed: number;
  totalCostMicros: number;
  totalCost: number;
}

export interface UserUsageDailyStatsResponse {
  id: string;
  userID: string;
  date: string;
  inputToken: number;
  outputToken: number;
  cachedCreationInputTokens: number;
  cachedReadInputTokens: number;
  requestSuccess: number;
  requestFailed: number;
  totalCostMicros: number;
  totalCost: number;
  createdAt: string;
  updatedAt: string;
}

export interface UserUsageHourlyStatsResponse {
  id: string;
  userID: string;
  date: string;
  hour: number;
  inputToken: number;
  outputToken: number;
  cachedCreationInputTokens: number;
  cachedReadInputTokens: number;
  requestSuccess: number;
  requestFailed: number;
  totalCostMicros: number;
  totalCost: number;
  createdAt: string;
  updatedAt: string;
}

// API functions
export async function getChannelStats(channelID: string,) {
  const response = await client.get<ApiResponse<ChannelStatsResponse>>(`/stats/channel/${channelID}`,);
  return response.data.data;
}

export async function listChannelStats() {
  const response = await client.get<ApiResponse<ChannelStatsResponse[]>>('/stats/channel',);
  return response.data.data;
}

export async function listChannelModelStats(channelID: string,) {
  const response = await client.get<ApiResponse<ChannelModelStatsResponse[]>>(`/stats/channel/${channelID}/model`,);
  return response.data.data;
}

export async function getTokenStats(tokenID: string,) {
  const response = await client.get<ApiResponse<TokenStatsResponse>>(`/stats/token/${tokenID}`,);
  return response.data.data;
}

export async function listTokenStats(ids: string[],) {
  const response = await client.get<ApiResponse<TokenStatsResponse[]>>('/stats/token', {
    params: { ids: ids.join(',',), },
  },);
  return response.data.data;
}

export async function getUserStats(userID: string,) {
  const response = await client.get<ApiResponse<UserStatsResponse>>(`/stats/user/${userID}`,);
  return response.data.data;
}

export interface ListUserUsageDailyStatsParams {
  page?: number;
  pageSize?: number;
  orderBy?: string;
  userID?: string;
  startTime?: string; // RFC3339, e.g. 2026-04-28T00:00:00+08:00
  endTime?: string; // RFC3339
}

export async function listUserUsageDailyStats(params: ListUserUsageDailyStatsParams = {},) {
  const response = await client.get<ApiResponse<ListResponse<UserUsageDailyStatsResponse>>>('/stats/user/usage/daily', {
    params: {
      page: params.page ?? 1,
      pageSize: params.pageSize ?? 30,
      ...params,
    },
  },);
  return response.data.data;
}

export interface ListUserUsageHourlyStatsParams {
  page?: number;
  pageSize?: number;
  orderBy?: string;
  userID?: string;
  startTime?: string; // RFC3339
  endTime?: string; // RFC3339
}

export async function listUserUsageHourlyStats(params: ListUserUsageHourlyStatsParams = {},) {
  const response = await client.get<ApiResponse<ListResponse<UserUsageHourlyStatsResponse>>>(
    '/stats/user/usage/hourly',
    {
      params: {
        page: params.page ?? 1,
        pageSize: params.pageSize ?? 24,
        ...params,
      },
    },
  );
  return response.data.data;
}

// Utility functions
export function formatTokens(tokens: number,): string {
  if (tokens >= 1_000_000) {
    return `${(tokens / 1_000_000).toFixed(2,)}M`;
  }
  if (tokens >= 1_000) {
    return `${(tokens / 1_000).toFixed(1,)}K`;
  }
  return tokens.toString();
}

export function formatCost(micros: number,): string {
  const dollars = micros / 1_000_000;
  if (dollars >= 1000) {
    return `$${(dollars / 1000).toFixed(2,)}K`;
  }
  return `$${dollars.toFixed(2,)}`;
}

export function formatNumber(num: number,): string {
  if (num >= 1_000_000) {
    return `${(num / 1_000_000).toFixed(1,)}M`;
  }
  if (num >= 1_000) {
    return `${(num / 1_000).toFixed(1,)}K`;
  }
  return num.toString();
}

export function calcSuccessRate(success: number, failed: number,): number {
  const total = success + failed;
  if (total === 0) return 100;
  return Math.round((success / total) * 100,);
}
