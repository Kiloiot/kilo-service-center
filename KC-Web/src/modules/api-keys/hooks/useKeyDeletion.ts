import { useState } from "react";

import type { ApiKeyAPI } from "@api-types/api";

import { useFeedback } from "@contexts/feedback";
import { useDeleteApiKey } from "@hooks/useApiKeys";
import { MSG_API_KEY_DELETED } from "@constants/messages";

/** Deletion of one key behind a confirmation; a refusal stays in the dialog. */
export function useKeyDeletion() {
  const deleteMutation = useDeleteApiKey();
  const feedback = useFeedback();
  const [keyToDelete, setKeyToDelete] = useState<ApiKeyAPI | null>(null);

  const confirm = async () => {
    if (!keyToDelete) return;
    await deleteMutation.mutateAsync(keyToDelete.id);
    setKeyToDelete(null);
    feedback.success(MSG_API_KEY_DELETED);
  };

  return {
    keyToDelete,
    request: setKeyToDelete,
    cancel: () => setKeyToDelete(null),
    confirm,
  };
}
