/**
 * Endpoint Mappers
 *
 * Transforms endpoint data from API format to UI format.
 * Aligns with KC-DB/storage/mioty/types.go field names.
 * Uses attachStatus for endpoint state and epEui as primary identifier.
 */

import type { EndpointAPI, EndpointUI } from "@api-types/api";

import {
  ENDPOINT_ACTIVITY,
  ENDPOINT_ACTIVITY_WINDOW_HOURS,
  ENDPOINT_ATTACH_STATUS,
  type EndpointActivity,
  type EndpointAttachStatus,
  MS_PER_HOUR,
} from "@constants/app";

/**
 * Calculate endpoint activity status based on lastSeen timestamp.
 * Fallback for when backend status is missing or unexpected.
 */
export function deriveActivityStatus(
  lastSeen: string | undefined,
): EndpointActivity {
  if (!lastSeen) return ENDPOINT_ACTIVITY.INACTIVE;

  const lastSeenDate = new Date(lastSeen);
  if (isNaN(lastSeenDate.getTime())) return ENDPOINT_ACTIVITY.INACTIVE;

  const now = new Date();
  const hoursSinceLastSeen =
    (now.getTime() - lastSeenDate.getTime()) / MS_PER_HOUR;
  return hoursSinceLastSeen <= ENDPOINT_ACTIVITY_WINDOW_HOURS
    ? ENDPOINT_ACTIVITY.ACTIVE
    : ENDPOINT_ACTIVITY.INACTIVE;
}

/**
 * Derive attach state from endpoint properties
 * Used for filtering and display purposes when attachStatus is not available from backend
 */
export function deriveAttachState(endpoint: EndpointAPI): EndpointAttachStatus {
  // If attachStatus is provided, use it directly
  if (endpoint.attachStatus) return endpoint.attachStatus;

  // If shAddr is assigned, endpoint is likely attached or pending
  if (endpoint.shAddr !== undefined && endpoint.shAddr > 0)
    return ENDPOINT_ATTACH_STATUS.PENDING;

  return ENDPOINT_ATTACH_STATUS.DETACHED;
}

/**
 * Transform a single endpoint from API to UI format
 * Uses epEui as primary identifier
 */
export function mapEndpoint(api: EndpointAPI): EndpointUI {
  return {
    id: api.epEui, // Backward compat alias for epEui (used as key in lists/grids)
    epEui: api.epEui,
    name: api.name,
    status:
      api.status === ENDPOINT_ACTIVITY.ACTIVE ||
      api.status === ENDPOINT_ACTIVITY.INACTIVE
        ? api.status
        : deriveActivityStatus(api.lastSeen),
    batteryLevel: api.batteryLevel,
    lastSeen: api.lastSeen,
    createdAt: api.createdAt,
    // SCACI §3.6.1 fields
    attachCnt: api.attachCnt,
    lastPacketCnt: api.lastPacketCnt,
    // Attach status - use API value with fallback to derived state
    attachStatus: api.attachStatus ?? deriveAttachState(api),
    reattachPending: api.reattachPending,
    // MIOTY configuration fields per BSSCI v1.0.0 §3.8.1
    shAddr: api.shAddr,
    bidi: api.bidi,
    preAttach: api.preAttach,
    carrierOffset: api.carrierOffset,
    nwkSnKeySet: api.nwkSnKeySet,
    appKeySet: api.appKeySet,
    dualChan: api.dualChan,
    repetition: api.repetition,
    wideCarrOff: api.wideCarrOff,
    longBlkDist: api.longBlkDist,
    typeEui: api.typeEui,
    deviceModelId: api.deviceModelId,
    lastRssi: api.lastRssi,
    lastSnr: api.lastSnr,
    lastEqSnr: api.lastEqSnr,
    servingBsEui: api.servingBsEui,
  };
}
