/**
 * DL RX status RPCs (CoreService, BSSCI §3.15): the statuses an end point
 * reported, the queries sent for them, and a new query.
 */

import type {
  DlRxStatusQueriesResponse,
  DlRxStatusResponse,
  QueryDlRxStatusResponse,
} from "@api-types/api";

import * as corePb from "@services/grpc/core_pb";

import { optionalInt64String } from "../codec";
import { grpcTransport } from "../transport";

export async function getDlRxStatus(data: {
  epEui: string;
  limit: number;
}): Promise<DlRxStatusResponse> {
  const request = new corePb.GetDLRXStatusRequest();
  request.setEpEui(data.epEui);
  request.setLimit(data.limit);

  const response = await grpcTransport.callCore<
    corePb.GetDLRXStatusRequest,
    corePb.GetDLRXStatusResponse
  >((c) => c.getDLRXStatus, request);

  return {
    statuses: response.getStatusesList().map((s) => ({
      bsEui: s.getBsEui(),
      rxTime: optionalInt64String(s.getRxTime()),
      packetCnt: s.getPacketCnt(),
      dlRxSnr: s.getDlRxSnr(),
      dlRxRssi: s.getDlRxRssi(),
    })),
    totalCount: response.getTotalCount(),
  };
}

export async function getDlRxStatusQueries(data: {
  epEui: string;
  limit: number;
}): Promise<DlRxStatusQueriesResponse> {
  const request = new corePb.GetDLRXStatusQueriesRequest();
  request.setEpEui(data.epEui);
  request.setLimit(data.limit);

  const response = await grpcTransport.callCore<
    corePb.GetDLRXStatusQueriesRequest,
    corePb.GetDLRXStatusQueriesResponse
  >((c) => c.getDLRXStatusQueries, request);

  return {
    queries: response.getQueriesList().map((q) => ({
      bsEui: q.getBsEui(),
      opId: optionalInt64String(q.getOpId()),
      status: q.getStatus(),
      requestedAt: q.getRequestedAt()?.toDate().toISOString(),
    })),
    pendingCount: response.getStats()?.getPending() ?? 0,
  };
}

export async function queryDlRxStatus(
  epEui: string,
): Promise<QueryDlRxStatusResponse> {
  const request = new corePb.QueryDLRXStatusRequest();
  request.setEpEui(epEui);

  const response = await grpcTransport.callCore<
    corePb.QueryDLRXStatusRequest,
    corePb.QueryDLRXStatusResponse
  >((c) => c.queryDLRXStatus, request);

  return {
    queryInitiated: response.getQueryInitiated(),
    message: response.getMessage(),
  };
}
