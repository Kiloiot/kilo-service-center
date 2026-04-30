/**
 * Storage Service
 * Provides safe localStorage access with centralized error handling
 *
 * Errors are logged for central capture rather than silently failing.
 * This abstraction allows easy mocking in tests and future storage backend swaps.
 */

import { logger } from '@utils/logger';
import {
  ERR_STORAGE_CLEAR_FAILED,
  ERR_STORAGE_GET_FAILED,
  ERR_STORAGE_REMOVE_FAILED,
  ERR_STORAGE_SET_FAILED,
} from '@constants/messages';

/**
 * Storage interface for dependency injection and testing
 */
export interface IStorage {
  getItem(key: string): string | null;
  setItem(key: string, value: string): void;
  removeItem(key: string): void;
  clear(): void;
}

/**
 * LocalStorageService provides safe localStorage access with centralized error handling.
 *
 * Operations fail gracefully (return null / no-op on error).
 * All error messages use centralized constants from constants/messages.ts.
 */
class LocalStorageService implements IStorage {
  /**
   * Safely retrieve item from localStorage.
   * @param key - Storage key to retrieve
   * @returns Stored value or null if not found/error
   */
  getItem(key: string): string | null {
    try {
      return localStorage.getItem(key);
    } catch (error) {
      logger.error(ERR_STORAGE_GET_FAILED.replace('{key}', key), error);
      return null;
    }
  }

  /**
   * Safely store item in localStorage.
   * @param key - Storage key
   * @param value - Value to store (must be string)
   */
  setItem(key: string, value: string): void {
    try {
      localStorage.setItem(key, value);
    } catch (error) {
      logger.error(ERR_STORAGE_SET_FAILED.replace('{key}', key), error);
    }
  }

  /**
   * Safely remove item from localStorage.
   * @param key - Storage key to remove
   */
  removeItem(key: string): void {
    try {
      localStorage.removeItem(key);
    } catch (error) {
      logger.error(ERR_STORAGE_REMOVE_FAILED.replace('{key}', key), error);
    }
  }

  /**
   * Safely clear all items from localStorage.
   */
  clear(): void {
    try {
      localStorage.clear();
    } catch (error) {
      logger.error(ERR_STORAGE_CLEAR_FAILED, error);
    }
  }
}

/**
 * Singleton storage service instance
 * Import this in your components/contexts to access storage
 *
 * Example:
 * ```typescript
 * import { storageService } from '../utils/storage';
 *
 * const token = storageService.getItem(STORAGE_KEYS.AUTH_TOKEN);
 * storageService.setItem(STORAGE_KEYS.AUTH_TOKEN, newToken);
 * ```
 */
export const storageService = new LocalStorageService();
