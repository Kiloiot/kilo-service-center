import { useState } from "react";

import { useCreateUser } from "@hooks";

import { useFeedback } from "@contexts/feedback";
import { useAddOrgUser } from "@hooks/useOrganizations";
import { getErrorMessage } from "@utils/error-message";
import { ORG_ROLE } from "@constants/app";
import { MSG_USER_CREATED, USER_FORM } from "@constants/messages";

import {
  createUserRequest,
  EMPTY_NEW_USER_FORM,
  type NewUserErrors,
  type NewUserForm,
  validateNewUser,
} from "../utils/new-user-form";

type AddOrgUser = ReturnType<typeof useAddOrgUser>["mutateAsync"];

/**
 * Adds a new user to the given organization, or to the ones picked in the
 * form; reports whether every addition succeeded.
 */
async function joinOrganizations(
  addOrgUser: AddOrgUser,
  orgId: string | undefined,
  userId: string | undefined,
  form: NewUserForm,
): Promise<boolean> {
  if (!userId) return true;
  const orgIds = orgId ? [orgId] : form.organizations.map((o) => o.id);
  const results = await Promise.allSettled(
    orgIds.map((orgId) =>
      addOrgUser({
        orgId,
        data: {
          user_id: userId,
          role: ORG_ROLE.MEMBER,
          is_org_admin: false,
          is_base_station_admin: form.isBaseStationManager,
          is_endpoint_admin: form.isEndpointManager,
        },
      }),
    ),
  );
  return results.every((result) => result.status === "fulfilled");
}

/**
 * State of the Add User dialog: creates the user, then adds it to the
 * organizations it was given (orgId, or the ones the admin picked).
 */
export function useAddUserForm(orgId: string | undefined, onDone: () => void) {
  const createUser = useCreateUser();
  const addOrgUser = useAddOrgUser();
  const feedback = useFeedback();
  const [form, setForm] = useState<NewUserForm>(EMPTY_NEW_USER_FORM);
  const [errors, setErrors] = useState<NewUserErrors>({});
  const [partialError, setPartialError] = useState("");

  const update = <K extends keyof NewUserForm>(key: K, value: NewUserForm[K]) =>
    setForm((prev) => ({ ...prev, [key]: value }));

  const reset = () => {
    setForm(EMPTY_NEW_USER_FORM);
    setErrors({});
    setPartialError("");
    createUser.reset();
  };

  const submit = async () => {
    const found = validateNewUser(form);
    setErrors(found);
    if (Object.keys(found).length > 0) return;
    try {
      const user = await createUser.mutateAsync(createUserRequest(form));
      const add = addOrgUser.mutateAsync;
      if (await joinOrganizations(add, orgId, user?.id, form)) {
        reset();
        onDone();
        feedback.success(MSG_USER_CREATED);
      } else setPartialError(USER_FORM.ERR_ADD_TO_ORG_PARTIAL);
    } catch {
      // The failure is read back from the mutation state as createError.
    }
  };

  return {
    form,
    update,
    errors,
    partialError,
    createError: createUser.isError
      ? getErrorMessage(createUser.error, USER_FORM.ERR_CREATE_FAILED)
      : null,
    pending: createUser.isPending,
    submit,
    reset,
  };
}
