/**
 * Environment Configuration Module
 *
 * GOVERNANCE: This is the ONLY file allowed to access import.meta.env
 * All other files MUST import from this module.
 *
 * LOCKED EXPORT SURFACE - no additions without governance review:
 * - grpcUrl: gRPC-web connection settings
 * - isProduction, isDevelopment: Environment flags
 * - externalOrgClaimPath: JWT claim path for external IdP org extraction
 * - mapTileUrl, mapTileAttribution: Deployment-configurable map tile provider
 *
 * IMPORTANT: No tenantId - tenant resolution happens server-side via
 * orgId→tenantId resolver. Frontend only knows organization, never tenant.
 * This is enforced by AGENTS governance.
 *
 * SECURITY: No dev org/user fallbacks - authenticated calls fail closed
 * when org/user context is missing. Public methods (login, getAuthSettings,
 * etc.) explicitly skip org/user requirements.
 */

import { DEFAULT_EXTERNAL_ORG_CLAIM_PATH } from "@constants/app";

export interface EnvConfig {
  /** Base URL for gRPC-web requests (empty string uses proxy) */
  grpcUrl: string;
  /** True when running in production mode */
  isProduction: boolean;
  /** True when running in development mode */
  isDevelopment: boolean;
  /** JWT claim path for external IdP organization ID extraction */
  externalOrgClaimPath: string;
  /** Application version from release manifest (centralized versioning) */
  appVersion: string;
  /** Build timestamp from release manifest */
  buildTime: string;
  /** Git commit hash from release manifest */
  gitCommit: string;
  /** Database schema version from release manifest */
  schemaVersion: number;
  /** Custom map tile provider URL (deployment-configurable) */
  mapTileUrl: string;
  /** Custom map tile attribution string (deployment-configurable) */
  mapTileAttribution: string;
}

// ONLY these keys exported - no tenantId, no passthrough import.meta.env
export const env: EnvConfig = {
  // gRPC-web URL - empty string uses Vite proxy in development
  grpcUrl: import.meta.env.VITE_GRPC_URL || "",
  isProduction: import.meta.env.PROD,
  isDevelopment: import.meta.env.DEV,
  externalOrgClaimPath:
    import.meta.env.VITE_EXTERNAL_ORG_CLAIM_PATH ||
    DEFAULT_EXTERNAL_ORG_CLAIM_PATH,
  // Centralized versioning - injected at build time from release/manifest.json
  appVersion: __APP_VERSION__,
  buildTime: __BUILD_TIME__,
  gitCommit: __GIT_COMMIT__,
  schemaVersion: __SCHEMA_VERSION__,
  // Map tile provider (deployment-configurable)
  mapTileUrl: import.meta.env.VITE_MAP_TILE_URL || "",
  mapTileAttribution: import.meta.env.VITE_MAP_TILE_ATTRIBUTION || "",
};

// Named exports for convenience
export const {
  grpcUrl,
  isProduction,
  isDevelopment,
  externalOrgClaimPath,
  appVersion,
  buildTime,
  gitCommit,
  schemaVersion,
  mapTileUrl,
  mapTileAttribution,
} = env;
