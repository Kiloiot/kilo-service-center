/**
 * Role Protected Route
 *
 * Admits a route only when the signed-in user's roles satisfy its
 * requirement. A user without any role sees the no-access state instead of
 * the page; a user whose roles do not cover the page is sent home.
 */

import type { ReactNode } from "react";
import React from "react";
import { Navigate } from "react-router-dom";

import GlobalLoader from "@components/common/GlobalLoader";
import NoAccessState from "@components/common/NoAccessState";
import { useCapabilities } from "@hooks/useCapabilities";
import { type RoleRequirement, ROUTES } from "@constants/app";

interface RoleProtectedRouteProps {
  children: ReactNode;
  /** Requirement the route places on the user's roles; none for public pages. */
  requires?: RoleRequirement;
}

export const RoleProtectedRoute: React.FC<RoleProtectedRouteProps> = ({
  children,
  requires,
}) => {
  const { rolesLoaded, hasAnyRole, can } = useCapabilities();

  if (!requires) {
    return <>{children}</>;
  }
  if (!rolesLoaded) {
    return <GlobalLoader />;
  }
  if (!hasAnyRole) {
    return <NoAccessState />;
  }
  if (!can(requires)) {
    return <Navigate to={ROUTES.HOME} replace />;
  }
  return <>{children}</>;
};
