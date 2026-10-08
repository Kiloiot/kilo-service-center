/**
 * Uplink traffic RPCs (CoreService): the tenant-wide and the per-station
 * ulData listings and the per-device uplink totals.
 */

import type { ListResult, TrafficSummaryUI, UplinkAPI } from "@api-types/api";
import type * as timestampPb from "google-protobuf/google/protobuf/timestamp_pb";

import * as corePb from "@services/grpc/core_pb";

import { dateToTimestamp } from "../codec";
import { grpcTransport } from "../transport";
import { mapStationUplinkMessage, mapUplinkMessage } from "./uplinks";

/** The ulData listing predicates both listing requests share. */
export interface UplinkListingParams {
  epEui?: string;
  bsEui?: string;
  duplicate?: boolean;
  dlOpen?: boolean;
  profile?: string;
  mode?: string;
  startTime?: Date;
  pageSize: number;
  pageToken?: string;
}

/** The setters ListMessagesRequest and ListBaseStationMessagesRequest share. */
interface UplinkListingSetters {
  setEpEui(value: string): void;
  setBsEui(value: string): void;
  setDuplicate(value: boolean): void;
  setDlOpen(value: boolean): void;
  setProfile(value: string): void;
  setMode(value: string): void;
  setStartTime(value?: timestampPb.Timestamp): void;
  setPageSize(value: number): void;
  setPageToken(value: string): void;
}

function applyUplinkListing(
  request: UplinkListingSetters,
  params: UplinkListingParams,
): void {
  if (params.epEui) request.setEpEui(params.epEui);
  if (params.bsEui) request.setBsEui(params.bsEui);
  if (params.duplicate !== undefined) request.setDuplicate(params.duplicate);
  if (params.dlOpen !== undefined) request.setDlOpen(params.dlOpen);
  if (params.profile) request.setProfile(params.profile);
  if (params.mode) request.setMode(params.mode);
  if (params.startTime) request.setStartTime(dateToTimestamp(params.startTime));
  request.setPageSize(params.pageSize);
  if (params.pageToken) request.setPageToken(params.pageToken);
}

export async function listMessages(
  params: UplinkListingParams,
): Promise<ListResult<UplinkAPI>> {
  const request = new corePb.ListMessagesRequest();
  applyUplinkListing(request, params);

  const response = await grpcTransport.callCore<
    corePb.ListMessagesRequest,
    corePb.ListMessagesResponse
  >((c) => c.listMessages, request);

  return {
    items: response.getMessagesList().map(mapUplinkMessage),
    nextPageToken: response.getNextPageToken() || undefined,
    totalCount: response.getTotalCount(),
  };
}

/** A station's uplinks; the service center reports the radio values of that station. */
export async function listBaseStationMessages(
  params: UplinkListingParams,
): Promise<ListResult<UplinkAPI>> {
  const request = new corePb.ListBaseStationMessagesRequest();
  applyUplinkListing(request, params);

  const response = await grpcTransport.callCore<
    corePb.ListBaseStationMessagesRequest,
    corePb.ListBaseStationMessagesResponse
  >((c) => c.listBaseStationMessages, request);

  return {
    items: response.getMessagesList().map(mapStationUplinkMessage),
    nextPageToken: response.getNextPageToken() || undefined,
    totalCount: response.getTotalCount(),
  };
}

export async function getBaseStationMessageStats(
  bsEui: string,
): Promise<TrafficSummaryUI> {
  const request = new corePb.GetBaseStationMessageStatsRequest();
  request.setBsEui(bsEui);

  const response = await grpcTransport.callCore<
    corePb.GetBaseStationMessageStatsRequest,
    corePb.GetBaseStationMessageStatsResponse
  >((c) => c.getBaseStationMessageStats, request);

  const stats = response.getStats();
  return {
    totalMessages: stats?.getTotalMessages() ?? 0,
    avgRssi: stats?.getAvgRssi() ?? 0,
    avgSnr: stats?.getAvgSnr() ?? 0,
    firstSeen: stats?.getFirstMessageAt()?.toDate().toISOString(),
    lastSeen: stats?.getLastMessageAt()?.toDate().toISOString(),
    uniqueEndpoints: stats?.getUniqueEndpoints() ?? 0,
    messagesToday: stats?.getMessagesToday() ?? 0,
    messagesThisWeek: stats?.getMessagesThisWeek() ?? 0,
    messagesThisMonth: stats?.getMessagesThisMonth() ?? 0,
  };
}

export async function getEndPointStats(
  epEui: string,
): Promise<TrafficSummaryUI> {
  const request = new corePb.GetEndPointStatsRequest();
  request.setEpEui(epEui);

  const response = await grpcTransport.callCore<
    corePb.GetEndPointStatsRequest,
    corePb.GetEndPointStatsResponse
  >((c) => c.getEndPointStats, request);

  return {
    totalMessages: response.getTotalCount(),
    avgRssi: response.getAvgRssi(),
    avgSnr: response.getAvgSnr(),
    firstSeen: response.getFirstSeen()?.toDate().toISOString(),
    lastSeen: response.getLastSeen()?.toDate().toISOString(),
    activeDays: response.getActiveDays(),
  };
}
