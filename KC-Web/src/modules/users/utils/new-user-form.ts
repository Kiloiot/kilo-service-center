/**
 * The Add User form: its fields, validation and the request it builds.
 */

import type { CreateUserRequest, OrganizationUI } from "@api-types/api";

import { PASSWORD_RULES, USER_FORM } from "@constants/messages";

export interface NewUserForm {
  email: string;
  password: string;
  confirmPassword: string;
  note: string;
  isActive: boolean;
  isAdmin: boolean;
  isTenantManager: boolean;
  isBaseStationManager: boolean;
  isEndpointManager: boolean;
  organizations: OrganizationUI[];
}

export type NewUserFlag = keyof Pick<
  NewUserForm,
  | "isActive"
  | "isAdmin"
  | "isTenantManager"
  | "isBaseStationManager"
  | "isEndpointManager"
>;

export const EMPTY_NEW_USER_FORM: NewUserForm = {
  email: "",
  password: "",
  confirmPassword: "",
  note: "",
  isActive: true,
  isAdmin: false,
  isTenantManager: false,
  isBaseStationManager: false,
  isEndpointManager: false,
  organizations: [],
};

export interface NewUserErrors {
  email?: string;
  password?: string;
  confirmPassword?: string;
}

export function validateNewUser(form: NewUserForm): NewUserErrors {
  const errors: NewUserErrors = {};
  if (!form.email.trim()) errors.email = USER_FORM.ERR_EMAIL_REQUIRED;
  if (!form.password) errors.password = USER_FORM.ERR_PASSWORD_REQUIRED;
  if (form.password !== form.confirmPassword) {
    errors.confirmPassword = PASSWORD_RULES.ERR_MISMATCH;
  }
  return errors;
}

export function createUserRequest(form: NewUserForm): CreateUserRequest {
  const note = form.note.trim();
  return {
    email: form.email.trim(),
    password: form.password,
    is_admin: form.isAdmin,
    is_active: form.isActive,
    is_tenant_manager: form.isTenantManager,
    is_base_station_manager: form.isBaseStationManager,
    is_endpoint_manager: form.isEndpointManager,
    ...(note && { note }),
  };
}
