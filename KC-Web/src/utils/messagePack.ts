/**
 * MessagePack Utility for MIOTY Protocol
 * Handles MIOTYA01 header parsing per MIOTY_SC-AC §3.1
 */

import { logger } from '@utils/logger';

export interface MIOTYHeader {
  version: string;
  type: string;
  length: number;
  raw: string;
}

/**
 * Parse MIOTYA01 header from MessagePack payload
 * Per MIOTY_SC-AC §3.1 specification
 *
 * @param payload - Base64-encoded MessagePack payload
 * @returns Parsed header information or null if invalid
 */
export const parseMIOTYA01Header = (payload: string): MIOTYHeader | null => {
  if (!payload) return null;

  try {
    const bytes = atob(payload);

    // MIOTYA01 header is typically first 8 bytes
    // Format: [version][type][length]
    if (bytes.length < 8) {
      return null;
    }

    const headerBytes = bytes.substring(0, 8);
    const hex = Array.from(headerBytes, (c) => c.charCodeAt(0).toString(16).padStart(2, '0')).join(
      ''
    );

    // Parse version (first 2 bytes)
    const version =
      headerBytes.charCodeAt(0).toString() + '.' + headerBytes.charCodeAt(1).toString();

    // Parse type (byte 3)
    const type = headerBytes.charCodeAt(2).toString(16).padStart(2, '0');

    // Parse length (bytes 4-7, big-endian)
    const length =
      ((headerBytes.charCodeAt(4) << 24) |
        (headerBytes.charCodeAt(5) << 16) |
        (headerBytes.charCodeAt(6) << 8) |
        headerBytes.charCodeAt(7)) >>>
      0;

    return {
      version,
      type: `0x${type}`,
      length,
      raw: `0x${hex}`,
    };
  } catch (error) {
    logger.error('Failed to parse MIOTYA01 header:', error);
    return null;
  }
};

/**
 * Validate MIOTYA01 header format
 *
 * @param header - Parsed header to validate
 * @returns True if header is valid
 */
export const validateMIOTYA01Header = (header: MIOTYHeader | null): boolean => {
  if (!header) return false;

  // Basic validation: version should be reasonable, length should be positive
  const versionParts = header.version.split('.');
  if (versionParts.length !== 2) return false;

  const major = parseInt(versionParts[0], 10);
  const minor = parseInt(versionParts[1], 10);

  if (isNaN(major) || isNaN(minor)) return false;
  if (major < 0 || major > 255 || minor < 0 || minor > 255) return false;
  if (header.length < 0 || header.length > 1048576) return false; // Max 1MB

  return true;
};

/**
 * Extract payload data (excluding header)
 *
 * @param payload - Base64-encoded MessagePack payload
 * @returns Payload data without MIOTYA01 header, or original if no valid header
 */
export const extractPayloadData = (payload: string): string => {
  if (!payload) return '';

  try {
    const bytes = atob(payload);

    // Skip first 8 bytes (MIOTYA01 header)
    if (bytes.length <= 8) {
      return payload; // Return original if too short
    }

    const dataBytes = bytes.substring(8);
    return btoa(dataBytes);
  } catch (error) {
    logger.error('Failed to extract payload data:', error);
    return payload;
  }
};
