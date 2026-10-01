import React, {
  createContext,
  type ReactNode,
  useCallback,
  useContext,
  useEffect,
  useState,
} from "react";

import { sessionApi } from "@services/api";
import {
  extractOrganizationId,
  extractOrganizationName,
  extractUserId,
} from "@utils/jwt";
import { storageService } from "@utils/storage";
import { DEFAULT_ORG_NAME, STORAGE_KEYS } from "@constants/app";
import { APP_ERRORS } from "@constants/messages";

import { useSession } from "./SessionContext";

/**
 * Organization Context
 *
 * GOVERNANCE: This context is the ONLY source of tenant identifiers.
 * All API hooks and router guards must consume it instead of local state.
 *
 * IMPORTANT: No tenantId exposed - tenant resolution happens server-side
 * via orgId→tenantId resolver. Frontend only knows organization, never tenant.
 *
 * Falls back to user profile's defaultOrgId when JWT claim is missing.
 * Requires SessionProvider to wrap OrganizationProvider in App.tsx.
 */

interface OrganizationContextValue {
  organizationId: string | null;
  organizationName: string | null;
  userId: string | null;
  setOrganization: (id: string, name: string, userId?: string) => void;
  clearOrganization: () => void;
}

const OrganizationContext = createContext<OrganizationContextValue | undefined>(
  undefined,
);

export const OrganizationProvider: React.FC<{ children: ReactNode }> = ({
  children,
}) => {
  // No dev fallbacks - org/user must come from JWT token or session profile (fail closed for tenant isolation)
  const [organizationId, setOrgId] = useState<string | null>(null);
  const [organizationName, setOrgName] = useState<string | null>(null);
  const [userId, setUserId] = useState<string | null>(null);

  // Get session user for fallback org resolution
  const { user, isHydrated } = useSession();

  // The transport is scoped before the context changes, so no descendant's
  // request can start under an organization the transport does not send yet.
  const applyOrganization = useCallback(
    (orgId: string | null, orgName: string | null, uid: string | null) => {
      sessionApi.setOrganization(orgId, uid);
      setOrgId(orgId);
      setOrgName(orgName);
      setUserId(uid);
    },
    [],
  );

  // Primary: Extract org from JWT token
  useEffect(() => {
    const token = storageService.getItem(STORAGE_KEYS.AUTH_TOKEN);
    if (!token) return;

    const orgId = extractOrganizationId(token);
    applyOrganization(
      orgId,
      orgId ? extractOrganizationName(token) || DEFAULT_ORG_NAME : null,
      extractUserId(token),
    );
  }, [applyOrganization]);

  // Fallback to session profile's defaultOrgId when JWT claim is missing
  useEffect(() => {
    if (!isHydrated || organizationId || !user?.defaultOrgId) return;

    const membership = user.memberships?.find(
      (m) => m.orgId === user.defaultOrgId,
    );
    applyOrganization(
      user.defaultOrgId,
      membership?.orgName || DEFAULT_ORG_NAME,
      userId || user.id || null,
    );
  }, [isHydrated, user, organizationId, userId, applyOrganization]);

  // Clear org context when session is invalidated (auth failure, logout)
  useEffect(() => {
    if (!isHydrated) return;
    if (!user && !storageService.getItem(STORAGE_KEYS.AUTH_TOKEN)) {
      applyOrganization(null, null, null);
    }
  }, [isHydrated, user, applyOrganization]);

  const setOrganization = (id: string, name: string, uid?: string) => {
    applyOrganization(id, name, uid || userId);
  };

  const clearOrganization = () => {
    applyOrganization(null, null, null);
  };

  return (
    <OrganizationContext.Provider
      value={{
        organizationId,
        organizationName,
        userId,
        setOrganization,
        clearOrganization,
      }}
    >
      {children}
    </OrganizationContext.Provider>
  );
};

// eslint-disable-next-line react-refresh/only-export-components
export const useOrganization = (): OrganizationContextValue => {
  const context = useContext(OrganizationContext);

  if (!context) {
    throw new Error(APP_ERRORS.ORGANIZATION_CONTEXT_REQUIRED);
  }

  return context;
};
