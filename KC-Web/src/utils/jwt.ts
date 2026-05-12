/**
 * JWT Utility Functions
 * RFC 7519 compliant JWT parsing with base64url support
 */

import { logger } from "@utils/logger";
import { externalOrgClaimPath } from "@config/env";

/**
 * Decode JWT payload with proper base64url handling
 *
 * @param token - Full JWT token (header.payload.signature)
 * @returns Decoded JWT payload as object, or null on error
 *
 * Standards:
 * - RFC 7519: JWT standard using base64url encoding
 * - Handles URL-safe characters: - and _
 * - Supports Unicode characters in claims
 */
export function decodeJwtPayload(
  token: string,
): Record<string, unknown> | null {
  if (!token || token.split(".").length !== 3) {
    return null;
  }

  try {
    // Step 1: Extract payload (middle part of JWT)
    const base64Url = token.split(".")[1];

    // Step 2: Convert base64url → base64 (RFC 7519 standard)
    // Replace URL-safe characters: - → +, _ → /
    const base64 = base64Url.replace(/-/g, "+").replace(/_/g, "/");

    // Step 3: Decode base64 with Unicode support
    // Handles special characters in organization names, user names, etc.
    const jsonPayload = decodeURIComponent(
      atob(base64)
        .split("")
        .map((c) => "%" + ("00" + c.charCodeAt(0).toString(16)).slice(-2))
        .join(""),
    );

    // Step 4: Parse JSON payload
    return JSON.parse(jsonPayload);
  } catch (error) {
    logger.error("Failed to decode JWT payload:", error);
    return null;
  }
}

/**
 * Extract organization ID from external IdP JWT
 * Uses configurable claim path from VITE_EXTERNAL_ORG_CLAIM_PATH env var.
 *
 * @param token - Full JWT token
 * @returns Organization UUID string, or null if not found/invalid
 */
export function extractOrganizationId(token: string): string | null {
  const payload = decodeJwtPayload(token);
  if (!payload) return null;

  const orgId = payload[externalOrgClaimPath];

  // Validate UUID format
  const uuidRegex =
    /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

  return typeof orgId === "string" && uuidRegex.test(orgId) ? orgId : null;
}

/**
 * Extract organization name from external IdP JWT
 *
 * @param token - Full JWT token
 * @returns Organization name string, or null if not found
 */
export function extractOrganizationName(token: string): string | null {
  const payload = decodeJwtPayload(token);
  const orgName = payload?.organization_name;
  return typeof orgName === "string" ? orgName : null;
}

/**
 * Extract user ID from external IdP JWT
 * Uses 'sub' claim as canonical user identifier for X-User-ID header
 *
 * @param token - Full JWT token
 * @returns User UUID from sub claim, or null if not found/invalid
 */
export function extractUserId(token: string): string | null {
  const payload = decodeJwtPayload(token);
  if (!payload) return null;

  const userId = payload.sub;

  // Validate UUID format
  const uuidRegex =
    /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

  return typeof userId === "string" && uuidRegex.test(userId) ? userId : null;
}

/**
 * Extract tenant ID from external IdP JWT
 * Uses same configurable claim path as organization ID for tenant context propagation.
 *
 * @param token - Full JWT token
 * @returns Tenant UUID string, or null if not found/invalid
 */
export function extractTenantId(token: string): string | null {
  const payload = decodeJwtPayload(token);
  if (!payload) return null;

  const tenantId = payload[externalOrgClaimPath];

  // Validate UUID format
  const uuidRegex =
    /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

  return typeof tenantId === "string" && uuidRegex.test(tenantId)
    ? tenantId
    : null;
}
