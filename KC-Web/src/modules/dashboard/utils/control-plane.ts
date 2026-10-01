import type { ScaciSessionUI, ScaciStatusUI } from "@api-types/system";

import { SCACI_SESSION_STATUS } from "@constants/app";

export const CONTROL_PLANE_STATE = {
  LISTENER_OFFLINE: "listener_offline",
  CONNECTED: "connected",
  DISCONNECTED: "disconnected",
} as const;

export type ControlPlaneState =
  (typeof CONTROL_PLANE_STATE)[keyof typeof CONTROL_PLANE_STATE];

const LIVE_SESSION_STATES: ReadonlySet<string> = new Set([
  SCACI_SESSION_STATUS.ACTIVE,
  SCACI_SESSION_STATUS.RESUMED,
]);

/** SC-AC link state from the listener and the most recent session. */
export function controlPlaneState(
  status: ScaciStatusUI,
  latestSession?: ScaciSessionUI,
): ControlPlaneState {
  if (!status.serviceOnline) return CONTROL_PLANE_STATE.LISTENER_OFFLINE;
  return latestSession && LIVE_SESSION_STATES.has(latestSession.status)
    ? CONTROL_PLANE_STATE.CONNECTED
    : CONTROL_PLANE_STATE.DISCONNECTED;
}
