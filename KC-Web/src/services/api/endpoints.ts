/**
 * Endpoints in the shapes the UI consumes.
 */

import type {
  ActivityFilter,
  ActivityPage,
  CreateEndpointRequest,
  EndpointAPI,
  EndpointUI,
  UpdateEndpointRequest,
} from "@api-types/api";
import { mapEndpoint } from "@mappers";
import { deriveAttachState } from "@mappers/endpoint.mapper";

import { GrpcApiError } from "@services/grpc/errors";
import * as endpointsRpc from "@services/grpc/rpc/endpoints";
import { ENDPOINT_EP_CLASS, type EndpointKeyName } from "@constants/app";

import { mapActivityPage } from "./activity";

/** Shape returned by the endpoint RPCs (the shared GrpcEndpointResult). */
export type GrpcEndpoint = Awaited<
  ReturnType<typeof endpointsRpc.listEndpoints>
>[number];

/** Maps a gRPC endpoint response to EndpointAPI and then to EndpointUI */
export function mapGrpcEndpointToUI(ep: GrpcEndpoint): EndpointUI {
  return mapEndpoint({
    epEui: ep.epEui,
    name: ep.name,
    status: ep.status as EndpointAPI["status"],
    bidi: ep.epClass === ENDPOINT_EP_CLASS.BIDIRECTIONAL,
    createdAt: ep.createdAt?.toISOString() || "",
    lastSeen: ep.lastSeenAt?.toISOString() || "",
    attachStatus:
      (ep.attachStatus as EndpointAPI["attachStatus"]) ??
      deriveAttachState({ shAddr: ep.shAddr } as EndpointAPI),
    reattachPending: ep.reattachPending,
    shAddr: ep.shAddr,
    attachCnt: ep.attachCnt,
    lastPacketCnt: ep.lastPacketCnt,
    preAttach: ep.preAttach,
    carrierOffset: ep.carrierOffset,
    dualChan: ep.dualChan,
    repetition: ep.repetition,
    wideCarrOff: ep.wideCarrOff,
    longBlkDist: ep.longBlkDist,
    nwkSnKeySet: ep.nwkSnKeySet,
    appKeySet: ep.appKeySet,
    typeEui: ep.typeEui,
    deviceModelId: ep.deviceModelId,
    lastRssi: ep.lastRssi,
    lastSnr: ep.lastSnr,
    lastEqSnr: ep.lastEqSnr,
    servingBsEui: ep.servingBsEui,
  });
}

export const endpointsApi = {
  async getEndpoints(): Promise<EndpointUI[]> {
    const response = await endpointsRpc.listEndpoints();
    return response.map(mapGrpcEndpointToUI);
  },

  /** One stored key of an endpoint in hex; the service records the reveal. */
  revealEndpointKey(
    epEui: string,
    key: EndpointKeyName,
  ): Promise<string | undefined> {
    return endpointsRpc.revealEndpointKey(epEui, key);
  },

  async getEndpointById(id: string): Promise<EndpointUI | null> {
    try {
      const ep = await endpointsRpc.getEndpoint(id);
      if (!ep) return null;

      return mapGrpcEndpointToUI(ep);
    } catch (error) {
      if (error instanceof GrpcApiError && error.isNotFound()) {
        return null;
      }
      throw error;
    }
  },

  async createEndpoint(data: CreateEndpointRequest): Promise<EndpointUI> {
    const response = await endpointsRpc.createEndpoint({
      epEui: data.epEui,
      name: data.name,
      epClass: data.bidi
        ? ENDPOINT_EP_CLASS.BIDIRECTIONAL
        : ENDPOINT_EP_CLASS.UNIDIRECTIONAL,
      nwkSnKey: data.nwkSnKey,
      appSnKey: data.appKey,
      shAddr: data.shAddr,
      dualChan: data.dualChan,
      repetition: data.repetition,
      wideCarrOff: data.wideCarrOff,
      longBlkDist: data.longBlkDist,
      attachCnt: data.attachCnt,
      preAttach: data.preAttach,
      lastPacketCnt: data.lastPacketCnt,
      carrierOffset: data.carrierOffset,
      typeEui: data.typeEui,
      deviceModelId: data.deviceModelId,
    });

    return mapGrpcEndpointToUI(response);
  },

  async updateEndpoint(
    epEui: string,
    data: UpdateEndpointRequest,
  ): Promise<EndpointUI> {
    const response = await endpointsRpc.updateEndpoint(epEui, {
      name: data.name,
      epClass:
        data.bidi !== undefined
          ? data.bidi
            ? ENDPOINT_EP_CLASS.BIDIRECTIONAL
            : ENDPOINT_EP_CLASS.UNIDIRECTIONAL
          : undefined,
      shAddr: data.shAddr,
      nwkSnKey: data.nwkSnKey,
      appKey: data.appKey,
      preAttach: data.preAttach,
      dualChan: data.dualChan,
      repetition: data.repetition,
      wideCarrOff: data.wideCarrOff,
      longBlkDist: data.longBlkDist,
      lastPacketCnt: data.lastPacketCnt,
      attachCnt: data.attachCnt,
      carrierOffset: data.carrierOffset,
      typeEui: data.typeEui,
      deviceModelId: data.deviceModelId,
      newEpEui: data.newEpEui,
    });

    return mapGrpcEndpointToUI(response);
  },

  async deleteEndpoint(id: string): Promise<void> {
    await endpointsRpc.deleteEndpoint(id);
  },

  async attachEndpoint(
    id: string,
  ): Promise<{ status: string; message?: string }> {
    const response = await endpointsRpc.attachEndpoint(id);
    return {
      status: response.status,
      message: undefined,
    };
  },

  async detachEndpoint(
    id: string,
  ): Promise<{ status: string; message?: string }> {
    const response = await endpointsRpc.detachEndpoint(id);
    return {
      status: response.status,
      message: undefined,
    };
  },

  /**
   * Get unified endpoint activity (events + messages)
   */
  async getEndpointActivity(
    epEui: string,
    filter: ActivityFilter,
    pageToken: string,
    pageSize: number,
  ): Promise<ActivityPage> {
    const response = await endpointsRpc.listEndpointActivity(epEui, {
      pageSize,
      pageToken,
      startTime: filter.startTime ? new Date(filter.startTime) : undefined,
      endTime: filter.endTime ? new Date(filter.endTime) : undefined,
    });

    return mapActivityPage(response);
  },
};
