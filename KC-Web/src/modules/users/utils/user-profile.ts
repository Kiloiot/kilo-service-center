import type { SystemUserUI, UpdateUserRequest } from "@api-types/api";

export interface UserProfileForm {
  email: string;
  note: string;
  isUserAdmin: boolean;
  isActive: boolean;
  isTenantManager: boolean;
  isBaseStationManager: boolean;
  isEndpointManager: boolean;
}

/**
 * Builds an UpdateUserRequest containing only the fields that differ from the
 * loaded user record. Returns an empty object when nothing changed.
 */
export function buildUpdateUserPayload(
  form: UserProfileForm,
  original: SystemUserUI,
): UpdateUserRequest {
  const updates: UpdateUserRequest = {};
  if (form.email !== original.email) updates.email = form.email;
  if (form.note !== (original.note || "")) {
    updates.note = form.note || undefined;
  }
  if (form.isUserAdmin !== original.isAdmin)
    updates.is_admin = form.isUserAdmin;
  if (form.isActive !== original.isActive) updates.is_active = form.isActive;
  if (form.isTenantManager !== original.isTenantManager) {
    updates.is_tenant_manager = form.isTenantManager;
  }
  if (form.isBaseStationManager !== original.isBaseStationManager) {
    updates.is_base_station_manager = form.isBaseStationManager;
  }
  if (form.isEndpointManager !== original.isEndpointManager) {
    updates.is_endpoint_manager = form.isEndpointManager;
  }
  return updates;
}

export const EMPTY_USER_PROFILE_FORM: UserProfileForm = {
  email: "",
  note: "",
  isUserAdmin: false,
  isActive: true,
  isTenantManager: false,
  isBaseStationManager: false,
  isEndpointManager: false,
};

export function userProfileFormFrom(user: SystemUserUI): UserProfileForm {
  return {
    email: user.email,
    note: user.note || "",
    isUserAdmin: user.isAdmin,
    isActive: user.isActive,
    isTenantManager: user.isTenantManager,
    isBaseStationManager: user.isBaseStationManager,
    isEndpointManager: user.isEndpointManager,
  };
}
