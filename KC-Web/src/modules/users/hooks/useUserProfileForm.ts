import { useEffect, useState } from "react";

import type { SystemUserUI } from "@api-types/api";
import { useUpdateUser } from "@hooks";

import { useFeedback } from "@contexts/feedback";
import { getErrorMessage } from "@utils/error-message";
import { MSG_USER_UPDATED, USER_FORM } from "@constants/messages";

import {
  buildUpdateUserPayload,
  EMPTY_USER_PROFILE_FORM,
  type UserProfileForm,
  userProfileFormFrom,
} from "../utils/user-profile";

export function useUserProfileForm(user: SystemUserUI | null | undefined) {
  const [form, setForm] = useState<UserProfileForm>(EMPTY_USER_PROFILE_FORM);
  const feedback = useFeedback();
  const updateUser = useUpdateUser();

  useEffect(() => {
    if (user) setForm(userProfileFormFrom(user));
  }, [user]);

  const updateForm = <K extends keyof UserProfileForm>(
    key: K,
    value: UserProfileForm[K],
  ) => setForm((prev) => ({ ...prev, [key]: value }));

  const save = async () => {
    if (!user) return;
    const updates = buildUpdateUserPayload(form, user);
    if (Object.keys(updates).length === 0) return;
    try {
      await updateUser.mutateAsync({ id: user.id, data: updates });
      feedback.success(MSG_USER_UPDATED);
    } catch (error) {
      feedback.error(getErrorMessage(error, USER_FORM.ERR_UPDATE_FAILED));
    }
  };

  return {
    form,
    updateForm,
    save,
    isSaving: updateUser.isPending,
  };
}
