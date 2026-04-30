/**
 * Endpoint Mappers
 *
 * Transforms endpoint data from API format to UI format.
 * Aligns with KC-DB/storage/mioty/types.go field names.
 * Uses attachStatus for endpoint state and epEui as primary identifier.
 */

import type { EndpointAPI, EndpointUI } from '@api-types/api';

import { ENDPOINT_ACTIVITY_WINDOW_HOURS } from '@constants/app';

/**
 * Calculate endpoint activity status based on lastSeen timestamp.
 * Fallback for when backend status is missing or unexpected.
 */
export function deriveActivityStatus(lastSeen: string | undefined): 'active' | 'inactive' {
  if (!lastSeen) return 'inactive';

  const lastSeenDate = new Date(lastSeen);
  if (isNaN(lastSeenDate.getTime())) return 'inactive';

  const now = new Date();
  const hoursSinceLastSeen = (now.getTime() - lastSeenDate.getTime()) / (1000 * 60 * 60);
  return hoursSinceLastSeen <= ENDPOINT_ACTIVITY_WINDOW_HOURS ? 'active' : 'inactive';
}

/**
 * Derive attach state from endpoint properties
 * Used for filtering and display purposes when attachStatus is not available from backend
 */
export function deriveAttachState(
  endpoint: EndpointAPI
): 'attached' | 'detached' | 'attaching' | 'pending' | 'unknown' {
  // If attachStatus is provided, use it directly
  if (endpoint.attachStatus) return endpoint.attachStatus;

  // If shAddr is assigned, endpoint is likely attached or pending
  if (endpoint.shAddr !== undefined && endpoint.shAddr > 0) return 'pending';

  // Default to detached
  return 'detached';
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
      api.status === 'active' || api.status === 'inactive'
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
    // MIOTY configuration fields per BSSCI v1.0.0 §3.8.1
    shAddr: api.shAddr,
    bidi: api.bidi,
    preAttach: api.preAttach,
    carrierOffset: api.carrierOffset,
    nwkSnKey: api.nwkSnKey,
    appKey: api.appKey,
    dualChan: api.dualChan,
    repetition: api.repetition,
    wideCarrOff: api.wideCarrOff,
    longBlkDist: api.longBlkDist,
    typeEui: api.typeEui,
    deviceModelId: api.deviceModelId,
  };
}
