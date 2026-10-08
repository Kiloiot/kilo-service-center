import React, {
  createContext,
  type ReactNode,
  useContext,
  useEffect,
  useState,
} from "react";

import type { UserProfileAPI } from "@api-types/api";

import { sessionApi } from "@services/api";
import { realtimeLifecycle } from "@services/realtime";
import { logger } from "@utils/logger";
import { storageService } from "@utils/storage";
import { scheduleRefresh, stopRefresh } from "@utils/tokenRefresh";
import { STORAGE_KEYS } from "@constants/app";
import { SESSION_ERRORS } from "@constants/messages";

/**
 * Session Context
 *
 * Provides the authenticated user's profile, persisted to localStorage and
 * hydrated on app load. Roles are not taken from the stored profile: they are
 * read from the service through useCapabilities.
 */

interface SessionContextValue {
  user: UserProfileAPI | null;
  isAuthenticated: boolean;
  isHydrated: boolean; // true once initial storage load completes
  setUser: (user: UserProfileAPI) => void;
  clearSession: () => void;
}

const SessionContext = createContext<SessionContextValue | undefined>(
  undefined,
);

export const SessionProvider: React.FC<{ children: ReactNode }> = ({
  children,
}) => {
  const [user, setUserState] = useState<UserProfileAPI | null>(null);
  const [isHydrated, setIsHydrated] = useState(false);

  // Hydrate from storage on mount
  useEffect(() => {
    try {
      const stored = storageService.getItem(STORAGE_KEYS.USER_PROFILE);
      if (stored) {
        const parsed = JSON.parse(stored) as UserProfileAPI;
        setUserState(parsed);
      }
    } catch {
      // Corrupted profile - clear storage to prevent recurring errors
      storageService.removeItem(STORAGE_KEYS.USER_PROFILE);
      logger.error(SESSION_ERRORS.PROFILE_PARSE_FAILED);
    } finally {
      setIsHydrated(true); // Always mark hydrated even on error
    }
  }, []);

  // Single cleanup helper for auth teardown (logout, 401 auth failure)
  const performCleanup = () => {
    stopRefresh();
    storageService.removeItem(STORAGE_KEYS.USER_PROFILE);
    storageService.removeItem(STORAGE_KEYS.AUTH_TOKEN);
    storageService.removeItem(STORAGE_KEYS.REFRESH_TOKEN);
    realtimeLifecycle.reset();
    sessionApi.clearOrganization();
    setUserState(null);
  };

  // Register auth failure callback with API service to clear session on 401
  useEffect(() => {
    sessionApi.setAuthFailureCallback(performCleanup);

    return () => {
      sessionApi.setAuthFailureCallback(undefined);
    };
  }, []);

  // Start proactive token refresh scheduler when authenticated
  useEffect(() => {
    if (
      isHydrated &&
      user &&
      storageService.getItem(STORAGE_KEYS.REFRESH_TOKEN)
    ) {
      scheduleRefresh();
    }
    return () => stopRefresh();
  }, [isHydrated, user]);

  const setUser = (profile: UserProfileAPI) => {
    storageService.setItem(STORAGE_KEYS.USER_PROFILE, JSON.stringify(profile));
    setUserState(profile);
  };

  const clearSession = () => {
    performCleanup();
  };

  // Check both user profile AND token existence for proper auth state
  const hasToken = !!storageService.getItem(STORAGE_KEYS.AUTH_TOKEN);
  const isAuthenticated = user !== null && hasToken;

  return (
    <SessionContext.Provider
      value={{
        user,
        isAuthenticated,
        isHydrated,
        setUser,
        clearSession,
      }}
    >
      {children}
    </SessionContext.Provider>
  );
};

// eslint-disable-next-line react-refresh/only-export-components
export const useSession = (): SessionContextValue => {
  const context = useContext(SessionContext);

  if (!context) {
    throw new Error(SESSION_ERRORS.CONTEXT_REQUIRED);
  }

  return context;
};
