/**
 * Router Barrel Export
 *
 * Centralized routing exports.
 * Import via '@router' alias.
 *
 * @example
 * import { AppRouter, routes, FeatureProtectedRoute } from '@router';
 */

// Main router component
export { default as AppRouter } from './AppRouter';
export { AppRouter as AppRouterNamed } from './AppRouter';

// Route configuration
export type { RouteConfig } from './routes';
export { getRouteByPath, getRouteTitleByPath, routes } from './routes';

// Route protection components
export { AuthGuard } from './AuthGuard';
export { FeatureProtectedRoute } from './FeatureProtectedRoute';
export { RouteErrorBoundary } from './RouteErrorBoundary';
