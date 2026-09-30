import { useQuery } from "@tanstack/react-query";

import { systemApi } from "@services/api";
import { queryKeys } from "@config/query-keys";

export function useVersionInfo() {
  return useQuery({
    queryKey: queryKeys.system.version(),
    queryFn: () => systemApi.getVersion(),
  });
}
