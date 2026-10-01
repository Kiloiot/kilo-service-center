/**
 * SCACI control-plane hooks. Session opens, resumes, losses, refused
 * connects and SCACI errors refresh the views through realtime invalidation
 * as they happen. Ping times, which no event carries, stay current through
 * polling.
 */

import type { PageRequest } from "@api-types/pagination";
import { keepPreviousData, useQuery } from "@tanstack/react-query";

import { scaciApi } from "@services/api";
import { TIMING_LIVE_POLL } from "@constants/app";
import { queryKeys } from "@config/query-keys";

const refetchInterval = TIMING_LIVE_POLL.CONTROL_PLANE_MS;

export function useScaciStatus() {
  return useQuery({
    queryKey: queryKeys.scaci.status(),
    queryFn: () => scaciApi.getStatus(),
    refetchInterval,
  });
}

export function useScaciSessions(page: PageRequest) {
  return useQuery({
    queryKey: queryKeys.scaci.sessions(page),
    queryFn: () => scaciApi.listSessions(page),
    placeholderData: keepPreviousData,
    refetchInterval,
  });
}
