import client from './client';

export interface RegistryChannel {
  id: string;
  name: string;
  type: string;
  baseURL: string;
  status: string;
  weight: number;
  priceRate: number;
  settings?: Record<string, unknown>;
  extra?: Record<string, unknown>;
  model?: string;
  customModel?: string;
  activeConns: number;
  lastUsedAt: number;
  successRate: number;
  [key: string]: unknown;
}

export interface RegistryEndpoint {
  channelId: string;
  channelType: string;
  model: string;
  upstreamModel: string;
  baseWeight: number;
  costRate: number;
  status: number;
  emaTTFT: number;
  emaTPS: number;
  emaSuccessRate: number;
  consecutiveFails: number;
  nextRetryTime: number;
  lastUsedAt: number;
}

export interface RegistryGroup {
  [key: string]: unknown;
}

export interface RegistryData {
  channels: Record<string, RegistryChannel>;
  groups: Record<string, RegistryGroup>;
  endpoints: Record<string, Record<string, RegistryEndpoint>>;
}

export async function getRegistry(): Promise<RegistryData> {
  const response = await client.get<RegistryData>('/_debug/registry');
  return response.data;
}

export const ENDPOINT_STATUS_LABELS: Record<number, { label: string; color: string }> = {
  0: { label: '健康', color: 'success' },
  1: { label: '亚健康', color: 'warning' },
  2: { label: '熔断', color: 'danger' },
};
