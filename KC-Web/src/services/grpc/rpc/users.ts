/**
 * User administration RPCs (IdentityService).
 */

import { FieldMask } from "google-protobuf/google/protobuf/field_mask_pb";

import * as identityPb from "@services/grpc/identity_pb";
import { HTTP_STATUS, USER_UPDATE_MASK_PATHS } from "@constants/app";
import { GRPC_CLIENT_ERRORS } from "@constants/messages";

import { GrpcApiError } from "../errors";
import { grpcTransport } from "../transport";

/**
 * List users
 */
export async function listUsers(params?: {
  pageSize?: number;
  pageToken?: string;
}): Promise<{
  users: Array<{
    id: string;
    email: string;
    isAdmin: boolean;
    isActive: boolean;
    isTenantManager: boolean;
    isBaseStationManager: boolean;
    isEndpointManager: boolean;
    note?: string;
    createdAt?: Date;
    updatedAt?: Date;
  }>;
  nextPageToken?: string;
  totalCount: number;
}> {
  const request = new identityPb.ListUsersRequest();
  if (params?.pageSize) request.setPageSize(params.pageSize);
  if (params?.pageToken) request.setPageToken(params.pageToken);

  const response = await grpcTransport.callIdentity<
    identityPb.ListUsersRequest,
    identityPb.ListUsersResponse
  >((c) => c.listUsers, request, { requireOrgUser: false });

  return {
    users: response.getUsersList().map((u) => ({
      id: u.getId(),
      email: u.getEmail(),
      isAdmin: u.getIsAdmin(),
      isActive: u.getIsActive(),
      isTenantManager: u.getIsTenantManager(),
      isBaseStationManager: u.getIsBaseStationManager(),
      isEndpointManager: u.getIsEndpointManager(),
      note: u.getNote() || undefined,
      createdAt: u.getCreatedAt()?.toDate(),
      updatedAt: u.getUpdatedAt()?.toDate(),
    })),
    nextPageToken: response.getNextPageToken() || undefined,
    totalCount: response.getTotalCount(),
  };
}

/**
 * Get user by ID
 */
export async function getUser(userId: string): Promise<{
  id: string;
  email: string;
  isAdmin: boolean;
  isActive: boolean;
  isTenantManager: boolean;
  isBaseStationManager: boolean;
  isEndpointManager: boolean;
  note?: string;
  createdAt?: Date;
  updatedAt?: Date;
} | null> {
  const request = new identityPb.GetUserRequest();
  request.setId(userId);

  try {
    const response = await grpcTransport.callIdentity<
      identityPb.GetUserRequest,
      identityPb.GetUserResponse
    >((c) => c.getUser, request, { requireOrgUser: false });

    const user = response.getUser();
    if (!user) return null;

    return {
      id: user.getId(),
      email: user.getEmail(),
      isAdmin: user.getIsAdmin(),
      isActive: user.getIsActive(),
      isTenantManager: user.getIsTenantManager(),
      isBaseStationManager: user.getIsBaseStationManager(),
      isEndpointManager: user.getIsEndpointManager(),
      note: user.getNote() || undefined,
      createdAt: user.getCreatedAt()?.toDate(),
      updatedAt: user.getUpdatedAt()?.toDate(),
    };
  } catch (error) {
    if (error instanceof GrpcApiError && error.isNotFound()) {
      return null;
    }
    throw error;
  }
}

/**
 * Create user
 */
export async function createUser(data: {
  email: string;
  password?: string;
  isAdmin?: boolean;
  isActive?: boolean;
  isTenantManager?: boolean;
  isBaseStationManager?: boolean;
  isEndpointManager?: boolean;
  note?: string;
}): Promise<{
  id: string;
  email: string;
  isAdmin: boolean;
  isActive: boolean;
  isTenantManager: boolean;
  isBaseStationManager: boolean;
  isEndpointManager: boolean;
  note?: string;
  createdAt?: Date;
  updatedAt?: Date;
}> {
  const request = new identityPb.CreateUserRequest();
  request.setEmail(data.email);
  if (data.password) request.setPassword(data.password);
  if (data.isAdmin !== undefined) request.setIsAdmin(data.isAdmin);
  if (data.isActive !== undefined) request.setIsActive(data.isActive);
  if (data.isTenantManager !== undefined)
    request.setIsTenantManager(data.isTenantManager);
  if (data.isBaseStationManager !== undefined)
    request.setIsBaseStationManager(data.isBaseStationManager);
  if (data.isEndpointManager !== undefined)
    request.setIsEndpointManager(data.isEndpointManager);
  if (data.note) request.setNote(data.note);

  const response = await grpcTransport.callIdentity<
    identityPb.CreateUserRequest,
    identityPb.CreateUserResponse
  >((c) => c.createUser, request, { requireOrgUser: false });

  const user = response.getUser();
  if (!user) {
    throw new GrpcApiError(
      HTTP_STATUS.INTERNAL_SERVER_ERROR,
      GRPC_CLIENT_ERRORS.INVALID_CREATE_USER_RESPONSE,
    );
  }

  return {
    id: user.getId(),
    email: user.getEmail(),
    isAdmin: user.getIsAdmin(),
    isActive: user.getIsActive(),
    isTenantManager: user.getIsTenantManager(),
    isBaseStationManager: user.getIsBaseStationManager(),
    isEndpointManager: user.getIsEndpointManager(),
    note: user.getNote() || undefined,
    createdAt: user.getCreatedAt()?.toDate(),
    updatedAt: user.getUpdatedAt()?.toDate(),
  };
}

/**
 * Update user with FieldMask for partial updates
 */
export async function updateUser(
  userId: string,
  data: {
    email?: string;
    isAdmin?: boolean;
    isActive?: boolean;
    isTenantManager?: boolean;
    isBaseStationManager?: boolean;
    isEndpointManager?: boolean;
    note?: string;
  },
): Promise<{
  id: string;
  email: string;
  isAdmin: boolean;
  isActive: boolean;
  isTenantManager: boolean;
  isBaseStationManager: boolean;
  isEndpointManager: boolean;
  note?: string;
  createdAt?: Date;
  updatedAt?: Date;
}> {
  const request = new identityPb.UpdateUserRequest();
  request.setId(userId);

  const paths: string[] = [];
  if (data.email !== undefined) {
    request.setEmail(data.email);
    paths.push(USER_UPDATE_MASK_PATHS.EMAIL);
  }
  if (data.isAdmin !== undefined) {
    request.setIsAdmin(data.isAdmin);
    paths.push(USER_UPDATE_MASK_PATHS.IS_ADMIN);
  }
  if (data.isActive !== undefined) {
    request.setIsActive(data.isActive);
    paths.push(USER_UPDATE_MASK_PATHS.IS_ACTIVE);
  }
  if (data.isTenantManager !== undefined) {
    request.setIsTenantManager(data.isTenantManager);
    paths.push(USER_UPDATE_MASK_PATHS.IS_TENANT_MANAGER);
  }
  if (data.isBaseStationManager !== undefined) {
    request.setIsBaseStationManager(data.isBaseStationManager);
    paths.push(USER_UPDATE_MASK_PATHS.IS_BASE_STATION_MANAGER);
  }
  if (data.isEndpointManager !== undefined) {
    request.setIsEndpointManager(data.isEndpointManager);
    paths.push(USER_UPDATE_MASK_PATHS.IS_ENDPOINT_MANAGER);
  }
  if (data.note !== undefined) {
    request.setNote(data.note);
    paths.push(USER_UPDATE_MASK_PATHS.NOTE);
  }

  const mask = new FieldMask();
  mask.setPathsList(paths);
  request.setUpdateMask(mask);

  const response = await grpcTransport.callIdentity<
    identityPb.UpdateUserRequest,
    identityPb.UpdateUserResponse
  >((c) => c.updateUser, request, { requireOrgUser: false });

  const user = response.getUser();
  if (!user) {
    throw new GrpcApiError(
      HTTP_STATUS.INTERNAL_SERVER_ERROR,
      GRPC_CLIENT_ERRORS.INVALID_UPDATE_USER_RESPONSE,
    );
  }

  return {
    id: user.getId(),
    email: user.getEmail(),
    isAdmin: user.getIsAdmin(),
    isActive: user.getIsActive(),
    isTenantManager: user.getIsTenantManager(),
    isBaseStationManager: user.getIsBaseStationManager(),
    isEndpointManager: user.getIsEndpointManager(),
    note: user.getNote() || undefined,
    createdAt: user.getCreatedAt()?.toDate(),
    updatedAt: user.getUpdatedAt()?.toDate(),
  };
}

/**
 * Delete user
 */
export async function deleteUser(userId: string): Promise<void> {
  const request = new identityPb.DeleteUserRequest();
  request.setId(userId);

  await grpcTransport.callIdentity<
    identityPb.DeleteUserRequest,
    identityPb.DeleteUserResponse
  >((c) => c.deleteUser, request, { requireOrgUser: false });
}

/**
 * Update user password (admin)
 */
export async function updateUserPassword(
  userId: string,
  newPassword: string,
): Promise<void> {
  const request = new identityPb.UpdateUserPasswordRequest();
  request.setId(userId);
  request.setNewPassword(newPassword);

  await grpcTransport.callIdentity<
    identityPb.UpdateUserPasswordRequest,
    identityPb.UpdateUserPasswordResponse
  >((c) => c.updateUserPassword, request, { requireOrgUser: false });
}

/**
 * List organizations a user belongs to
 */
export async function listUserOrganizations(userId: string): Promise<{
  memberships: Array<{
    orgId: string;
    orgName: string;
    role: string;
    status: string;
  }>;
}> {
  const request = new identityPb.ListUserOrganizationsRequest();
  request.setUserId(userId);
  const response = await grpcTransport.callIdentity<
    identityPb.ListUserOrganizationsRequest,
    identityPb.ListUserOrganizationsResponse
  >((c) => c.listUserOrganizations, request, { requireOrgUser: false });
  return {
    memberships: response.getMembershipsList().map((m) => ({
      orgId: m.getOrgId(),
      orgName: m.getOrgName(),
      role: m.getRole(),
      status: m.getStatus(),
    })),
  };
}
