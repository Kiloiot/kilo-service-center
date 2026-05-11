/**
 * Dashboard Module
 *
 * Re-exports Dashboard page component.
 * For lazy loading in router, use:
 *   const Dashboard = lazy(() => import('@modules/dashboard').then(m => ({ default: m.Dashboard })));
 */

export { default as Dashboard } from "./pages/Dashboard";
