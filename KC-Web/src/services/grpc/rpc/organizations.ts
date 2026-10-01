/**
 * Organization and membership RPCs (IdentityService).
 */

import { FieldMask } from "google-protobuf/google/protobuf/field_mask_pb";

import * as identityPb from "@services/grpc/identity_pb";
import { HTTP_STATUS, ORG_USER_UPDATE_MASK_PATHS } from "@constants/app";
import { GRPC_CLIENT_ERRORS } from "@constants/messages";

import { GrpcApiError } from "../errors";
import { grpcTransport } from "../transport";

/**
 * List organizations
 */
export async function listOrganizations(params?: {
  pageSize?: number;
  pageToken?: string;
}): Promise<{
  organizations: Array<{
    id: string;
    name: string;
    description?: string;
    tenantId: string;
    state?: string;
    createdAt?: Date;
    updatedAt?: Date;
  }>;
  nextPageToken?: string;
  totalCount: number;
}> {
  const request = new identityPb.ListOrganizationsRequest();
  if (params?.pageSize) request.setPageSize(params.pageSize);
  if (params?.pageToken) request.setPageToken(params.pageToken);

  const response = await grpcTransport.callIdentity<
    identityPb.ListOrganizationsRequest,
    identityPb.ListOrganizationsResponse
  >((c) => c.listOrganizations, request, { requireOrgUser: false });

  return {
    organizations: response.getOrganizationsList().map((o) => ({
      id: o.getId(),
      name: o.getName(),
      description: o.getDescription() || undefined,
      tenantId: String(o.getTenantId()),
      state: o.getState(),
      createdAt: o.getCreatedAt()?.toDate(),
      updatedAt: o.getUpdatedAt()?.toDate(),
    })),
    nextPageToken: response.getNextPageToken() || undefined,
    totalCount: response.getTotalCount(),
  };
}

/**
 * Get organization by ID
 */
export async function getOrganization(orgId: string): Promise<{
  id: string;
  name: string;
  description?: string;
  tenantId: string;
  state?: string;
  createdAt?: Date;
  updatedAt?: Date;
} | null> {
  const request = new identityPb.GetOrganizationRequest();
  request.setId(orgId);

  try {
    const response = await grpcTransport.callIdentity<
      identityPb.GetOrganizationRequest,
      identityPb.GetOrganizationResponse
    >((c) => c.getOrganization, request, { requireOrgUser: false });

    const org = response.getOrganization();
    if (!org) return null;

    return {
      id: org.getId(),
      name: org.getName(),
      description: org.getDescription() || undefined,
      tenantId: String(org.getTenantId()),
      state: org.getState(),
      createdAt: org.getCreatedAt()?.toDate(),
      updatedAt: org.getUpdatedAt()?.toDate(),
    };
  } catch (error) {
    if (error instanceof GrpcApiError && error.isNotFound()) {
      return null;
    }
    throw error;
  }
}

/**
 * Create organization
 */
export async function createOrganization(data: {
  name: string;
  description?: string;
}): Promise<{
  id: string;
  name: string;
  description?: string;
  tenantId: string;
}> {
  const request = new identityPb.CreateOrganizationRequest();
  request.setName(data.name);
  if (data.description) request.setDescription(data.description);

  const response = await grpcTransport.callIdentity<
    identityPb.CreateOrganizationRequest,
    identityPb.CreateOrganizationResponse
  >((c) => c.createOrganization, request, { requireOrgUser: false });

  const org = response.getOrganization();
  if (!org) {
    throw new GrpcApiError(
      HTTP_STATUS.INTERNAL_SERVER_ERROR,
      GRPC_CLIENT_ERRORS.INVALID_CREATE_ORGANIZATION_RESPONSE,
    );
  }

  return {
    id: org.getId(),
    name: org.getName(),
    description: org.getDescription() || undefined,
    tenantId: String(org.getTenantId()),
  };
}

/**
 * Update organization
 */
export async function updateOrganization(
  orgId: string,
  data: {
    name?: string;
    description?: string;
  },
): Promise<{
  id: string;
  name: string;
  description?: string;
  tenantId: string;
}> {
  const request = new identityPb.UpdateOrganizationRequest();
  request.setId(orgId);
  if (data.name) request.setName(data.name);
  if (data.description !== undefined) request.setDescription(data.description);

  const response = await grpcTransport.callIdentity<
    identityPb.UpdateOrganizationRequest,
    identityPb.UpdateOrganizationResponse
  >((c) => c.updateOrganization, request, { requireOrgUser: false });

  const org = response.getOrganization();
  if (!org) {
    throw new GrpcApiError(
      HTTP_STATUS.INTERNAL_SERVER_ERROR,
      GRPC_CLIENT_ERRORS.INVALID_UPDATE_ORGANIZATION_RESPONSE,
    );
  }

  return {
    id: org.getId(),
    name: org.getName(),
    description: org.getDescription() || undefined,
    tenantId: String(org.getTenantId()),
  };
}

/**
 * Delete organization
 */
export async function deleteOrganization(orgId: string): Promise<void> {
  const request = new identityPb.DeleteOrganizationRequest();
  request.setId(orgId);

  await grpcTransport.callIdentity<
    identityPb.DeleteOrganizationRequest,
    identityPb.DeleteOrganizationResponse
  >((c) => c.deleteOrganization, request, { requireOrgUser: false });
}

/**
 * Add user to organization
 */
export async function addOrganizationUser(
  orgId: string,
  data: {
    userId?: string;
    email?: string;
    role: string;
    isOrgAdmin?: boolean;
    isBaseStationAdmin?: boolean;
    isEndpointAdmin?: boolean;
  },
): Promise<{
  userId: string;
  orgId: string;
  email: string;
  role: string;
  status: string;
  isOrgAdmin?: boolean;
  isBaseStationAdmin?: boolean;
  isEndpointAdmin?: boolean;
  createdAt?: Date;
  updatedAt?: Date;
}> {
  const request = new identityPb.AddOrganizationUserRequest();
  request.setOrgId(orgId);
  if (data.userId) request.setUserId(data.userId);
  if (data.email) request.setEmail(data.email);
  request.setRole(data.role);
  if (data.isOrgAdmin !== undefined) request.setIsOrgAdmin(data.isOrgAdmin);
  if (data.isBaseStationAdmin !== undefined)
    request.setIsBaseStationAdmin(data.isBaseStationAdmin);
  if (data.isEndpointAdmin !== undefined)
    request.setIsEndpointAdmin(data.isEndpointAdmin);

  const response = await grpcTransport.callIdentity<
    identityPb.AddOrganizationUserRequest,
    identityPb.AddOrganizationUserResponse
  >((c) => c.addOrganizationUser, request, { requireOrgUser: false });

  const member = response.getMember();
  if (!member) {
    throw new GrpcApiError(
      HTTP_STATUS.INTERNAL_SERVER_ERROR,
      GRPC_CLIENT_ERRORS.INVALID_ADD_ORG_USER_RESPONSE,
    );
  }

  return {
    userId: member.getUserId(),
    orgId: member.getOrgId(),
    email: member.getEmail(),
    role: member.getRole(),
    status: member.getStatus(),
    isOrgAdmin: member.getIsOrgAdmin(),
    isBaseStationAdmin: member.getIsBaseStationAdmin(),
    isEndpointAdmin: member.getIsEndpointAdmin(),
    createdAt: member.getCreatedAt()?.toDate(),
    updatedAt: member.getUpdatedAt()?.toDate(),
  };
}

/**
 * Get organization user
 */
export async function getOrganizationUser(
  orgId: string,
  userId: string,
): Promise<{
  userId: string;
  orgId: string;
  role: string;
  status: string;
  email: string;
  isOrgAdmin?: boolean;
  isBaseStationAdmin?: boolean;
  isEndpointAdmin?: boolean;
  createdAt?: Date;
  updatedAt?: Date;
} | null> {
  const request = new identityPb.GetOrganizationUserRequest();
  request.setOrgId(orgId);
  request.setUserId(userId);

  try {
    const response = await grpcTransport.callIdentity<
      identityPb.GetOrganizationUserRequest,
      identityPb.GetOrganizationUserResponse
    >((c) => c.getOrganizationUser, request, { requireOrgUser: false });

    const member = response.getMember();
    if (!member) return null;

    return {
      userId: member.getUserId(),
      orgId: member.getOrgId(),
      role: member.getRole(),
      status: member.getStatus(),
      email: member.getEmail(),
      isOrgAdmin: member.getIsOrgAdmin(),
      isBaseStationAdmin: member.getIsBaseStationAdmin(),
      isEndpointAdmin: member.getIsEndpointAdmin(),
      createdAt: member.getCreatedAt()?.toDate(),
      updatedAt: member.getUpdatedAt()?.toDate(),
    };
  } catch (error) {
    if (error instanceof GrpcApiError && error.isNotFound()) {
      return null;
    }
    throw error;
  }
}

/**
 * List organization users
 */
export async function listOrganizationUsers(
  orgId: string,
  params?: {
    pageSize?: number;
    pageToken?: string;
    status?: string;
  },
): Promise<{
  users: Array<{
    userId: string;
    orgId: string;
    role: string;
    status: string;
    email: string;
    isOrgAdmin?: boolean;
    isBaseStationAdmin?: boolean;
    isEndpointAdmin?: boolean;
    createdAt?: Date;
    updatedAt?: Date;
  }>;
  nextPageToken?: string;
  totalCount: number;
}> {
  const request = new identityPb.ListOrganizationUsersRequest();
  request.setOrgId(orgId);
  if (params?.pageSize) request.setPageSize(params.pageSize);
  if (params?.pageToken) request.setPageToken(params.pageToken);
  if (params?.status) request.setStatus(params.status);

  const response = await grpcTransport.callIdentity<
    identityPb.ListOrganizationUsersRequest,
    identityPb.ListOrganizationUsersResponse
  >((c) => c.listOrganizationUsers, request, { requireOrgUser: false });

  return {
    users: response.getMembersList().map((u) => ({
      userId: u.getUserId(),
      orgId: u.getOrgId(),
      role: u.getRole(),
      status: u.getStatus(),
      email: u.getEmail(),
      isOrgAdmin: u.getIsOrgAdmin(),
      isBaseStationAdmin: u.getIsBaseStationAdmin(),
      isEndpointAdmin: u.getIsEndpointAdmin(),
      createdAt: u.getCreatedAt()?.toDate(),
      updatedAt: u.getUpdatedAt()?.toDate(),
    })),
    nextPageToken: response.getNextPageToken() || undefined,
    totalCount: response.getTotalCount(),
  };
}

/**
 * Remove user from organization
 */
export async function removeOrganizationUser(
  orgId: string,
  userId: string,
): Promise<void> {
  const request = new identityPb.RemoveOrganizationUserRequest();
  request.setOrgId(orgId);
  request.setUserId(userId);

  await grpcTransport.callIdentity<
    identityPb.RemoveOrganizationUserRequest,
    identityPb.RemoveOrganizationUserResponse
  >((c) => c.removeOrganizationUser, request, { requireOrgUser: false });
}

/**
 * Update organization user with FieldMask for partial updates
 */
export async function updateOrganizationUser(
  orgId: string,
  userId: string,
  data: {
    role?: string;
    isOrgAdmin?: boolean;
    isBaseStationAdmin?: boolean;
    isEndpointAdmin?: boolean;
  },
): Promise<{
  userId: string;
  orgId: string;
  email: string;
  role: string;
  status: string;
  isOrgAdmin?: boolean;
  isBaseStationAdmin?: boolean;
  isEndpointAdmin?: boolean;
  createdAt?: Date;
  updatedAt?: Date;
}> {
  const request = new identityPb.UpdateOrganizationUserRequest();
  request.setOrgId(orgId);
  request.setUserId(userId);

  const paths: string[] = [];
  if (data.role !== undefined) {
    request.setRole(data.role);
    paths.push(ORG_USER_UPDATE_MASK_PATHS.ROLE);
  }
  if (data.isOrgAdmin !== undefined) {
    request.setIsOrgAdmin(data.isOrgAdmin);
    paths.push(ORG_USER_UPDATE_MASK_PATHS.IS_ORG_ADMIN);
  }
  if (data.isBaseStationAdmin !== undefined) {
    request.setIsBaseStationAdmin(data.isBaseStationAdmin);
    paths.push(ORG_USER_UPDATE_MASK_PATHS.IS_BASE_STATION_ADMIN);
  }
  if (data.isEndpointAdmin !== undefined) {
    request.setIsEndpointAdmin(data.isEndpointAdmin);
    paths.push(ORG_USER_UPDATE_MASK_PATHS.IS_ENDPOINT_ADMIN);
  }

  const mask = new FieldMask();
  mask.setPathsList(paths);
  request.setUpdateMask(mask);

  const response = await grpcTransport.callIdentity<
    identityPb.UpdateOrganizationUserRequest,
    identityPb.UpdateOrganizationUserResponse
  >((c) => c.updateOrganizationUser, request, { requireOrgUser: false });

  const member = response.getMember();
  if (!member) {
    throw new GrpcApiError(
      HTTP_STATUS.INTERNAL_SERVER_ERROR,
      GRPC_CLIENT_ERRORS.INVALID_UPDATE_ORG_USER_RESPONSE,
    );
  }

  return {
    userId: member.getUserId(),
    orgId: member.getOrgId(),
    email: member.getEmail(),
    role: member.getRole(),
    status: member.getStatus(),
    isOrgAdmin: member.getIsOrgAdmin(),
    isBaseStationAdmin: member.getIsBaseStationAdmin(),
    isEndpointAdmin: member.getIsEndpointAdmin(),
    createdAt: member.getCreatedAt()?.toDate(),
    updatedAt: member.getUpdatedAt()?.toDate(),
  };
}
