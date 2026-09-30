/**
 * Event feed and failure grouping RPCs (CoreService).
 */

import type { ErrorGroupUI, EventRecordAPI, ListResult } from "@api-types/api";

import * as corePb from "@services/grpc/core_pb";
import { decodeExactJson } from "@utils/exact-json";

import { dateToTimestamp } from "../codec";
import { grpcTransport } from "../transport";

/** A system event as ListEvents and the activity feeds carry it. */
export function mapEventRecord(e: corePb.Event): EventRecordAPI {
  return {
    id: e.getId(),
    eventType: e.getEventType(),
    category: e.getCategory(),
    severity: e.getSeverity(),
    title: e.getTitle(),
    description: e.getDescription(),
    sourceName: e.getSourceName(),
    userId: e.getUserId(),
    userEmail: e.getUserEmail(),
    timestamp: e.getTimestamp()?.toDate() || new Date(),
    data:
      e.getData_asU8().length > 0
        ? decodeExactJson(e.getData_asU8())
        : undefined,
  };
}

/**
 * List system events
 */
export async function listEvents(params?: {
  pageSize?: number;
  pageToken?: string;
  categories?: readonly string[];
  eventTypes?: readonly string[];
  severity?: string;
  startTime?: Date;
  opId?: string;
  epEui?: string;
  bsEui?: string;
  outcome?: string;
  search?: string;
}): Promise<{
  events: EventRecordAPI[];
  nextPageToken?: string;
  totalCount: number;
}> {
  const request = new corePb.ListEventsRequest();
  if (params?.pageSize) request.setPageSize(params.pageSize);
  if (params?.pageToken) request.setPageToken(params.pageToken);
  if (params?.categories?.length) {
    request.setCategoriesList([...params.categories]);
  }
  if (params?.severity) request.setSeverity(params.severity);
  if (params?.eventTypes?.length) {
    request.setEventTypesList([...params.eventTypes]);
  }
  if (params?.startTime)
    request.setStartTime(dateToTimestamp(params.startTime));
  if (params?.opId) request.setOpId(params.opId);
  if (params?.epEui) request.setEpEui(params.epEui);
  if (params?.bsEui) request.setBsEui(params.bsEui);
  if (params?.outcome) request.setOutcome(params.outcome);
  if (params?.search) request.setSearch(params.search);

  const response = await grpcTransport.callCore<
    corePb.ListEventsRequest,
    corePb.ListEventsResponse
  >((c) => c.listEvents, request);

  return {
    events: response.getEventsList().map(mapEventRecord),
    nextPageToken: response.getNextPageToken() || undefined,
    totalCount: response.getTotalCount(),
  };
}

/**
 * List the failure groups of one Errors Center bucket.
 */
export async function listErrorGroups(params: {
  bucket: string;
  startTime?: Date;
  pageSize: number;
  pageToken?: string;
}): Promise<ListResult<ErrorGroupUI>> {
  const request = new corePb.ListErrorGroupsRequest();
  request.setBucket(params.bucket);
  if (params.startTime) request.setStartTime(dateToTimestamp(params.startTime));
  request.setPageSize(params.pageSize);
  if (params.pageToken) request.setPageToken(params.pageToken);

  const response = await grpcTransport.callCore<
    corePb.ListErrorGroupsRequest,
    corePb.ListErrorGroupsResponse
  >((c) => c.listErrorGroups, request);

  return {
    items: response.getGroupsList().map((g) => ({
      eventType: g.getEventType(),
      code: g.getCode(),
      message: g.getMessage(),
      sourceName: g.getSourceName(),
      firstSeen: g.getFirstSeen()?.toDate().toISOString() ?? "",
      lastSeen: g.getLastSeen()?.toDate().toISOString() ?? "",
      count: g.getCount(),
      lastOpId: g.getLastOpId() || undefined,
    })),
    nextPageToken: response.getNextPageToken() || undefined,
    totalCount: response.getTotalCount(),
  };
}
