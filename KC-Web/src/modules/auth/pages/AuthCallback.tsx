import React, { useEffect, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";

import { Alert, Box, CircularProgress, Typography } from "@mui/material";

import { apiService } from "@services/api";
import { useOrganization } from "@contexts/OrganizationContext";
import { useSession } from "@contexts/SessionContext";
import { persistAuthSession } from "@utils/auth-session";
import { storageService } from "@utils/storage";
import {
  AUTH_LAYOUT,
  DEFAULT_ORG_NAME,
  ROUTES,
  STORAGE_KEYS,
} from "@constants/app";
import {
  ACTION_RETURN_TO_LOGIN,
  ERR_AUTH_CALLBACK_EXCHANGE_FAILED,
  ERR_AUTH_CALLBACK_MISSING_CODE,
  ERR_AUTH_ORG_REQUIRED,
  ERR_AUTH_PROFILE_LOAD_FAILED,
  MSG_AUTH_COMPLETING,
} from "@constants/messages";

const AuthCallback: React.FC = () => {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { setOrganization } = useOrganization();
  const { setUser } = useSession();
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const handleCallback = async () => {
      const code = searchParams.get("code");
      const state = searchParams.get("state");

      // Check for error response from provider
      const errorParam = searchParams.get("error");
      if (errorParam) {
        setError(ERR_AUTH_CALLBACK_EXCHANGE_FAILED);
        return;
      }

      const hashParams = new URLSearchParams(
        window.location.hash.replace(/^#/, ""),
      );
      const accessToken = hashParams.get("access_token");
      if (accessToken) {
        storageService.setItem(STORAGE_KEYS.AUTH_TOKEN, accessToken);

        try {
          const profile = await apiService.getAuthProfile();

          const defaultOrg = profile.memberships.find(
            (m) => m.orgId === profile.defaultOrgId,
          );
          const firstOrg = profile.memberships[0];
          const org = defaultOrg || firstOrg;

          if (!org) {
            storageService.removeItem(STORAGE_KEYS.AUTH_TOKEN);
            setError(ERR_AUTH_ORG_REQUIRED);
            return;
          }

          setUser(profile);
          setOrganization(
            org.orgId,
            org.orgName || DEFAULT_ORG_NAME,
            profile.id,
          );
          window.history.replaceState(
            null,
            document.title,
            ROUTES.AUTH_CALLBACK,
          );
          navigate(ROUTES.HOME);
          return;
        } catch {
          storageService.removeItem(STORAGE_KEYS.AUTH_TOKEN);
          setError(ERR_AUTH_PROFILE_LOAD_FAILED);
          return;
        }
      }

      // Validate required parameters
      if (!code || !state) {
        setError(ERR_AUTH_CALLBACK_MISSING_CODE);
        return;
      }

      try {
        // Try OIDC exchange first, then OAuth2 if it fails
        // Backend will determine the correct flow based on the state
        let loginResponse;
        try {
          loginResponse = await apiService.exchangeOIDC({ code, state });
        } catch {
          // If OIDC fails, try OAuth2
          loginResponse = await apiService.exchangeOAuth2({ code, state });
        }

        const session = persistAuthSession(loginResponse);
        setUser(session.user);
        setOrganization(session.orgId, session.orgName, session.user.id);
        navigate(ROUTES.HOME);
      } catch (err) {
        if (err instanceof Error && err.message === ERR_AUTH_ORG_REQUIRED) {
          setError(ERR_AUTH_ORG_REQUIRED);
        } else {
          setError(ERR_AUTH_CALLBACK_EXCHANGE_FAILED);
        }
      }
    };

    handleCallback();
  }, [searchParams, navigate, setOrganization, setUser]);

  if (error) {
    return (
      <Box
        display="flex"
        flexDirection="column"
        justifyContent="center"
        alignItems="center"
        minHeight={AUTH_LAYOUT.FULL_HEIGHT}
        gap={AUTH_LAYOUT.SPACING_MT}
      >
        <Alert severity="error">{error}</Alert>
        <Typography
          variant="body2"
          color="primary"
          sx={{ cursor: "pointer", textDecoration: "underline" }}
          onClick={() => navigate(ROUTES.LOGIN)}
        >
          {ACTION_RETURN_TO_LOGIN}
        </Typography>
      </Box>
    );
  }

  return (
    <Box
      display="flex"
      justifyContent="center"
      alignItems="center"
      minHeight={AUTH_LAYOUT.FULL_HEIGHT}
    >
      <CircularProgress />
      <Typography sx={{ ml: AUTH_LAYOUT.SPACING_ML }}>
        {MSG_AUTH_COMPLETING}
      </Typography>
    </Box>
  );
};

export default AuthCallback;
