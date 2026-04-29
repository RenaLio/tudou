import type { ApiResponse, LoginRequest, LoginResponse, } from '@/types';
import client from './client';

export async function login(data: LoginRequest,): Promise<LoginResponse> {
  const response = await client.post<ApiResponse<LoginResponse>>('/user/login', data,);
  return response.data.data;
}

export async function updatePassword(password: string,): Promise<void> {
  await client.patch('/self/password', { password, },);
}
