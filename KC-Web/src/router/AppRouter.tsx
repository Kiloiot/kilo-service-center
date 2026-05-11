/**
 * Application Router
 *
 * Centralized routing component that maps route configuration
 * to React Router routes with feature protection.
 *
 * Default export for lazy loading compatibility.
 */

import React from "react";
import { Navigate, Route, Routes } from "react-router-dom";

import { FeatureProtectedRoute } from "./FeatureProtectedRoute";
import { routes } from "./routes";

/**
 * Application Router Component
 *
 * Maps route configuration to React Router routes.
 * Each route is wrapped with FeatureProtectedRoute for:
 * - Feature flag checking
 * - Suspense boundary (lazy loading)
 * - RouteErrorBoundary (error isolation)
 *
 * @example
 * // In App.tsx
 * <AppRouter />
 */
const AppRouter: React.FC = () => {
  return (
    <Routes>
      {routes.map((route) => (
        <Route
          key={route.path}
          path={route.path}
          element={
            <FeatureProtectedRoute
              featureFlag={route.featureFlag}
              routeName={route.title}
            >
              <route.element />
            </FeatureProtectedRoute>
          }
        />
      ))}

      {/* Catch-all redirect to dashboard */}
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
};

// Default export for React Router lazy loading compatibility
export default AppRouter;

// Named export for explicit imports
export { AppRouter };
