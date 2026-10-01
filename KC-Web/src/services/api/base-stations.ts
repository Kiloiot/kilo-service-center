/**
 * Base stations in the shapes the UI consumes.
 */

import type {
  ActivityFilter,
  ActivityPage,
  BaseStationAPI,
  BaseStationDetailAPI,
  BaseStationLocationUI,
  BaseStationMessagesFilter,
  BaseStationStatusRequestResult,
  BaseStationUI,
} from "@api-types/api";
import { mapBaseStation, mapBaseStationLocation } from "@mappers";

import { GrpcApiError } from "@services/grpc/errors";
import * as baseStationLocationsRpc from "@services/grpc/rpc/base-station-locations";
import * as baseStationStatusRpc from "@services/grpc/rpc/base-station-status";
import * as baseStationsRpc from "@services/grpc/rpc/base-stations";
import { filterAndSortBaseStations } from "@utils/list-query";
import type { SortDirection } from "@constants/app";
import {
  BASE_STATION_STATUS,
  BS_CONNECTION_TYPE,
  UPLINK_EXPORT_FORMAT,
  type UplinkExportFormat,
} from "@constants/app";

import { mapActivityPage } from "./activity";

export /** Builds the common filter params object for base station message queries */
function buildMessageFilterParams(
  filter: BaseStationMessagesFilter | undefined,
  pageSize: number,
): {
  pageSize: number;
  direction: "uplink" | "downlink" | undefined;
  epEui: string | undefined;
  startTime: Date | undefined;
  endTime: Date | undefined;
} {
  return {
    pageSize,
    direction: filter?.direction as "uplink" | "downlink" | undefined,
    epEui: filter?.epEui,
    startTime: filter?.startTime ? new Date(filter.startTime) : undefined,
    endTime: filter?.endTime ? new Date(filter.endTime) : undefined,
  };
}

export const baseStationsApi = {
  async getBaseStations(filters?: {
    search?: string;
    status?: string[];
    sort?: { field: string; direction: SortDirection };
  }): Promise<BaseStationUI[]> {
    const response = await baseStationsRpc.listBaseStations();

    // Map gRPC response to API format expected by mappers
    const items = response.map((bs) => {
      const apiFormat: BaseStationAPI = {
        bsEui: bs.bsEui,
        name: bs.name,
        isOnline: bs.status === BASE_STATION_STATUS.ONLINE,
        lastSeen: bs.lastSeenAt?.toISOString() || "",
        firstSeen: bs.createdAt?.toISOString() || "",
        latitude: bs.latitude,
        longitude: bs.longitude,
        altitude: bs.altitude,
        locationSource: bs.locationSource,
        locationUpdatedAt: bs.locationUpdatedAt?.toISOString(),
        certificateExpiresAt: bs.certificateExpiresAt?.toISOString(),
        tlsCertFingerprint: bs.tlsCertFingerprint,
        sessionStartedAt: bs.sessionStartedAt?.toISOString(),
      };
      return mapBaseStation(apiFormat);
    });

    return filterAndSortBaseStations(items, filters);
  },

  async getBaseStationDetails(eui: string): Promise<BaseStationUI | null> {
    try {
      const bs = await baseStationsRpc.getBaseStation(eui);
      if (!bs) return null;

      const apiFormat: BaseStationDetailAPI = {
        eui: bs.bsEui,
        bsEui: bs.bsEui,
        name: bs.name,
        isOnline: bs.status === BASE_STATION_STATUS.ONLINE,
        lastSeen: bs.lastSeenAt?.toISOString() || "",
        firstSeen: bs.createdAt?.toISOString() || "",
        connectionType: BS_CONNECTION_TYPE.BSSCI,
        createdAt: bs.createdAt?.toISOString() || "",
        updatedAt: bs.updatedAt?.toISOString() || "",
        // MIOTY status fields (BSSCI v1.0.0 §5.5.2)
        systemTime: bs.systemTime,
        dutyCycle: bs.dutyCycle,
        uptimeSeconds: bs.uptimeSeconds,
        temperatureCelsius: bs.temperatureCelsius,
        cpuLoad: bs.cpuLoad,
        memoryLoad: bs.memoryLoad,
        bsConfig: bs.bsConfig,
        lastStatusAt: bs.lastStatusAt?.toISOString(),
        serviceCenterUrl: bs.serviceCenterUrl,
        latitude: bs.latitude,
        longitude: bs.longitude,
        altitude: bs.altitude,
        locationSource: bs.locationSource,
        locationUpdatedAt: bs.locationUpdatedAt?.toISOString(),
        certificateExpiresAt: bs.certificateExpiresAt?.toISOString(),
        tlsCertFingerprint: bs.tlsCertFingerprint,
        sessionStartedAt: bs.sessionStartedAt?.toISOString(),
      };
      return mapBaseStation(apiFormat);
    } catch (error) {
      if (error instanceof GrpcApiError && error.isNotFound()) {
        return null;
      }
      throw error;
    }
  },

  async createBaseStation(data: {
    eui: string;
    name?: string;
    description?: string;
    latitude?: number;
    longitude?: number;
    altitude?: number;
  }): Promise<BaseStationUI> {
    const response = await baseStationsRpc.createBaseStation({
      bsEui: data.eui,
      name: data.name,
      description: data.description,
      latitude: data.latitude,
      longitude: data.longitude,
      altitude: data.altitude,
    });

    const apiFormat: BaseStationAPI = {
      bsEui: response.bsEui,
      name: response.name,
      isOnline: response.status === BASE_STATION_STATUS.ONLINE,
      lastSeen: "",
      firstSeen: "",
      latitude: response.latitude,
      longitude: response.longitude,
      altitude: response.altitude,
      locationSource: response.locationSource,
      locationUpdatedAt: response.locationUpdatedAt?.toISOString(),
    };
    return mapBaseStation(apiFormat);
  },

  /** Every located base station on the server (server administrators only). */
  async listLocations(): Promise<BaseStationLocationUI[]> {
    const locations =
      await baseStationLocationsRpc.listAllBaseStationLocations();
    return locations.map(mapBaseStationLocation);
  },

  async deleteBaseStation(eui: string): Promise<void> {
    await baseStationsRpc.deleteBaseStation(eui);
  },

  /** Asks a connected base station for its status now (BSSCI §3.5). */
  requestStatus(eui: string): Promise<BaseStationStatusRequestResult> {
    return baseStationStatusRpc.requestBaseStationStatus(eui);
  },

  /**
   * Update base station
   * Enables editing from detail page dialog.
   */
  async updateBaseStation(
    eui: string,
    data: {
      name?: string;
      latitude?: number | null;
      longitude?: number | null;
      altitude?: number | null;
    },
  ): Promise<BaseStationUI> {
    const response = await baseStationsRpc.updateBaseStation(eui, {
      name: data.name,
      latitude: data.latitude,
      longitude: data.longitude,
      altitude: data.altitude,
    });

    const apiFormat: BaseStationAPI = {
      bsEui: response.bsEui,
      name: response.name,
      isOnline: response.status === BASE_STATION_STATUS.ONLINE,
      lastSeen: "",
      firstSeen: "",
      latitude: response.latitude,
      longitude: response.longitude,
      altitude: response.altitude,
      locationSource: response.locationSource,
      locationUpdatedAt: response.locationUpdatedAt?.toISOString(),
    };
    return mapBaseStation(apiFormat);
  },

  /**
   * Update base station EUI with cascade to all dependent tables.
   */
  async updateBaseStationEui(
    eui: string,
    newEui: string,
  ): Promise<BaseStationUI> {
    const response = await baseStationsRpc.updateBaseStationEui(eui, newEui);

    const apiFormat: BaseStationAPI = {
      bsEui: response.bsEui,
      name: response.name,
      isOnline: response.status === BASE_STATION_STATUS.ONLINE,
      lastSeen: "",
      firstSeen: "",
      latitude: response.latitude,
      longitude: response.longitude,
      altitude: response.altitude,
      locationSource: response.locationSource,
      locationUpdatedAt: response.locationUpdatedAt?.toISOString(),
    };
    return mapBaseStation(apiFormat);
  },

  /**
   * Get unified base station activity (events + messages)
   */
  async getBaseStationActivity(
    eui: string,
    filter: ActivityFilter,
    pageToken: string,
    pageSize: number,
  ): Promise<ActivityPage> {
    const response = await baseStationsRpc.listBaseStationActivity(eui, {
      pageSize,
      pageToken,
      startTime: filter.startTime ? new Date(filter.startTime) : undefined,
      endTime: filter.endTime ? new Date(filter.endTime) : undefined,
    });

    return mapActivityPage(response, eui);
  },

  async exportBaseStationMessages(
    eui: string,
    filter?: BaseStationMessagesFilter,
    format: UplinkExportFormat = UPLINK_EXPORT_FORMAT.CSV,
  ): Promise<Blob> {
    const filterParams = buildMessageFilterParams(filter, 0);
    const response = await baseStationsRpc.exportBaseStationMessages(
      eui,
      format,
      {
        direction: filterParams.direction,
        epEui: filterParams.epEui,
        startTime: filterParams.startTime,
        endTime: filterParams.endTime,
      },
    );
    return new Blob([new Uint8Array(response.content)], {
      type: response.contentType,
    });
  },
};
