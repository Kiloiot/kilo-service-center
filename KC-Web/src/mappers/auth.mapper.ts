/**
 * Auth Mappers
 *
 * Transforms auth-related API responses from snake_case to camelCase.
 */

import type {
  AuthTokensAPI,
  LoginResponseAPI,
  UserMembershipAPI,
  UserProfileAPI,
} from "@api-types/api";

/**
 * Raw API response types (snake_case from backend)
 */
interface RawAuthTokens {
  access_token: string;
  access_expires_in: number;
  refresh_token?: string;
  refresh_expires_in?: number;
}

interface RawUserMembership {
  org_id: string;
  org_name: string;
  role: string;
  status: string;
  is_org_admin: boolean;
  is_base_station_admin: boolean;
  is_endpoint_admin: boolean;
}

interface RawUserProfile {
  id: string;
  email: string;
  is_admin: boolean;
  has_password?: boolean;
  memberships: RawUserMembership[];
  default_org_id?: string;
  first_name?: string;
  last_name?: string;
}

// Export RawUserProfile for use in api.ts
export type { RawUserProfile };

export interface RawLoginResponse {
  tokens: RawAuthTokens;
  user: RawUserProfile;
}

/**
 * Map raw auth tokens from API to typed interface.
 */
export function mapAuthTokens(raw: RawAuthTokens): AuthTokensAPI {
  return {
    accessToken: raw.access_token,
    accessExpiresIn: raw.access_expires_in,
    refreshToken: raw.refresh_token,
    refreshExpiresIn: raw.refresh_expires_in,
  };
}

/**
 * Map raw user membership from API to typed interface.
 */
export function mapUserMembership(raw: RawUserMembership): UserMembershipAPI {
  return {
    orgId: raw.org_id,
    orgName: raw.org_name,
    role: raw.role,
    status: raw.status,
    isOrgAdmin: raw.is_org_admin ?? false,
    isBaseStationAdmin: raw.is_base_station_admin ?? false,
    isEndpointAdmin: raw.is_endpoint_admin ?? false,
  };
}

/**
 * Map raw user profile from API to typed interface.
 */
export function mapUserProfile(raw: RawUserProfile): UserProfileAPI {
  return {
    id: raw.id,
    email: raw.email,
    isAdmin: raw.is_admin,
    hasPassword: raw.has_password,
    memberships: raw.memberships.map(mapUserMembership),
    defaultOrgId: raw.default_org_id,
    firstName: raw.first_name,
    lastName: raw.last_name,
  };
}

/**
 * Map raw login response from API to typed interface.
 */
export function mapLoginResponse(raw: RawLoginResponse): LoginResponseAPI {
  return {
    tokens: mapAuthTokens(raw.tokens),
    user: mapUserProfile(raw.user),
  };
}
