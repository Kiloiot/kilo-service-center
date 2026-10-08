/**
 * Coordinates of every base station on the server (CoreService
 * ListAllBaseStationLocations, server administrators only).
 */

import type { BaseStationLocationDTO } from "@api-types/api";

import * as corePb from "@services/grpc/core_pb";

import { wrapperValue } from "../codec";
import { grpcTransport } from "../transport";

export async function listAllBaseStationLocations(): Promise<
  BaseStationLocationDTO[]
> {
  const response = await grpcTransport.callCore<
    corePb.ListAllBaseStationLocationsRequest,
    corePb.ListAllBaseStationLocationsResponse
  >(
    (c) => c.listAllBaseStationLocations,
    new corePb.ListAllBaseStationLocationsRequest(),
  );

  return response.getLocationsList().map((location) => ({
    bsEui: location.getBsEui(),
    name: location.getName(),
    latitude: location.getLatitude(),
    longitude: location.getLongitude(),
    altitude: wrapperValue(location.getAltitude()),
    locationSource: location.getLocationSource(),
    isOnline: location.getIsOnline(),
    orgId: location.getOrgId(),
  }));
}
