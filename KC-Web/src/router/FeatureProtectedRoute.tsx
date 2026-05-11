/**
 * Feature Protected Route
 *
 * Guards routes based on feature flags.
 * Wraps route content with Suspense and ErrorBoundary.
 */

import type { ReactNode } from "react";
import React, { Suspense } from "react";
import { Navigate, useLocation } from "react-router-dom";

import GlobalLoader from "@components/common/GlobalLoader";
import {
  type FeatureFlagName,
  useFeatureFlags,
} from "@contexts/FeatureFlagContext";

import { RouteErrorBoundary } from "./RouteErrorBoundary";

interface FeatureProtectedRouteProps {
  children: ReactNode;
  /** Feature flag that must be enabled for this route */
  featureFlag?: FeatureFlagName;
  /** Route name for error reporting */
  routeName?: string;
  /** Custom loading component */
  loadingFallback?: ReactNode;
}

/**
 * Feature-protected route wrapper
 *
 * Checks feature flag before rendering route content.
 * Provides Suspense boundary for lazy-loaded components.
 * Wraps content in RouteErrorBoundary for error isolation.
 *
 * @example
 * <FeatureProtectedRoute featureFlag="downlink_management" routeName="Downlink">
 *   <DownlinkQueue />
 * </FeatureProtectedRoute>
 */
export const FeatureProtectedRoute: React.FC<FeatureProtectedRouteProps> = ({
  children,
  featureFlag,
  routeName = "Route",
  loadingFallback = <GlobalLoader />,
}) => {
  const { isEnabled, loading } = useFeatureFlags();
  const location = useLocation();

  // Show loader while flags are loading
  if (loading) {
    return <>{loadingFallback}</>;
  }

  // Check feature flag if specified
  if (featureFlag && !isEnabled(featureFlag)) {
    // Navigate to home with message about disabled feature
    // In the future, could show a dedicated "Feature Disabled" page
    return (
      <Navigate
        to="/"
        state={{ from: location, disabledFeature: featureFlag }}
        replace
      />
    );
  }

  // Render route with Suspense and ErrorBoundary
  return (
    <RouteErrorBoundary routeName={routeName}>
      <Suspense fallback={loadingFallback}>{children}</Suspense>
    </RouteErrorBoundary>
  );
};
