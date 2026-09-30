/**
 * Authentication settings RPC (IdentityService.GetAuthSettings): the sign-in
 * providers and the password policy, read before a user signs in.
 */

import * as identityPb from "@services/grpc/identity_pb";
import { HTTP_STATUS } from "@constants/app";
import { GRPC_CLIENT_ERRORS } from "@constants/messages";

import { GrpcApiError } from "../errors";
import { grpcTransport } from "../transport";

/** The rules a new password must meet. */
export interface PasswordPolicy {
  minLength: number;
  maxLength: number;
  requiresLetter: boolean;
  requiresDigit: boolean;
}

/**
 * Get authentication settings (public endpoint)
 */
export async function getAuthSettings(): Promise<{
  enabled: boolean;
  localLoginEnabled: boolean;
  loginUrl?: string;
  loginLabel?: string;
  loginRedirect: boolean;
  logoutUrl?: string;
  refreshTokenEnabled: boolean;
  registrationEnabled: boolean;
  oidc?: {
    enabled: boolean;
    loginUrl?: string;
    loginLabel?: string;
    loginRedirect: boolean;
    logoutUrl?: string;
  };
  oauth2?: {
    enabled: boolean;
    loginUrl?: string;
    loginLabel?: string;
    loginRedirect: boolean;
    logoutUrl?: string;
  };
  passwordPolicy?: PasswordPolicy;
}> {
  const request = new identityPb.GetAuthSettingsRequest();
  // Public method: skip org/user requirement (pre-auth)
  const response = await grpcTransport.callIdentity<
    identityPb.GetAuthSettingsRequest,
    identityPb.GetAuthSettingsResponse
  >((c) => c.getAuthSettings, request, { requireOrgUser: false });

  const settings = response.getSettings();
  if (!settings) {
    throw new GrpcApiError(
      HTTP_STATUS.INTERNAL_SERVER_ERROR,
      GRPC_CLIENT_ERRORS.INVALID_AUTH_SETTINGS_RESPONSE,
    );
  }

  const oidc = settings.getOidc();
  const oauth2 = settings.getOauth2();
  const policy = settings.getPasswordPolicy();

  return {
    enabled: settings.getEnabled(),
    localLoginEnabled: settings.getLocalLoginEnabled(),
    loginUrl: settings.getLoginUrl() || undefined,
    loginLabel: settings.getLoginLabel() || undefined,
    loginRedirect: settings.getLoginRedirect(),
    logoutUrl: settings.getLogoutUrl() || undefined,
    refreshTokenEnabled: settings.getRefreshTokenEnabled(),
    registrationEnabled: settings.getRegistrationEnabled(),
    oidc: oidc
      ? {
          enabled: oidc.getEnabled(),
          loginUrl: oidc.getLoginUrl() || undefined,
          loginLabel: oidc.getLoginLabel() || undefined,
          loginRedirect: oidc.getLoginRedirect(),
          logoutUrl: oidc.getLogoutUrl() || undefined,
        }
      : undefined,
    oauth2: oauth2
      ? {
          enabled: oauth2.getEnabled(),
          loginUrl: oauth2.getLoginUrl() || undefined,
          loginLabel: oauth2.getLoginLabel() || undefined,
          loginRedirect: oauth2.getLoginRedirect(),
          logoutUrl: oauth2.getLogoutUrl() || undefined,
        }
      : undefined,
    passwordPolicy: policy
      ? {
          minLength: policy.getMinLength(),
          maxLength: policy.getMaxLength(),
          requiresLetter: policy.getRequiresLetter(),
          requiresDigit: policy.getRequiresDigit(),
        }
      : undefined,
  };
}
