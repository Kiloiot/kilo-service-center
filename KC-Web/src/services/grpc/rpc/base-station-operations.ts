/**
 * Base station operation RPCs (CoreService): availability buckets and the
 * SC-initiated ping (BSSCI §5.4).
 */

import type { BaseStationAvailabilityDTO } from "@api-types/base-station-operations";
import type { AnalyticsWindow } from "@api-types/system";

import * as corePb from "@services/grpc/core_pb";

import { dateToTimestamp } from "../codec";
import { grpcTransport } from "../transport";

export async function getBaseStationAvailability(
  bsEui: string,
  window: AnalyticsWindow,
  intervalSeconds: number,
): Promise<BaseStationAvailabilityDTO> {
  const request = new corePb.GetBaseStationAvailabilityRequest();
  request.setBseui(bsEui);
  request.setStartTime(dateToTimestamp(window.startTime));
  request.setEndTime(dateToTimestamp(window.endTime));
  request.setIntervalSeconds(intervalSeconds);

  const response = await grpcTransport.callCore<
    corePb.GetBaseStationAvailabilityRequest,
    corePb.GetBaseStationAvailabilityResponse
  >((c) => c.getBaseStationAvailability, request);
  return {
    availability: response.getAvailabilityList(),
    intervalSeconds: response.getIntervalSeconds(),
  };
}

export async function initiatePing(bsEui: string): Promise<{
  success: boolean;
  message: string;
  opId: string;
}> {
  const request = new corePb.InitiatePingRequest();
  request.setBsEuiHex(bsEui);

  const response = await grpcTransport.callCore<
    corePb.InitiatePingRequest,
    corePb.InitiatePingResponse
  >((c) => c.initiatePing, request);
  return {
    success: response.getSuccess(),
    message: response.getMessage(),
    opId: response.getOpId(),
  };
}
