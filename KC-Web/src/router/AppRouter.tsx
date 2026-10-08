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

import { ROUTE_CATCH_ALL, ROUTES } from "@constants/app";

import { FeatureProtectedRoute } from "./FeatureProtectedRoute";
import { RoleProtectedRoute } from "./RoleProtectedRoute";
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
              <RoleProtectedRoute requires={route.requires}>
                <route.element />
              </RoleProtectedRoute>
            </FeatureProtectedRoute>
          }
        />
      ))}

      {/* Catch-all redirect to dashboard */}
      <Route
        path={ROUTE_CATCH_ALL}
        element={<Navigate to={ROUTES.HOME} replace />}
      />
    </Routes>
  );
};

// Default export for React Router lazy loading compatibility
export default AppRouter;
