import { formatPasswordRule } from "@utils/formatters";

import { useAuthSettings } from "./useAuthSettings";

/** The password rule the identity service publishes, once its settings loaded. */
export function usePasswordRule(): string | undefined {
  const { data } = useAuthSettings();
  return data?.password_policy
    ? formatPasswordRule(data.password_policy)
    : undefined;
}
