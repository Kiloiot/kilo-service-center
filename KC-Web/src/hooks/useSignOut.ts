import { useNavigate } from "react-router-dom";

import { useAuthSettings, useLogout } from "@hooks";

import { useOrganization } from "@contexts/OrganizationContext";
import { useSession } from "@contexts/SessionContext";
import { storageService } from "@utils/storage";
import { ROUTES, STORAGE_KEYS } from "@constants/app";
import { resetQueryClient } from "@config/query-client";

/**
 * Ends the session: revokes the refresh token when the service keeps them,
 * clears every local trace of the user and opens the provider's logout page,
 * or the sign-in page without one.
 */
export function useSignOut() {
  const navigate = useNavigate();
  const { clearSession } = useSession();
  const { clearOrganization } = useOrganization();
  const { data: authSettings } = useAuthSettings();
  const logout = useLogout();

  return async () => {
    const refreshToken = storageService.getItem(STORAGE_KEYS.REFRESH_TOKEN);
    // Without refresh tokens the service answers the logout call with 501.
    if (authSettings?.refresh_token_enabled === true && refreshToken) {
      await logout.mutateAsync(refreshToken).catch(() => undefined);
    }

    clearSession();
    clearOrganization();
    storageService.removeItem(STORAGE_KEYS.REFRESH_TOKEN);
    resetQueryClient();

    const logoutUrl =
      authSettings?.oidc?.logout_url || authSettings?.oauth2?.logout_url;
    if (logoutUrl) {
      window.location.href = logoutUrl;
    } else {
      navigate(ROUTES.LOGIN);
    }
  };
}
