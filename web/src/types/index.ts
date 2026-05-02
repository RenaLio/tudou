// Common API response wrapper
export interface ApiResponse<T,> {
  code: number;
  message: string;
  data: T;
}

// Paginated list response
export interface ListResponse<T,> {
  total: number;
  items: T[];
  page: number;
  pageSize: number;
}

// Login
export interface LoginRequest {
  username: string;
  password: string;
}

export interface LoginResponse {
  accessToken: string;
  expiresIn: number;
  user: User;
}

// User
export interface User {
  id: string;
  username: string;
  email: string;
  phone: string;
  nickname: string;
  avatar: string;
  status: 'enabled' | 'disabled' | 'locked';
  role: 'admin' | 'user' | 'guest';
  lastLoginAt: string | null;
  lastLoginIP: string;
  loginCount: number;
  createdAt: string;
  updatedAt: string;
}

// Token (API Key)
export type TokenStatus = 'enabled' | 'disabled' | 'expired';

export interface TokenSettings {
  maxRequestsPerMinute?: number;
  maxTokensPerRequest?: number;
  allowedModels?: string[];
  deniedModels?: string[];
}

export interface Token {
  id: string;
  userID: string;
  groupID: string;
  token: string;
  name: string;
  status: TokenStatus;
  limit: number;
  expiresAt: string | null;
  loadBalanceStrategy: LoadBalanceStrategy;
  settings: TokenSettings;
  createdAt: string;
  updatedAt: string;
}

export interface TokenWithRelations extends Token {
  user?: User;
  group?: ChannelGroup;
  stats?: TokenStats;
}

export interface TokenStats {
  tokenID: string;
  tokenName: string;
  inputToken: number;
  outputToken: number;
  requestSuccess: number;
  requestFailed: number;
  totalCostMicros: number;
  avgTTFT: number;
  avgTPS: number;
}

// Channel
export type ChannelType = string;
export type ChannelStatus = 'enabled' | 'disabled' | 'expired';

export interface ChannelSettings {
  timeout?: number;
  maxRetries?: number;
  retryInterval?: number;
  enableStream?: boolean;
  streamTimeout?: number;
  maxTokens?: number;
  defaultTemperature?: number;
  circuitThreshold?: number;
  circuitTimeout?: number;
  maxConcurrent?: number;
}

export interface ChannelExtra {
  headers?: Record<string, string>;
  description?: string;
  docsURL?: string;
  region?: string;
  tier?: string;
  modelMappings?: Record<string, string>;
}

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

export interface ChannelStats {
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
  window3h?: ObservationWindow3H;
}

export interface Channel {
  id: string;
  type: ChannelType;
  name: string;
  baseURL: string;
  apiKey: string;
  weight: number;
  status: ChannelStatus;
  remark: string;
  tag: string;
  model: string;
  customModel: string;
  settings: ChannelSettings;
  extra: ChannelExtra;
  priceRate: number;
  expiredAt: string | null;
  createdAt: string;
  updatedAt: string;
  groupIDs?: string[];
  groups?: ChannelGroup[];
  stats?: ChannelStats;
}

// Channel Group
export interface ChannelGroup {
  id: string;
  name: string;
  nameRemark: string;
  loadBalanceStrategy: LoadBalanceStrategy;
  createdAt: string;
  updatedAt: string;
}

export type LoadBalanceStrategy =
  | 'random'
  | 'performance'
  | 'ttft_first'
  | 'tps_first'
  | 'success_first'
  | 'cost_first'
  | 'weighted'
  | 'least_conn';

// AI Model
export type ModelType = 'chat' | 'embedding' | 'image' | 'audio' | 'multi';
export type ModelPricingType = 'tokens' | 'request';

export interface ModelPricing {
  inputPrice?: number;
  outputPrice?: number;
  cacheCreatePrice?: number;
  cacheReadPrice?: number;
  perRequestPrice?: number;
  over200KInputPrice?: number;
  over200KOutputPrice?: number;
  over200KCacheCreatePrice?: number;
  over200KCacheReadPrice?: number;
  over200KPerRequestPrice?: number;
}

export interface AIModelExtra {
  syncModelInfoPath?: string;
  disableSync?: boolean;
}

export interface AIModel {
  id: string;
  name: string;
  type: ModelType;
  description: string;
  pricing: ModelPricing;
  pricingType: ModelPricingType;
  isEnabled: boolean;
  extra: AIModelExtra;
  createdAt: string;
  updatedAt: string;
}

// System Config
export interface SystemConfig {
  id: string;
  key: string;
  value: unknown;
  type: string;
  scope: 'system' | 'user';
  description: string;
  isEditable: boolean;
  isVisible: boolean;
  sort: number;
  createdAt: string;
  updatedAt: string;
}
