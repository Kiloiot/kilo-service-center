import { useState } from "react";

import { endOfDay } from "date-fns";

import { useCreateApiKey } from "@hooks/useApiKeys";
import { parseDateInputEU } from "@utils/date-format";
import { getErrorMessage } from "@utils/error-message";
import { API_KEY_TYPES } from "@constants/app";
import { ERR_CREATE_API_KEY } from "@constants/messages";

/** The day a key expires on (DD/MM/YYYY) becomes the last moment of that day. */
function expiryOf(day: Date | undefined): Date | undefined {
  return day ? endOfDay(day) : undefined;
}

/**
 * State of the Create API Key form: its fields, their validation, and the
 * server's reason when the key could not be created.
 */
export function useApiKeyForm(onCreated: (rawKey: string) => void) {
  const createMutation = useCreateApiKey();
  const [name, setName] = useState("");
  const [keyType, setKeyType] = useState<string>(API_KEY_TYPES.USER);
  const [expiresOn, setExpiresOn] = useState("");
  const [error, setError] = useState<string | null>(null);

  const expiresOnDay = parseDateInputEU(expiresOn);
  const expiresOnInvalid = expiresOn !== "" && !expiresOnDay;
  const canSubmit =
    !!name.trim() && !expiresOnInvalid && !createMutation.isPending;

  const reset = () => {
    setName("");
    setKeyType(API_KEY_TYPES.USER);
    setExpiresOn("");
    setError(null);
  };

  const submit = () => {
    if (!canSubmit) return;
    setError(null);
    createMutation.mutate(
      { name: name.trim(), keyType, expiresAt: expiryOf(expiresOnDay) },
      {
        onSuccess: (response) => {
          reset();
          onCreated(response.rawKey);
        },
        onError: (err) => setError(getErrorMessage(err, ERR_CREATE_API_KEY)),
      },
    );
  };

  return {
    fields: { name, keyType, expiresOn, expiresOnInvalid },
    setName,
    setKeyType,
    setExpiresOn,
    error,
    pending: createMutation.isPending,
    canSubmit,
    submit,
    reset,
  };
}
