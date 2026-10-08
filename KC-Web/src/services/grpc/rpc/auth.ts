/**
 * Authentication RPCs (IdentityService): login, registration, provider code exchange, profile, token refresh, logout and password change.
 */

import type { UserRolesAPI } from "@api-types/api";

import * as identityPb from "@services/grpc/identity_pb";
import { HTTP_STATUS } from "@constants/app";
import { GRPC_CLIENT_ERRORS } from "@constants/messages";

import { GrpcApiError } from "../errors";
import { grpcTransport } from "../transport";

/** Shared membership shape returned by login/exchange endpoints. */
export interface AuthMembership {
  orgId: string;
  orgName: string;
  role: string;
  status: string;
  isOrgAdmin: boolean;
  isBaseStationAdmin: boolean;
  isEndpointAdmin: boolean;
}

/** Shared response shape returned by login, exchangeOIDC, and exchangeOAuth2. */
export interface AuthTokensAndUser {
  tokens: {
    accessToken: string;
    refreshToken: string;
    accessExpiresIn: number;
    refreshExpiresIn: number;
  };
  user: {
    id: string;
    email: string;
    isAdmin: boolean;
    hasPassword: boolean;
    memberships: AuthMembership[];
    defaultOrgId?: string;
    firstName?: string;
    lastName?: string;
  };
}

/**
 * Extract tokens and user data from a LoginResponse protobuf message.
 * Throws GrpcApiError when the response lacks required fields.
 */
export function extractAuthTokensAndUser(
  response: identityPb.LoginResponse,
  errorToken: string,
): AuthTokensAndUser {
  const tokens = response.getTokens();
  const user = response.getUser();

  if (!tokens || !user) {
    throw new GrpcApiError(HTTP_STATUS.INTERNAL_SERVER_ERROR, errorToken);
  }

  return {
    tokens: {
      accessToken: tokens.getAccessToken(),
      refreshToken: tokens.getRefreshToken(),
      accessExpiresIn: tokens.getAccessExpiresIn(),
      refreshExpiresIn: tokens.getRefreshExpiresIn(),
    },
    user: {
      id: user.getId(),
      email: user.getEmail(),
      isAdmin: user.getIsAdmin(),
      hasPassword: user.getHasPassword(),
      memberships: user.getMembershipsList().map((m) => ({
        orgId: m.getOrgId(),
        orgName: m.getOrgName(),
        role: m.getRole(),
        status: m.getStatus(),
        isOrgAdmin: m.getIsOrgAdmin(),
        isBaseStationAdmin: m.getIsBaseStationAdmin(),
        isEndpointAdmin: m.getIsEndpointAdmin(),
      })),
      defaultOrgId: user.getDefaultOrgId() || undefined,
      firstName: user.getFirstName() || undefined,
      lastName: user.getLastName() || undefined,
    },
  };
}

/**
 * Login with email and password
 */
export async function login(
  email: string,
  password: string,
): Promise<AuthTokensAndUser> {
  const request = new identityPb.LoginRequest();
  request.setEmail(email);
  request.setPassword(password);

  // Public method: skip org/user requirement (pre-auth).
  // skipTokenRefresh prevents the 401-refresh handler from swallowing
  // invalid-credential errors into a never-resolving redirect promise.
  const response = await grpcTransport.callIdentity<
    identityPb.LoginRequest,
    identityPb.LoginResponse
  >((c) => c.login, request, {
    requireOrgUser: false,
    skipTokenRefresh: true,
  });

  return extractAuthTokensAndUser(
    response,
    GRPC_CLIENT_ERRORS.INVALID_LOGIN_RESPONSE,
  );
}

/**
 * Register a new account (public endpoint)
 */
export async function registerAccount(
  email: string,
  password: string,
  firstName: string,
  lastName: string,
  companyName: string,
): Promise<{
  tokens: {
    accessToken: string;
    refreshToken: string;
    accessExpiresIn: number;
    refreshExpiresIn: number;
  };
  user: {
    id: string;
    email: string;
    isAdmin: boolean;
    hasPassword: boolean;
    memberships: Array<{
      orgId: string;
      orgName: string;
      role: string;
      status: string;
      isOrgAdmin: boolean;
      isBaseStationAdmin: boolean;
      isEndpointAdmin: boolean;
    }>;
    defaultOrgId?: string;
    firstName?: string;
    lastName?: string;
  };
}> {
  const request = new identityPb.RegisterAccountRequest();
  request.setEmail(email);
  request.setPassword(password);
  request.setFirstName(firstName);
  request.setLastName(lastName);
  request.setCompanyName(companyName);

  const response = await grpcTransport.callIdentity<
    identityPb.RegisterAccountRequest,
    identityPb.LoginResponse
  >((c) => c.registerAccount, request, { requireOrgUser: false });

  const tokens = response.getTokens();
  const user = response.getUser();

  if (!tokens || !user) {
    throw new GrpcApiError(
      HTTP_STATUS.INTERNAL_SERVER_ERROR,
      GRPC_CLIENT_ERRORS.INVALID_LOGIN_RESPONSE,
    );
  }

  return {
    tokens: {
      accessToken: tokens.getAccessToken(),
      refreshToken: tokens.getRefreshToken(),
      accessExpiresIn: tokens.getAccessExpiresIn(),
      refreshExpiresIn: tokens.getRefreshExpiresIn(),
    },
    user: {
      id: user.getId(),
      email: user.getEmail(),
      isAdmin: user.getIsAdmin(),
      hasPassword: user.getHasPassword(),
      memberships: user.getMembershipsList().map((m) => ({
        orgId: m.getOrgId(),
        orgName: m.getOrgName(),
        role: m.getRole(),
        status: m.getStatus(),
        isOrgAdmin: m.getIsOrgAdmin(),
        isBaseStationAdmin: m.getIsBaseStationAdmin(),
        isEndpointAdmin: m.getIsEndpointAdmin(),
      })),
      defaultOrgId: user.getDefaultOrgId() || undefined,
      firstName: user.getFirstName() || undefined,
      lastName: user.getLastName() || undefined,
    },
  };
}

/**
 * Logout and revoke refresh token
 */
export async function logout(refreshToken: string): Promise<void> {
  const request = new identityPb.LogoutRequest();
  request.setRefreshToken(refreshToken);

  await grpcTransport.callIdentity<
    identityPb.LogoutRequest,
    identityPb.LogoutResponse
  >((c) => c.logout, request);
}

/**
 * Get authenticated user profile
 */
export async function getProfile(): Promise<{
  id: string;
  email: string;
  isAdmin: boolean;
  hasPassword: boolean;
  memberships: Array<{
    orgId: string;
    orgName: string;
    role: string;
    status: string;
    isOrgAdmin: boolean;
    isBaseStationAdmin: boolean;
    isEndpointAdmin: boolean;
  }>;
  defaultOrgId?: string;
  externalId?: string;
  externalIdp?: string;
  firstName?: string;
  lastName?: string;
  roles: UserRolesAPI;
}> {
  const request = new identityPb.GetProfileRequest();
  const response = await grpcTransport.callIdentity<
    identityPb.GetProfileRequest,
    identityPb.GetProfileResponse
  >((c) => c.getProfile, request);

  const user = response.getUser();
  if (!user) {
    throw new GrpcApiError(
      HTTP_STATUS.INTERNAL_SERVER_ERROR,
      GRPC_CLIENT_ERRORS.INVALID_PROFILE_RESPONSE,
    );
  }

  return {
    id: user.getId(),
    email: user.getEmail(),
    isAdmin: user.getIsAdmin(),
    hasPassword: user.getHasPassword(),
    memberships: user.getMembershipsList().map((m) => ({
      orgId: m.getOrgId(),
      orgName: m.getOrgName(),
      role: m.getRole(),
      status: m.getStatus(),
      isOrgAdmin: m.getIsOrgAdmin(),
      isBaseStationAdmin: m.getIsBaseStationAdmin(),
      isEndpointAdmin: m.getIsEndpointAdmin(),
    })),
    defaultOrgId: user.getDefaultOrgId() || undefined,
    firstName: user.getFirstName() || undefined,
    lastName: user.getLastName() || undefined,
    roles: {
      admin: user.getRoles()?.getAdmin() ?? false,
      tenantManager: user.getRoles()?.getTenantManager() ?? false,
      baseStationManager: user.getRoles()?.getBaseStationManager() ?? false,
      endpointManager: user.getRoles()?.getEndpointManager() ?? false,
    },
  };
}

/**
 * Change own password
 */
export async function changePassword(
  currentPassword: string,
  newPassword: string,
): Promise<void> {
  const request = new identityPb.ChangePasswordRequest();
  request.setCurrentPassword(currentPassword);
  request.setNewPassword(newPassword);

  await grpcTransport.callIdentity<
    identityPb.ChangePasswordRequest,
    identityPb.ChangePasswordResponse
  >((c) => c.changePassword, request);
}

/**
 * Exchange OIDC authorization code for tokens.
 * Backend resolves the redirect URI from server-side config, so the
 * client doesn't send one over the wire.
 */
export async function exchangeOIDC(
  code: string,
  state: string,
): Promise<AuthTokensAndUser> {
  const request = new identityPb.ExchangeOIDCRequest();
  request.setCode(code);
  request.setState(state);

  const response = await grpcTransport.callIdentity<
    identityPb.ExchangeOIDCRequest,
    identityPb.LoginResponse
  >((c) => c.exchangeOIDC, request, {
    requireOrgUser: false,
    skipTokenRefresh: true,
  });

  return extractAuthTokensAndUser(
    response,
    GRPC_CLIENT_ERRORS.INVALID_OIDC_EXCHANGE_RESPONSE,
  );
}

/**
 * Exchange OAuth2 authorization code for tokens.
 * Backend resolves the redirect URI from server-side config, so the
 * client doesn't send one over the wire.
 */
export async function exchangeOAuth2(
  code: string,
  state: string,
): Promise<AuthTokensAndUser> {
  const request = new identityPb.ExchangeOAuth2Request();
  request.setCode(code);
  request.setState(state);

  const response = await grpcTransport.callIdentity<
    identityPb.ExchangeOAuth2Request,
    identityPb.LoginResponse
  >((c) => c.exchangeOAuth2, request, {
    requireOrgUser: false,
    skipTokenRefresh: true,
  });

  return extractAuthTokensAndUser(
    response,
    GRPC_CLIENT_ERRORS.INVALID_OAUTH2_EXCHANGE_RESPONSE,
  );
}
