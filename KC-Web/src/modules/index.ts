/**
 * Modules Root Barrel Export
 *
 * Feature-first module organization.
 * Import modules via '@modules/*' aliases.
 *
 * @example
 * import { Dashboard } from '@modules/dashboard';
 * import { BaseStations } from '@modules/base-stations';
 */

// Module re-exports (for convenience, prefer direct module imports)
export * from './auth';
export * from './base-stations';
export * from './certificates';
export * from './dashboard';
export * from './endpoints';
export * from './organizations';
export * from './users';
