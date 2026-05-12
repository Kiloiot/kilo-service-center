/**
 * Authentication Guard
 *
 * Checks if authentication is required and redirects to login if not authenticated.
 * Protects routes when backend auth is enabled.
 */

import type { ReactNode } from "react";
import React, { useEffect, useState } from "react";
import { Navigate, useLocation } from "react-router-dom";

import GlobalLoader from "@components/common/GlobalLoader";
import { apiService } from "@services/api";
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
  const [authEnabled, setAuthEnabled] = useState<boolean | null>(null);
  const [loading, setLoading] = useState(true);

  // Public routes that don't require authentication
  const publicPaths = [ROUTES.LOGIN, ROUTES.REGISTER, ROUTES.AUTH_CALLBACK];
  const isPublicRoute = publicPaths.some((path) =>
    location.pathname.startsWith(path),
  );

  useEffect(() => {
    const checkAuthSettings = async () => {
      try {
        const settings = await apiService.getAuthSettings();
        // Auth is enabled if local login is enabled OR external providers are enabled
        const enabled =
          settings.local_login_enabled ||
          settings.oidc?.enabled ||
          settings.oauth2?.enabled ||
          false;
        setAuthEnabled(enabled);
      } catch (error) {
        // Fail-closed: if settings can't be fetched, require authentication
        logger.error(ERR_AUTH_SETTINGS_LOAD, error);
        setAuthEnabled(true);
      } finally {
        setLoading(false);
      }
    };

    checkAuthSettings();
  }, []);

  // Wait for both session hydration and auth settings
  if (!isHydrated || loading) {
    return <GlobalLoader />;
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
