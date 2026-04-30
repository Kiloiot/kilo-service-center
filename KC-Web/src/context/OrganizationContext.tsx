import React, { createContext, type ReactNode, useContext, useEffect, useState } from 'react';

import { extractOrganizationId, extractOrganizationName, extractUserId } from '@utils/jwt';
import { storageService } from '@utils/storage';
import { DEFAULT_ORG_NAME, STORAGE_KEYS } from '@constants/app';

import { useSession } from './SessionContext';

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

const OrganizationContext = createContext<OrganizationContextValue | undefined>(undefined);

export const OrganizationProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
  // No dev fallbacks - org/user must come from JWT token or session profile (fail closed for tenant isolation)
  const [organizationId, setOrgId] = useState<string | null>(null);
  const [organizationName, setOrgName] = useState<string | null>(null);
  const [userId, setUserId] = useState<string | null>(null);

  // Get session user for fallback org resolution
  const { user, isHydrated } = useSession();

  // Primary: Extract org from JWT token
  useEffect(() => {
    const token = storageService.getItem(STORAGE_KEYS.AUTH_TOKEN);
    if (!token) return;

    const orgId = extractOrganizationId(token);
    const orgName = extractOrganizationName(token);
    const uid = extractUserId(token);

    if (orgId) {
      setOrgId(orgId);
      setOrgName(orgName || DEFAULT_ORG_NAME);
    }

    // NOTE: No tenantId extraction - tenant resolution is server-side only
    // per AGENTS governance. Frontend only knows organization.

    if (uid) {
      setUserId(uid);
    }
  }, []);

  // Fallback to session profile's defaultOrgId when JWT claim is missing
  useEffect(() => {
    // Only attempt fallback after session is hydrated and if we don't have an org yet
    if (!isHydrated || organizationId) return;

    // If we have a user with a defaultOrgId, use it as fallback
    if (user?.defaultOrgId) {
      setOrgId(user.defaultOrgId);

      // Find the org name from memberships
      const membership = user.memberships?.find((m) => m.orgId === user.defaultOrgId);
      setOrgName(membership?.orgName || DEFAULT_ORG_NAME);

      // Also set userId if not already set
      if (!userId && user.id) {
        setUserId(user.id);
      }
    }
  }, [isHydrated, user, organizationId, userId]);

  // Clear org context when session is invalidated (auth failure, logout)
  useEffect(() => {
    if (!isHydrated) return;
    if (!user && !storageService.getItem(STORAGE_KEYS.AUTH_TOKEN)) {
      setOrgId(null);
      setOrgName(null);
      setUserId(null);
    }
  }, [isHydrated, user]);

  const setOrganization = (id: string, name: string, uid?: string) => {
    setOrgId(id);
    setOrgName(name);
    if (uid) setUserId(uid);
  };

  const clearOrganization = () => {
    setOrgId(null);
    setOrgName(null);
    setUserId(null);
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
    throw new Error('useOrganization must be used within OrganizationProvider');
  }

  return context;
};
