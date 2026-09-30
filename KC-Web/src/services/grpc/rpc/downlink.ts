/**
 * Downlink RPCs (CoreService): queue, edit, revoke and results.
 */

import type {
  ListResult,
  RevokeDownlinkResponse,
  SCACIDownlinkQueueDTO,
  SendDownlinkRequest,
  SendDownlinkResponse,
  UpdatePendingDownlinkRequest,
} from "@api-types/api";

import * as corePb from "@services/grpc/core_pb";
import { bytesToHex, hexToBytes } from "@utils/formatters";

import { dateToTimestamp, optionalInt64String } from "../codec";
import { grpcTransport } from "../transport";

/** The dlDataQue setters shared by queueing and editing a downlink. */
interface DownlinkContentSetters {
  setPayloadsList(value: Array<Uint8Array | string>): void;
  setPriority(value: number): void;
  setCntDepend(value: boolean): void;
  setPacketCntList(value: Array<number>): void;
  setFormat(value: number): void;
  setResponseExp(value: boolean): void;
  setResponsePrio(value: boolean): void;
  setDlWindReq(value: boolean): void;
  setExpOnly(value: boolean): void;
  setDlRxStatQry(value: boolean): void;
}

function applyDownlinkContent(
  request: DownlinkContentSetters,
  data: SendDownlinkRequest,
): void {
  // An empty payload is the ACK-only downlink, which carries no userData entry.
  request.setPayloadsList(
    data.payloads.filter((p) => p.length > 0).map((p) => hexToBytes(p)),
  );
  request.setPriority(data.priority);
  request.setCntDepend(data.cntDepend);
  request.setPacketCntList(data.packetCnt ?? []);
  request.setFormat(data.format);
  request.setResponseExp(data.responseExp);
  request.setResponsePrio(data.responsePrio);
  request.setDlWindReq(data.dlWindReq);
  request.setExpOnly(data.expOnly);
  request.setDlRxStatQry(data.dlRxStatQry);
}

export async function sendDownlink(
  data: SendDownlinkRequest,
): Promise<SendDownlinkResponse> {
  const request = new corePb.SendDownlinkRequest();
  request.setEpeui(data.epEui);
  applyDownlinkContent(request, data);

  const response = await grpcTransport.callCore<
    corePb.SendDownlinkRequest,
    corePb.SendDownlinkResponse
  >((c) => c.sendDownlink, request);

  return {
    id: response.getId(),
    status: response.getStatus(),
  };
}

export async function updatePendingDownlink(
  data: UpdatePendingDownlinkRequest,
): Promise<SCACIDownlinkQueueDTO> {
  const request = new corePb.UpdatePendingDownlinkRequest();
  request.setEpeui(data.epEui);
  request.setQueId(data.queId);
  applyDownlinkContent(request, data);

  const response = await grpcTransport.callCore<
    corePb.UpdatePendingDownlinkRequest,
    corePb.DownlinkMessage
  >((c) => c.updatePendingDownlink, request);

  return mapDownlinkMessage(response);
}

export async function revokeDownlink(data: {
  epEui: string;
  queueId: string;
}): Promise<RevokeDownlinkResponse> {
  const request = new corePb.RevokeDownlinkRequest();
  request.setEpeui(data.epEui);
  request.setQueueId(data.queueId);

  const response = await grpcTransport.callCore<
    corePb.RevokeDownlinkRequest,
    corePb.RevokeDownlinkResponse
  >((c) => c.revokeDownlink, request);

  return {
    status: response.getStatus(),
    message: response.getMessage(),
  };
}

export async function listDownlinkQueue(params: {
  epEui?: string;
  bsEui?: string;
  status?: string;
  priority?: number;
  queId?: string;
  pageSize: number;
  pageToken?: string;
}): Promise<ListResult<SCACIDownlinkQueueDTO>> {
  const request = new corePb.ListDownlinkQueueRequest();
  if (params.epEui) request.setEpeui(params.epEui);
  if (params.bsEui) request.setBsEui(params.bsEui);
  if (params.status) request.setStatus(params.status);
  if (params.priority !== undefined) request.setPriority(params.priority);
  if (params.queId) request.setQueId(params.queId);
  request.setPageSize(params.pageSize);
  if (params.pageToken) request.setPageToken(params.pageToken);

  const response = await grpcTransport.callCore<
    corePb.ListDownlinkQueueRequest,
    corePb.ListDownlinkQueueResponse
  >((c) => c.listDownlinkQueue, request);

  return {
    items: response.getMessagesList().map(mapDownlinkMessage),
    nextPageToken: response.getNextPageToken() || undefined,
    totalCount: response.getTotalCount(),
  };
}

export async function getDownlinkResults(params: {
  epEui?: string;
  bsEui?: string;
  result?: string;
  queId?: string;
  timeFrom?: Date;
  pageSize: number;
  pageToken?: string;
}): Promise<ListResult<SCACIDownlinkQueueDTO>> {
  const request = new corePb.GetDownlinkResultsRequest();
  if (params.epEui) request.setEpeui(params.epEui);
  if (params.bsEui) request.setBsEui(params.bsEui);
  if (params.result) request.setStatusFilter(params.result);
  if (params.queId) request.setQueId(params.queId);
  if (params.timeFrom) request.setTimeFrom(dateToTimestamp(params.timeFrom));
  request.setPageSize(params.pageSize);
  if (params.pageToken) request.setPageToken(params.pageToken);

  const response = await grpcTransport.callCore<
    corePb.GetDownlinkResultsRequest,
    corePb.GetDownlinkResultsResponse
  >((c) => c.getDownlinkResults, request);

  return {
    items: response.getResultsList().map(mapDownlinkMessage),
    nextPageToken: response.getNextPageToken() || undefined,
    totalCount: response.getTotalCount(),
  };
}

// A counter-dependent downlink lists one userData entry per packet counter; any
// other downlink carries its single userData entry in payload.
const downlinkPayloads = (m: corePb.DownlinkMessage): Uint8Array[] => {
  const perCounter = m.getPayloadsList_asU8();
  if (perCounter.length > 0) return perCounter;
  const single = m.getPayload_asU8();
  return single.length > 0 ? [single] : [];
};

export function mapDownlinkMessage(
  m: corePb.DownlinkMessage,
): SCACIDownlinkQueueDTO {
  return {
    id: m.getId(),
    queId: m.getQueId(),
    epEui: m.getEpeui(),
    payloads: downlinkPayloads(m).map((p) => bytesToHex(p)),
    priority: m.getPriority(),
    status: m.getStatus(),
    cntDepend: m.getCntDepend(),
    packetCnt: m.getPacketCntList(),
    format: m.getFormat(),
    responseExp: m.getResponseExp(),
    responsePrio: m.getResponsePrio(),
    dlWindReq: m.getDlWindReq(),
    expOnly: m.getExpOnly(),
    dlRxStatQry: m.getDlRxStatQry(),
    result: m.getResult() || undefined,
    txTime: optionalInt64String(m.getTxTime()),
    bsEui: m.getBsEui() || undefined,
    createdAt: m.getCreatedAt()?.toDate().toISOString() ?? "",
    scheduledAt: m.getScheduledAt()?.toDate().toISOString(),
    transmittedAt: m.getTransmittedAt()?.toDate().toISOString(),
    transmissionPacketCnt: m.getTransmissionPacketCnt(),
    endpointAckedAt: m.getEndpointAckedAt()?.toDate().toISOString(),
    acceptedAt: m.getAcceptedAt()?.toDate().toISOString(),
  };
}
