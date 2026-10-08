/**
 * API keys of the current organization, in the shapes the UI consumes.
 */

import type {
  ApiKeyAPI,
  ApiKeyCreateResponse,
  ApiKeyListResponse,
} from "@api-types/api";

import {
  createApiKey,
  type CreateApiKeyInput,
  deleteApiKey,
  type GrpcApiKey,
  listApiKeys,
  type ListApiKeysParams,
} from "@services/grpc/rpc/api-keys";

function toApiKeyAPI(k: GrpcApiKey): ApiKeyAPI {
  return {
    id: k.id,
    orgId: k.orgId,
    userId: k.userId,
    name: k.name,
    keyPrefix: k.keyPrefix,
    keyType: k.keyType,
    isActive: k.isActive,
    expiresAt: k.expiresAt?.toISOString(),
    lastUsedAt: k.lastUsedAt?.toISOString(),
    createdAt: k.createdAt?.toISOString() || new Date().toISOString(),
  };
}

export const apiKeysApi = {
  async list(params?: ListApiKeysParams): Promise<ApiKeyListResponse> {
    const response = await listApiKeys(params);
    return {
      apiKeys: response.apiKeys.map(toApiKeyAPI),
      nextPageToken: response.nextPageToken,
      totalCount: response.totalCount,
    };
  },

  /** The raw key is returned once, on creation, and never stored. */
  async create(data: CreateApiKeyInput): Promise<ApiKeyCreateResponse> {
    const response = await createApiKey(data);
    return { apiKey: toApiKeyAPI(response.apiKey), rawKey: response.rawKey };
  },

  delete(id: string): Promise<boolean> {
    return deleteApiKey(id);
  },
};
