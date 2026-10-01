import React, { useEffect, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";

import { useExchangeAuthCode, useLoadAuthProfile } from "@hooks";
import { Alert, Box, CircularProgress, Link, Typography } from "@mui/material";

import { useOrganization } from "@contexts/OrganizationContext";
import { useSession } from "@contexts/SessionContext";
import { persistAuthSession } from "@utils/auth-session";
import { storageService } from "@utils/storage";
import {
  AUTH_LAYOUT,
  DEFAULT_ORG_NAME,
  OAUTH_CALLBACK_PARAMS,
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
  const { mutateAsync: exchangeAuthCode } = useExchangeAuthCode();
  const { mutateAsync: loadAuthProfile } = useLoadAuthProfile();

  useEffect(() => {
    const handleCallback = async () => {
      const code = searchParams.get(OAUTH_CALLBACK_PARAMS.CODE);
      const state = searchParams.get(OAUTH_CALLBACK_PARAMS.STATE);

      // Check for error response from provider
      const errorParam = searchParams.get(OAUTH_CALLBACK_PARAMS.ERROR);
      if (errorParam) {
        setError(ERR_AUTH_CALLBACK_EXCHANGE_FAILED);
        return;
      }

      const hashParams = new URLSearchParams(
        window.location.hash.replace(/^#/, ""),
      );
      const accessToken = hashParams.get(OAUTH_CALLBACK_PARAMS.ACCESS_TOKEN);
      if (accessToken) {
        storageService.setItem(STORAGE_KEYS.AUTH_TOKEN, accessToken);

        try {
          const profile = await loadAuthProfile();

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
        const loginResponse = await exchangeAuthCode({ code, state });

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
  }, [
    searchParams,
    navigate,
    setOrganization,
    setUser,
    exchangeAuthCode,
    loadAuthProfile,
  ]);

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
        <Link
          component="button"
          variant="body2"
          onClick={() => navigate(ROUTES.LOGIN)}
        >
          {ACTION_RETURN_TO_LOGIN}
        </Link>
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
