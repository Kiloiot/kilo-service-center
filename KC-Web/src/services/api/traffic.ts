/**
 * Uplink traffic in the shapes the UI consumes.
 */

import type {
  ListResult,
  Page,
  TrafficSummaryUI,
  UplinkAPI,
  UplinkFilter,
  UplinkUI,
} from "@api-types/api";
import { mapStationUplink, mapUplink } from "@mappers";

import * as trafficRpc from "@services/grpc/rpc/traffic";
import { timeRangeStart } from "@utils/date-format";

import { euiParam, offsetPageToken } from "./paging";

/** The request fields both uplink listings share. */
function uplinkQuery(filter: UplinkFilter, page: number, pageSize: number) {
  return {
    epEui: euiParam(filter.epEui),
    duplicate: filter.duplicate,
    dlOpen: filter.dlOpen,
    profile: filter.profile,
    mode: filter.mode,
    startTime: timeRangeStart(filter.timeRange, new Date()),
    pageSize,
    pageToken: offsetPageToken(page, pageSize),
  };
}

function uplinkPage(
  result: ListResult<UplinkAPI>,
  toRow: (message: UplinkAPI) => UplinkUI,
): Page<UplinkUI> {
  return { items: result.items.map(toRow), totalCount: result.totalCount };
}

export const trafficApi = {
  async listUplinks(
    filter: UplinkFilter,
    page: number,
    pageSize: number,
  ): Promise<Page<UplinkUI>> {
    const result = await trafficRpc.listMessages({
      ...uplinkQuery(filter, page, pageSize),
      bsEui: euiParam(filter.bsEui),
    });
    return uplinkPage(result, (message) => mapUplink(message, filter.bsEui));
  },

  /** The uplinks one base station heard; filter.bsEui names the station. */
  async listStationUplinks(
    filter: UplinkFilter,
    page: number,
    pageSize: number,
  ): Promise<Page<UplinkUI>> {
    const result = await trafficRpc.listBaseStationMessages({
      ...uplinkQuery(filter, page, pageSize),
      bsEui: euiParam(filter.bsEui),
    });
    return uplinkPage(result, mapStationUplink);
  },

  getBaseStationSummary(bsEui: string): Promise<TrafficSummaryUI> {
    return trafficRpc.getBaseStationMessageStats(bsEui);
  },

  getEndpointSummary(epEui: string): Promise<TrafficSummaryUI> {
    return trafficRpc.getEndPointStats(epEui);
  },
};
