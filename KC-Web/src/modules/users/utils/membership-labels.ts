/**
 * System users carry admin/active flags; organization members carry a role
 * and a status. These derive one vocabulary for tables and sorting.
 */

import type { OrganizationUserUI, SystemUserUI } from "@api-types/api";

import { memberRoleLabel } from "@utils/chipMappings";
import { ORG_MEMBER_STATUS, ORG_ROLE } from "@constants/app";
import { ROLE_NAMES } from "@constants/messages";

export const isOrgUser = (
  user: SystemUserUI | OrganizationUserUI,
): user is OrganizationUserUI => "orgId" in user;

export const systemUserRole = (user: SystemUserUI): string =>
  user.isAdmin ? ORG_ROLE.ADMIN : ORG_ROLE.MEMBER;

export const systemUserStatus = (user: SystemUserUI): string =>
  user.isActive ? ORG_MEMBER_STATUS.ACTIVE : ORG_MEMBER_STATUS.REMOVED;

export const unifiedRole = (user: SystemUserUI | OrganizationUserUI): string =>
  isOrgUser(user) ? user.role : systemUserRole(user);

/** The role a row names: an organization role, or Admin for an administrator (none otherwise). */
export const roleLabel = (
  user: SystemUserUI | OrganizationUserUI,
): string | null => {
  if (isOrgUser(user)) return memberRoleLabel(user.role);
  return user.isAdmin ? ROLE_NAMES.ADMIN : null;
};

export const unifiedStatus = (
  user: SystemUserUI | OrganizationUserUI,
): string => (isOrgUser(user) ? user.status : systemUserStatus(user));
