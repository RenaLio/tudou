import type { AIModel, ApiResponse, ListResponse, ModelPricing, ModelPricingType, } from '@/types';
import client from './client';

export interface CreateAIModelRequest {
  name: string;
  description?: string;
  pricing?: ModelPricing;
  pricingType?: ModelPricingType;
}

export interface UpdateAIModelRequest {
  name?: string;
  description?: string;
  pricing?: ModelPricing;
  pricingType?: ModelPricingType;
}

export interface ListAIModelsParams {
  page?: number;
  pageSize?: number;
  orderBy?: string;
  keyword?: string;
}

export async function listAIModels(params: ListAIModelsParams = {},) {
  const response = await client.get<ApiResponse<ListResponse<AIModel>>>('/model', {
    params: {
      page: params.page ?? 1,
      pageSize: params.pageSize ?? 20,
      ...params,
    },
  },);
  return response.data.data;
}

export async function getAIModel(id: string,) {
  const response = await client.get<ApiResponse<AIModel>>(`/model/${id}`,);
  return response.data.data;
}

export async function createAIModel(data: CreateAIModelRequest,) {
  const response = await client.post<ApiResponse<AIModel>>('/model', data,);
  return response.data.data;
}

export async function updateAIModel(id: string, data: UpdateAIModelRequest,) {
  const response = await client.put<ApiResponse<AIModel>>(`/model/${id}`, data,);
  return response.data.data;
}

export async function deleteAIModel(id: string,) {
  await client.delete(`/model/${id}`,);
}

// Pricing type labels
export const PRICING_TYPE_LABELS: Record<ModelPricingType, string> = {
  tokens: '按 Token 计费',
  request: '按次计费',
};

// Format price per million tokens
export function formatPricePerMillion(price?: number,): string {
  if (price === undefined || price === null) return '-';
  return `$${price.toFixed(2,)}`;
}

// Format price for display
export function formatPricingSummary(pricing: ModelPricing, pricingType: ModelPricingType,): string {
  if (pricingType === 'request') {
    return pricing.perRequestPrice ? `$${pricing.perRequestPrice.toFixed(4,)}/次` : '-';
  }
  const parts: string[] = [];
  if (pricing.inputPrice) parts.push(`输入 $${pricing.inputPrice.toFixed(2,)}`,);
  if (pricing.outputPrice) parts.push(`输出 $${pricing.outputPrice.toFixed(2,)}`,);
  return parts.length > 0 ? parts.join(' / ',) : '-';
}
