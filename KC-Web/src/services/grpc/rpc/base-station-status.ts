/**
 * On-demand base station status request (CoreService RequestBaseStationStatus,
 * BSSCI §3.5): the service center sends the station a status request.
 */

import type { BaseStationStatusRequestResult } from "@api-types/api";

import * as corePb from "@services/grpc/core_pb";

import { optionalInt64String } from "../codec";
import { grpcTransport } from "../transport";

export async function requestBaseStationStatus(
  bsEui: string,
): Promise<BaseStationStatusRequestResult> {
  const request = new corePb.BaseStationStatusRequest();
  request.setBsEuiHex(bsEui);

  const response = await grpcTransport.callCore<
    corePb.BaseStationStatusRequest,
    corePb.BaseStationStatusResponse
  >((c) => c.requestBaseStationStatus, request);

  return {
    success: response.getSuccess(),
    message: response.getMessage(),
    opId: optionalInt64String(response.getOpId()),
  };
}
