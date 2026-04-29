import type { ApiResponse, ChannelGroup, ListResponse, } from '@/types';
import client from './client';

export interface CreateChannelGroupRequest {
  name: string;
  nameRemark?: string;
  loadBalanceStrategy?: string;
}

export interface UpdateChannelGroupRequest {
  name?: string;
  nameRemark?: string;
  loadBalanceStrategy?: string;
}

export interface ListChannelGroupsParams {
  page?: number;
  pageSize?: number;
  keyword?: string;
}

export interface LoadBalanceStrategyItem {
  value: string;
  label: string;
  description: string;
}

export async function listChannelGroups(params: ListChannelGroupsParams = {},) {
  const response = await client.get<ApiResponse<ListResponse<ChannelGroup>>>('/channel-group', {
    params: {
      page: params.page ?? 1,
      pageSize: params.pageSize ?? 100,
      keyword: params.keyword,
    },
  },);
  return response.data.data;
}

export async function createChannelGroup(data: CreateChannelGroupRequest,) {
  const response = await client.post<ApiResponse<ChannelGroup>>('/channel-group', data,);
  return response.data.data;
}

export async function updateChannelGroup(id: string, data: UpdateChannelGroupRequest,) {
  const response = await client.put<ApiResponse<ChannelGroup>>(`/channel-group/${id}`, data,);
  return response.data.data;
}

export async function deleteChannelGroup(id: string,) {
  await client.delete(`/channel-group/${id}`,);
}

export async function listLoadBalanceStrategies() {
  const response = await client.get<ApiResponse<LoadBalanceStrategyItem[]>>('/channel-group/load-balance-strategies',);
  return response.data.data;
}
