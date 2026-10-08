/**
 * Mounts a panel only for a user whose roles satisfy its requirement; anyone
 * else sees the no-access panel and the panel's requests are never made.
 */

import type { ReactNode } from "react";
import React from "react";

import NoAccessPanel from "@components/common/NoAccessPanel";
import { useCapabilities } from "@hooks/useCapabilities";
import type { RoleRequirement } from "@constants/app";

interface RoleGateProps {
  requires: RoleRequirement;
  children: ReactNode;
}

export const RoleGate: React.FC<RoleGateProps> = ({ requires, children }) => {
  const { rolesLoaded, can } = useCapabilities();
  if (!rolesLoaded) {
    return null;
  }
  if (!can(requires)) {
    return <NoAccessPanel />;
  }
  return <>{children}</>;
};
