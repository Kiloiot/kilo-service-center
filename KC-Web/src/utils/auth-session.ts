/**
 * Shared helper for persisting authentication session data after login
 * or external auth callback. Stores tokens and resolves the user's
 * default organization context.
 */

import type { LoginResponseAPI } from "@api-types/api";

import { storageService } from "@utils/storage";
import { DEFAULT_ORG_NAME, STORAGE_KEYS } from "@constants/app";
import { ERR_AUTH_ORG_REQUIRED } from "@constants/messages";

export interface PersistAuthSessionResult {
  user: LoginResponseAPI["user"];
  orgId: string;
  orgName: string;
}

/**
 * Stores access/refresh tokens, resolves the user's organization, and
 * returns the session data needed by callers to finish the auth flow.
 *
 * Throws with a user-facing message when the profile has no memberships.
 */
export function persistAuthSession(
  loginResponse: LoginResponseAPI,
): PersistAuthSessionResult {
  storageService.setItem(
    STORAGE_KEYS.AUTH_TOKEN,
    loginResponse.tokens.accessToken,
  );

  if (loginResponse.tokens.refreshToken) {
    storageService.setItem(
      STORAGE_KEYS.REFRESH_TOKEN,
      loginResponse.tokens.refreshToken,
    );
  } else {
    storageService.removeItem(STORAGE_KEYS.REFRESH_TOKEN);
  }

  const { user } = loginResponse;
  const defaultOrg = user.memberships.find(
    (m) => m.orgId === user.defaultOrgId,
  );
  const firstOrg = user.memberships[0];
  const org = defaultOrg || firstOrg;

  if (!org) {
    storageService.removeItem(STORAGE_KEYS.AUTH_TOKEN);
    throw new Error(ERR_AUTH_ORG_REQUIRED);
  }

  return {
    user,
    orgId: org.orgId,
    orgName: org.orgName || DEFAULT_ORG_NAME,
  };
}
