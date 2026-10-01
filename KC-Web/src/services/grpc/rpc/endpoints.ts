/**
 * Endpoint RPCs (CoreService): inventory, attach/detach and activity feed.
 */

import { Empty } from "google-protobuf/google/protobuf/empty_pb";
import { FieldMask } from "google-protobuf/google/protobuf/field_mask_pb";

import * as corePb from "@services/grpc/core_pb";
import { bytesToHex, hexToBytes } from "@utils/formatters";
import {
  ENDPOINT_KEY,
  ENDPOINT_UPDATE_MASK_PATHS,
  type EndpointKeyName,
} from "@constants/app";

import { applyPaginationAndTimeRange, tagsMapToRecord } from "../codec";
import { GrpcApiError } from "../errors";
import { grpcTransport } from "../transport";
import { type ActivityPageDTO, mapActivityResponse } from "./activity";
import { mapUplinkMessage } from "./uplinks";

/**
 * Shared return type for all gRPC endpoint methods.
 * Single source of truth — prevents field drift across list/get/create/update.
 */
export interface GrpcEndpointResult {
  epEui: string;
  tenantId: string;
  name: string;
  description?: string;
  epClass: string;
  status: string;
  attachStatus?: string;
  reattachPending: boolean;
  shAddr?: number;
  attachCnt?: number;
  lastPacketCnt?: number;
  preAttach?: boolean;
  carrierOffset?: number;
  dualChan?: boolean;
  repetition?: boolean;
  wideCarrOff?: boolean;
  longBlkDist?: boolean;
  deviceModelId?: string;
  typeEui?: string;
  nwkSnKeySet: boolean;
  appKeySet: boolean;
  tags: Record<string, string>;
  createdAt?: Date;
  updatedAt?: Date;
  lastSeenAt?: Date;
  lastRssi?: number;
  lastSnr?: number;
  lastEqSnr?: number;
  servingBsEui?: string;
}

/**
 * Maps a protobuf EndPoint message to the shared GrpcEndpointResult shape.
 */
export function mapEndpointProto(ep: corePb.EndPoint): GrpcEndpointResult {
  return {
    epEui: ep.getEpeui(),
    tenantId: String(ep.getTenantId()),
    name: ep.getName(),
    description: ep.getDescription() || undefined,
    epClass: ep.getEpClass(),
    status: ep.getStatus(),
    attachStatus: ep.getAttachStatus() || undefined,
    reattachPending: ep.getReattachPending(),
    shAddr: ep.getShAddr(),
    attachCnt: ep.getAttachCnt(),
    lastPacketCnt: ep.getLastPacketCnt(),
    preAttach: ep.getPreAttach(),
    carrierOffset: ep.getCarrierOffset(),
    dualChan: ep.getDualChan(),
    repetition: ep.getRepetition(),
    wideCarrOff: ep.getWideCarrOff(),
    longBlkDist: ep.getLongBlkDist(),
    deviceModelId: ep.getDeviceModelId() || undefined,
    typeEui: (() => {
      const t = ep.getTypeEui_asU8();
      return t.length > 0 ? bytesToHex(t) : undefined;
    })(),
    nwkSnKeySet: ep.getNwkSnKeySet(),
    appKeySet: ep.getAppKeySet(),
    tags: tagsMapToRecord(ep.getTagsMap()),
    createdAt: ep.getCreatedAt()?.toDate(),
    updatedAt: ep.getUpdatedAt()?.toDate(),
    lastSeenAt: ep.getLastSeenAt()?.toDate(),
    lastRssi: ep.getLastRssi()?.getValue(),
    lastSnr: ep.getLastSnr()?.getValue(),
    lastEqSnr: ep.getLastEqSnr()?.getValue(),
    servingBsEui: ep.getServingBsEui() || undefined,
  };
}

/** Endpoint create payload (drives buildCreateEndpointRequest). */
export interface CreateEndpointInput {
  epEui: string;
  name?: string;
  description?: string;
  epClass?: string;
  nwkSnKey?: string;
  appSnKey?: string;
  shAddr?: number;
  dualChan?: boolean;
  repetition?: boolean;
  wideCarrOff?: boolean;
  longBlkDist?: boolean;
  attachCnt?: number;
  preAttach?: boolean;
  lastPacketCnt?: number;
  typeEui?: string;
  carrierOffset?: number;
  deviceModelId?: string;
}

/** Endpoint update payload (drives applyEndpointUpdateFields). */
export interface UpdateEndpointInput {
  name?: string;
  description?: string;
  epClass?: string;
  status?: string;
  shAddr?: number;
  nwkSnKey?: string;
  appKey?: string | null;
  preAttach?: boolean;
  dualChan?: boolean;
  repetition?: boolean;
  wideCarrOff?: boolean;
  longBlkDist?: boolean;
  attachCnt?: number;
  lastPacketCnt?: number;
  carrierOffset?: number;
  typeEui?: string | null;
  deviceModelId?: string | null;
  newEpEui?: string;
}

/** Builds the nested EndPoint + outer request for CreateEndPoint. */
export function buildCreateEndpointRequest(
  data: CreateEndpointInput,
): corePb.CreateEndPointRequest {
  const endpoint = new corePb.EndPoint();
  endpoint.setEpeui(data.epEui);
  if (data.name) endpoint.setName(data.name);
  if (data.description) endpoint.setDescription(data.description);
  if (data.epClass) endpoint.setEpClass(data.epClass);
  if (data.nwkSnKey) endpoint.setNwkSnKey(hexToBytes(data.nwkSnKey));
  if (data.appSnKey) endpoint.setAppKey(hexToBytes(data.appSnKey));
  if (data.shAddr !== undefined) endpoint.setShAddr(data.shAddr);
  if (data.dualChan !== undefined) endpoint.setDualChan(data.dualChan);
  if (data.repetition !== undefined) endpoint.setRepetition(data.repetition);
  if (data.wideCarrOff !== undefined) endpoint.setWideCarrOff(data.wideCarrOff);
  if (data.longBlkDist !== undefined) endpoint.setLongBlkDist(data.longBlkDist);
  if (data.attachCnt !== undefined) endpoint.setAttachCnt(data.attachCnt);
  if (data.preAttach !== undefined) endpoint.setPreAttach(data.preAttach);
  if (data.lastPacketCnt !== undefined)
    endpoint.setLastPacketCnt(data.lastPacketCnt);
  if (data.typeEui) endpoint.setTypeEui(hexToBytes(data.typeEui));
  if (data.carrierOffset !== undefined)
    endpoint.setCarrierOffset(data.carrierOffset);
  if (data.deviceModelId) endpoint.setDeviceModelId(data.deviceModelId);

  const request = new corePb.CreateEndPointRequest();
  request.setEndpoint(endpoint);
  return request;
}

/**
 * Sets each declared field on the nested EndPoint message and returns the
 * FieldMask paths corresponding to the fields that were set. appKey, typeEui
 * and deviceModelId support an explicit clear ({null} clears, missing leaves
 * unchanged).
 */
export function applyEndpointUpdateFields(
  endpoint: corePb.EndPoint,
  data: UpdateEndpointInput,
): string[] {
  const paths: string[] = [];
  if (data.name !== undefined) {
    endpoint.setName(data.name);
    paths.push(ENDPOINT_UPDATE_MASK_PATHS.NAME);
  }
  if (data.description !== undefined) {
    endpoint.setDescription(data.description);
    paths.push(ENDPOINT_UPDATE_MASK_PATHS.DESCRIPTION);
  }
  if (data.epClass !== undefined) {
    endpoint.setEpClass(data.epClass);
    paths.push(ENDPOINT_UPDATE_MASK_PATHS.EP_CLASS);
  }
  if (data.status !== undefined) {
    endpoint.setStatus(data.status);
    paths.push(ENDPOINT_UPDATE_MASK_PATHS.STATUS);
  }
  if (data.shAddr !== undefined) {
    endpoint.setShAddr(data.shAddr);
    paths.push(ENDPOINT_UPDATE_MASK_PATHS.SH_ADDR);
  }
  if (data.nwkSnKey !== undefined) {
    endpoint.setNwkSnKey(hexToBytes(data.nwkSnKey));
    paths.push(ENDPOINT_UPDATE_MASK_PATHS.NWK_SN_KEY);
  }
  // appKey: null removes the stored key (empty bytes), string sets, missing leaves
  if (data.appKey !== undefined) {
    if (data.appKey) {
      endpoint.setAppKey(hexToBytes(data.appKey));
    }
    paths.push(ENDPOINT_UPDATE_MASK_PATHS.APP_KEY);
  }
  if (data.attachCnt !== undefined) {
    endpoint.setAttachCnt(data.attachCnt);
    paths.push(ENDPOINT_UPDATE_MASK_PATHS.ATTACH_CNT);
  }
  if (data.lastPacketCnt !== undefined) {
    endpoint.setLastPacketCnt(data.lastPacketCnt);
    paths.push(ENDPOINT_UPDATE_MASK_PATHS.LAST_PACKET_CNT);
  }
  if (data.carrierOffset !== undefined) {
    endpoint.setCarrierOffset(data.carrierOffset);
    paths.push(ENDPOINT_UPDATE_MASK_PATHS.CARRIER_OFFSET);
  }
  if (data.preAttach !== undefined) {
    endpoint.setPreAttach(data.preAttach);
    paths.push(ENDPOINT_UPDATE_MASK_PATHS.PRE_ATTACH);
  }
  if (data.dualChan !== undefined) {
    endpoint.setDualChan(data.dualChan);
    paths.push(ENDPOINT_UPDATE_MASK_PATHS.DUAL_CHAN);
  }
  if (data.repetition !== undefined) {
    endpoint.setRepetition(data.repetition);
    paths.push(ENDPOINT_UPDATE_MASK_PATHS.REPETITION);
  }
  if (data.wideCarrOff !== undefined) {
    endpoint.setWideCarrOff(data.wideCarrOff);
    paths.push(ENDPOINT_UPDATE_MASK_PATHS.WIDE_CARR_OFF);
  }
  if (data.longBlkDist !== undefined) {
    endpoint.setLongBlkDist(data.longBlkDist);
    paths.push(ENDPOINT_UPDATE_MASK_PATHS.LONG_BLK_DIST);
  }
  // typeEui: null clears (empty bytes), string sets, missing leaves
  if (data.typeEui !== undefined) {
    if (data.typeEui) {
      endpoint.setTypeEui(hexToBytes(data.typeEui));
    }
    paths.push(ENDPOINT_UPDATE_MASK_PATHS.TYPE_EUI);
  }
  // deviceModelId: null clears (empty string), string sets, missing leaves
  if (data.deviceModelId !== undefined) {
    endpoint.setDeviceModelId(data.deviceModelId ?? "");
    paths.push(ENDPOINT_UPDATE_MASK_PATHS.DEVICE_MODEL_ID);
  }
  return paths;
}

/** Wraps a non-empty paths array in a FieldMask, or returns undefined. */
export function buildEndpointUpdateMask(
  paths: string[],
): FieldMask | undefined {
  if (paths.length === 0) return undefined;
  const mask = new FieldMask();
  mask.setPathsList(paths);
  return mask;
}

/** Builds the UpdateEndPointRequest from epEui + update payload. */
export function buildUpdateEndpointRequest(
  epEui: string,
  data: UpdateEndpointInput,
): corePb.UpdateEndPointRequest {
  const endpoint = new corePb.EndPoint();
  endpoint.setEpeui(epEui);
  const paths = applyEndpointUpdateFields(endpoint, data);

  const request = new corePb.UpdateEndPointRequest();
  request.setEndpoint(endpoint);
  if (data.newEpEui !== undefined) {
    request.setNewEpEui(data.newEpEui);
  }
  const mask = buildEndpointUpdateMask(paths);
  if (mask) {
    request.setUpdateMask(mask);
  }
  return request;
}

/**
 * List all endpoints
 */
export async function listEndpoints(): Promise<GrpcEndpointResult[]> {
  const request = new corePb.ListEndPointsRequest();

  const response = await grpcTransport.callCore<
    corePb.ListEndPointsRequest,
    corePb.ListEndPointsResponse
  >((c) => c.listEndPoints, request);

  return response.getEndpointsList().map(mapEndpointProto);
}

/**
 * Get endpoint by EUI
 */
export async function getEndpoint(
  epEui: string,
): Promise<GrpcEndpointResult | null> {
  const request = new corePb.GetEndPointRequest();
  request.setEpeui(epEui);

  try {
    const response = await grpcTransport.callCore<
      corePb.GetEndPointRequest,
      corePb.EndPoint
    >((c) => c.getEndPoint, request);

    return mapEndpointProto(response);
  } catch (error) {
    if (error instanceof GrpcApiError && error.isNotFound()) {
      return null;
    }
    throw error;
  }
}

const PROTO_ENDPOINT_KEYS: Record<
  EndpointKeyName,
  corePb.EndpointKeyMap[keyof corePb.EndpointKeyMap]
> = {
  [ENDPOINT_KEY.NETWORK]: corePb.EndpointKey.ENDPOINT_KEY_NWK_SN_KEY,
  [ENDPOINT_KEY.APPLICATION]: corePb.EndpointKey.ENDPOINT_KEY_APP_KEY,
};

/**
 * Reveals one stored key of an endpoint in hex; the service records the
 * reveal. Undefined when the key is not stored.
 */
export async function revealEndpointKey(
  epEui: string,
  key: EndpointKeyName,
): Promise<string | undefined> {
  const request = new corePb.GetEndPointRequest();
  request.setEpeui(epEui);
  request.setRevealKeysList([PROTO_ENDPOINT_KEYS[key]]);

  const response = await grpcTransport.callCore<
    corePb.GetEndPointRequest,
    corePb.EndPoint
  >((c) => c.getEndPoint, request);

  const revealed =
    key === ENDPOINT_KEY.NETWORK
      ? response.getNwkSnKey_asU8()
      : response.getAppKey_asU8();
  return revealed.length > 0 ? bytesToHex(revealed) : undefined;
}

/**
 * Create endpoint
 */
export async function createEndpoint(
  data: CreateEndpointInput,
): Promise<GrpcEndpointResult> {
  const request = buildCreateEndpointRequest(data);
  const ep = await grpcTransport.callCore<
    corePb.CreateEndPointRequest,
    corePb.EndPoint
  >((c) => c.createEndPoint, request);
  return mapEndpointProto(ep);
}

/**
 * Update endpoint
 * Uses FieldMask to track which fields are being updated, enabling partial updates
 * for boolean fields without resetting unspecified values to false.
 */
export async function updateEndpoint(
  epEui: string,
  data: UpdateEndpointInput,
): Promise<GrpcEndpointResult> {
  const request = buildUpdateEndpointRequest(epEui, data);
  const ep = await grpcTransport.callCore<
    corePb.UpdateEndPointRequest,
    corePb.EndPoint
  >((c) => c.updateEndPoint, request);
  return mapEndpointProto(ep);
}

/**
 * Delete endpoint
 */
export async function deleteEndpoint(epEui: string): Promise<void> {
  const request = new corePb.DeleteEndPointRequest();
  request.setEpeui(epEui);

  await grpcTransport.callCore<corePb.DeleteEndPointRequest, Empty>(
    (c) => c.deleteEndPoint,
    request,
  );
}

/**
 * Attach endpoint to network
 */
export async function attachEndpoint(
  epEui: string,
): Promise<{ operationId: string; status: string }> {
  const request = new corePb.AttachEndPointRequest();
  request.setEpEui(epEui);

  const response = await grpcTransport.callCore<
    corePb.AttachEndPointRequest,
    corePb.AttachEndPointResponse
  >((c) => c.attachEndPoint, request);

  return {
    operationId: response.getOperationId(),
    status: response.getStatus(),
  };
}

/**
 * Detach endpoint from network
 */
export async function detachEndpoint(
  epEui: string,
): Promise<{ operationId: string; status: string }> {
  const request = new corePb.DetachEndPointRequest();
  request.setEpEui(epEui);

  const response = await grpcTransport.callCore<
    corePb.DetachEndPointRequest,
    corePb.DetachEndPointResponse
  >((c) => c.detachEndPoint, request);

  return {
    operationId: response.getOperationId(),
    status: response.getStatus(),
  };
}

/**
 * List endpoint activity (unified events and messages)
 */
export async function listEndpointActivity(
  epEui: string,
  params?: {
    pageSize?: number;
    pageToken?: string;
    startTime?: Date;
    endTime?: Date;
  },
): Promise<ActivityPageDTO> {
  const request = new corePb.ListEndpointActivityRequest();
  request.setEpEui(epEui);
  applyPaginationAndTimeRange(request, params);

  const response = await grpcTransport.callCore<
    corePb.ListEndpointActivityRequest,
    corePb.ListEndpointActivityResponse
  >((c) => c.listEndpointActivity, request);

  return mapActivityResponse(response, mapUplinkMessage);
}
