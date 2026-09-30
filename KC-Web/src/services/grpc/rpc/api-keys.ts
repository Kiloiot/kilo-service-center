/**
 * API key RPCs (IdentityService): request building and response mapping to
 * the transport DTO.
 */

import * as identityPb from "@services/grpc/identity_pb";
import { HTTP_STATUS } from "@constants/app";
import { GRPC_CLIENT_ERRORS } from "@constants/messages";

import { dateToTimestamp } from "../codec";
import { GrpcApiError } from "../errors";
import { grpcTransport } from "../transport";

/** gRPC-layer API key (timestamps as Date from protobuf Timestamp). */
export interface GrpcApiKey {
  id: string;
  orgId: string;
  userId: string;
  name: string;
  keyPrefix: string;
  keyType: string;
  isActive: boolean;
  expiresAt?: Date;
  lastUsedAt?: Date;
  createdAt: Date;
}

export interface ListApiKeysParams {
  pageSize?: number;
  pageToken?: string;
  userId?: string;
}

export interface CreateApiKeyInput {
  name: string;
  keyType: string;
  expiresAt?: Date;
}

function mapApiKey(k: identityPb.ApiKey): GrpcApiKey {
  return {
    id: k.getId(),
    orgId: k.getOrgId(),
    userId: k.getUserId(),
    name: k.getName(),
    keyPrefix: k.getKeyPrefix(),
    keyType: k.getKeyType(),
    isActive: k.getIsActive(),
    expiresAt: k.getExpiresAt()?.toDate(),
    lastUsedAt: k.getLastUsedAt()?.toDate(),
    createdAt: k.getCreatedAt()?.toDate() || new Date(),
  };
}

export async function listApiKeys(params?: ListApiKeysParams): Promise<{
  apiKeys: GrpcApiKey[];
  nextPageToken?: string;
  totalCount: number;
}> {
  const request = new identityPb.ListApiKeysRequest();
  if (params?.pageSize) request.setPageSize(params.pageSize);
  if (params?.pageToken) request.setPageToken(params.pageToken);
  if (params?.userId) request.setUserId(params.userId);

  const response = await grpcTransport.callIdentity<
    identityPb.ListApiKeysRequest,
    identityPb.ListApiKeysResponse
  >((c) => c.listApiKeys, request);

  return {
    apiKeys: response.getApiKeysList().map(mapApiKey),
    nextPageToken: response.getNextPageToken() || undefined,
    totalCount: response.getTotalCount(),
  };
}

/** The raw key is returned once, on creation, and never stored. */
export async function createApiKey(data: CreateApiKeyInput): Promise<{
  apiKey: GrpcApiKey;
  rawKey: string;
}> {
  const request = new identityPb.CreateApiKeyRequest();
  request.setName(data.name);
  request.setKeyType(data.keyType);
  if (data.expiresAt) request.setExpiresAt(dateToTimestamp(data.expiresAt));

  const response = await grpcTransport.callIdentity<
    identityPb.CreateApiKeyRequest,
    identityPb.CreateApiKeyResponse
  >((c) => c.createApiKey, request);

  const apiKey = response.getApiKey();
  if (!apiKey) {
    throw new GrpcApiError(
      HTTP_STATUS.INTERNAL_SERVER_ERROR,
      GRPC_CLIENT_ERRORS.API_KEY_CREATION_FAILED,
    );
  }
  return { apiKey: mapApiKey(apiKey), rawKey: response.getRawKey() };
}

export async function deleteApiKey(id: string): Promise<boolean> {
  const request = new identityPb.DeleteApiKeyRequest();
  request.setId(id);

  const response = await grpcTransport.callIdentity<
    identityPb.DeleteApiKeyRequest,
    identityPb.DeleteApiKeyResponse
  >((c) => c.deleteApiKey, request);

  return response.getSuccess();
}
