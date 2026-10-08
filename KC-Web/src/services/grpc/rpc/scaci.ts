/**
 * SCACI monitoring RPCs (CoreService): the control-plane status and the
 * Application Center sessions (SCACI §3.3, §3.4).
 */

import type { ListPage, PageRequest } from "@api-types/pagination";
import type { ScaciSessionDTO, ScaciStatusDTO } from "@api-types/system";

import * as corePb from "@services/grpc/core_pb";

import { offsetPageToken, pageOf } from "../codec";
import { grpcTransport } from "../transport";

function mapSession(session: corePb.ScaciSession): ScaciSessionDTO {
  return {
    acEui: session.getAcEui(),
    status: session.getStatus(),
    protocolVersion: session.getProtocolVersion(),
    snAcUuid: session.getSnAcUuid(),
    snScUuid: session.getSnScUuid(),
    lastOpIdAc: session.getLastOpIdAc(),
    lastOpIdSc: session.getLastOpIdSc(),
  };
}

export async function getScaciStatus(): Promise<ScaciStatusDTO | undefined> {
  const response = await grpcTransport.callCore<
    corePb.GetScaciStatusRequest,
    corePb.GetScaciStatusResponse
  >((c) => c.getScaciStatus, new corePb.GetScaciStatusRequest());
  const status = response.getStatus();
  if (!status) return undefined;
  return {
    serviceOnline: status.getServiceOnline(),
    protocolVersion: status.getProtocolVersion(),
    scEui: status.getScEui(),
    lastPingAt: status.getLastPingAt()?.toDate(),
    lastPingRttMs: status.getLastPingRttMs(),
    missedPings: status.getMissedPings(),
    lastConnectResult: status.getLastConnectResult(),
  };
}

export async function listScaciSessions(
  page: PageRequest,
): Promise<ListPage<ScaciSessionDTO>> {
  const request = new corePb.ListScaciSessionsRequest();
  request.setPageSize(page.pageSize);
  request.setPageToken(offsetPageToken(page));

  const response = await grpcTransport.callCore<
    corePb.ListScaciSessionsRequest,
    corePb.ListScaciSessionsResponse
  >((c) => c.listScaciSessions, request);
  return pageOf(
    response.getSessionsList().map(mapSession),
    response.getTotalCount(),
    response.getNextPageToken(),
  );
}
