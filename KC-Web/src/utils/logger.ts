/**
 * Frontend Logger
 *
 * Provides a centralized logging interface that is silenced in production.
 * All source files must use this logger instead of console directly.
 */

import { isDevelopment } from '@config/env';

// eslint-disable-next-line @typescript-eslint/no-unused-vars
const silent = (..._: unknown[]): void => {};

export const logger = {
  // logger.ts is excluded from the no-console rule (see eslint.config.js ignores)
  error: isDevelopment ? console.error : silent,
  warn: isDevelopment ? console.warn : silent,
};
