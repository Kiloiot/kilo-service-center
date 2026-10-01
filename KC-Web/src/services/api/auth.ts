/**
 * Authentication and session-bootstrap calls in the shapes the UI consumes.
 */

import type {
  AuthSettingsAPI,
  ExchangeRequest,
  LoginRequest,
  LoginResponseAPI,
  RegisterAccountRequest,
  UserProfileAPI,
  UserRolesAPI,
} from "@api-types/api";
import { mapLoginResponse, mapUserProfile } from "@mappers";

import * as authRpc from "@services/grpc/rpc/auth";
import * as authSettingsRpc from "@services/grpc/rpc/auth-settings";

/** Shape of gRPC auth response returned by login/register/exchange methods */
export type GrpcAuthResponse = Awaited<ReturnType<typeof authRpc.login>>;

/** Maps a gRPC auth response to the snake_case shape expected by mapLoginResponse */
export function mapExchangeResponse(
  response: GrpcAuthResponse,
): LoginResponseAPI {
  return mapLoginResponse({
    tokens: {
      access_token: response.tokens.accessToken,
      refresh_token: response.tokens.refreshToken,
      access_expires_in: response.tokens.accessExpiresIn,
      refresh_expires_in: response.tokens.refreshExpiresIn,
    },
    user: {
      id: response.user.id,
      email: response.user.email,
      is_admin: response.user.isAdmin,
      has_password: response.user.hasPassword,
      memberships: response.user.memberships.map((m) => ({
        org_id: m.orgId,
        org_name: m.orgName,
        role: m.role,
        status: m.status,
        is_org_admin: m.isOrgAdmin ?? false,
        is_base_station_admin: m.isBaseStationAdmin ?? false,
        is_endpoint_admin: m.isEndpointAdmin ?? false,
      })),
      default_org_id: response.user.defaultOrgId,
      first_name: response.user.firstName,
      last_name: response.user.lastName,
    },
  });
}

export const authApi = {
  async getAuthSettings(): Promise<AuthSettingsAPI> {
    const response = await authSettingsRpc.getAuthSettings();
    return {
      enabled: response.enabled,
      local_login_enabled: response.localLoginEnabled,
      refresh_token_enabled: response.refreshTokenEnabled,
      registration_enabled: response.registrationEnabled,
      login_url: response.loginUrl,
      login_label: response.loginLabel,
      login_redirect: response.loginRedirect,
      logout_url: response.logoutUrl,
      oidc: response.oidc
        ? {
            enabled: response.oidc.enabled,
            login_url: response.oidc.loginUrl || "",
            login_label: response.oidc.loginLabel || "",
            login_redirect: response.oidc.loginRedirect,
            logout_url: response.oidc.logoutUrl,
          }
        : undefined,
      oauth2: response.oauth2
        ? {
            enabled: response.oauth2.enabled,
            login_url: response.oauth2.loginUrl || "",
            login_label: response.oauth2.loginLabel || "",
            login_redirect: response.oauth2.loginRedirect,
            logout_url: response.oauth2.logoutUrl,
          }
        : undefined,
      password_policy: response.passwordPolicy
        ? {
            min_length: response.passwordPolicy.minLength,
            max_length: response.passwordPolicy.maxLength,
            requires_letter: response.passwordPolicy.requiresLetter,
            requires_digit: response.passwordPolicy.requiresDigit,
          }
        : undefined,
    };
  },

  async getAuthProfile(): Promise<UserProfileAPI> {
    const response = await authRpc.getProfile();
    return mapUserProfile({
      id: response.id,
      email: response.email,
      is_admin: response.isAdmin,
      has_password: response.hasPassword,
      memberships: response.memberships.map((m) => ({
        org_id: m.orgId,
        org_name: m.orgName,
        role: m.role,
        status: m.status,
        is_org_admin: m.isOrgAdmin,
        is_base_station_admin: m.isBaseStationAdmin,
        is_endpoint_admin: m.isEndpointAdmin,
      })),
      default_org_id: response.defaultOrgId,
      first_name: response.firstName,
      last_name: response.lastName,
    });
  },

  /** Roles of the signed-in user in the organization the request acts in. */
  async getCurrentRoles(): Promise<UserRolesAPI> {
    const response = await authRpc.getProfile();
    return response.roles;
  },

  async login(payload: LoginRequest): Promise<LoginResponseAPI> {
    const response = await authRpc.login(payload.email, payload.password);
    return mapExchangeResponse(response);
  },

  async registerAccount(
    payload: RegisterAccountRequest,
  ): Promise<LoginResponseAPI> {
    const response = await authRpc.registerAccount(
      payload.email,
      payload.password,
      payload.firstName,
      payload.lastName,
      payload.companyName,
    );
    return mapExchangeResponse(response);
  },

  async exchangeOIDC(payload: ExchangeRequest): Promise<LoginResponseAPI> {
    const response = await authRpc.exchangeOIDC(payload.code, payload.state);
    return mapExchangeResponse(response);
  },

  async exchangeOAuth2(payload: ExchangeRequest): Promise<LoginResponseAPI> {
    const response = await authRpc.exchangeOAuth2(payload.code, payload.state);
    return mapExchangeResponse(response);
  },

  async logout(refreshToken: string): Promise<void> {
    await authRpc.logout(refreshToken);
  },

  async changeOwnPassword(
    currentPassword: string,
    newPassword: string,
  ): Promise<void> {
    await authRpc.changePassword(currentPassword, newPassword);
  },
};
