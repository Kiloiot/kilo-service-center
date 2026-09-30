/**
 * Authentication Guard
 *
 * Checks if authentication is required and redirects to login if not authenticated.
 * Protects routes when backend auth is enabled.
 */

import type { ReactNode } from "react";
import React, { useEffect } from "react";
import { Navigate, useLocation } from "react-router-dom";

import { useAuthSettings } from "@hooks";

import GlobalLoader from "@components/common/GlobalLoader";
import { useSession } from "@contexts/SessionContext";
import { logger } from "@utils/logger";
import { ROUTES } from "@constants/app";
import { ERR_AUTH_SETTINGS_LOAD } from "@constants/messages";

interface AuthGuardProps {
  children: ReactNode;
}

/**
 * AuthGuard wraps protected content and redirects to login when:
 * - Backend auth is enabled (local_login_enabled=true in auth settings)
 * - User is not authenticated (no token/session)
 *
 * Public routes (login, auth callback) bypass this guard via pathname check.
 */
export const AuthGuard: React.FC<AuthGuardProps> = ({ children }) => {
  const { isAuthenticated, isHydrated } = useSession();
  const location = useLocation();
  const {
    data: settings,
    isPending: settingsPending,
    error: settingsError,
  } = useAuthSettings();

  // Public routes that don't require authentication
  const publicPaths = [ROUTES.LOGIN, ROUTES.REGISTER, ROUTES.AUTH_CALLBACK];
  const isPublicRoute = publicPaths.some((path) =>
    location.pathname.startsWith(path),
  );

  useEffect(() => {
    if (settingsError) logger.error(ERR_AUTH_SETTINGS_LOAD, settingsError);
  }, [settingsError]);

  // A failed refetch keeps the last loaded settings; with none loaded, fail closed.
  const authEnabled = settings
    ? settings.local_login_enabled ||
      settings.oidc?.enabled ||
      settings.oauth2?.enabled ||
      false
    : settingsError !== null;

  // A paused (offline) settings request is pending but not loading; it must not open the gate.
  if (!isHydrated || settingsPending) {
    return <GlobalLoader />;
  }

  // Registration is for visitors; a signed-in user goes to the dashboard.
  if (isAuthenticated && location.pathname.startsWith(ROUTES.REGISTER)) {
    return <Navigate to={ROUTES.HOME} replace />;
  }

  // Public routes bypass auth check
  if (isPublicRoute) {
    return <>{children}</>;
  }

  // If auth is disabled, allow access
  if (!authEnabled) {
    return <>{children}</>;
  }

  // Auth enabled but not authenticated - redirect to login
  if (!isAuthenticated) {
    return <Navigate to={ROUTES.LOGIN} state={{ from: location }} replace />;
  }

  // Authenticated - render protected content
  return <>{children}</>;
};
