/**
 * Session wiring: the organization/user context every authenticated call is
 * scoped to, the callback invoked when a call stays unauthenticated, and the
 * token refresh the transport shares across tabs.
 */

import type { RefreshOutcome } from "@services/grpc/refreshCoordinator";
import {
  type AuthFailureCallback,
  grpcTransport,
} from "@services/grpc/transport";

export const sessionApi = {
  clearOrganization(): void {
    grpcTransport.clearOrganization();
  },

  setOrganization(
    orgId: string | null | undefined,
    userId?: string | null | undefined,
  ): void {
    grpcTransport.setOrganization(orgId, userId);
  },

  setAuthFailureCallback(callback: AuthFailureCallback | undefined): void {
    grpcTransport.setAuthFailureCallback(callback);
  },

  refreshTokens(): Promise<RefreshOutcome> {
    return grpcTransport.refreshTokens();
  },
};
