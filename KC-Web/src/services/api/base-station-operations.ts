/**
 * Base station operations in the shapes the UI consumes.
 */

import type {
  BaseStationAvailabilityUI,
  PingResultUI,
} from "@api-types/base-station-operations";
import type { AnalyticsWindow } from "@api-types/system";
import { mapBaseStationAvailability } from "@mappers";

import * as operationsRpc from "@services/grpc/rpc/base-station-operations";

export const baseStationOperationsApi = {
  async getAvailability(
    bsEui: string,
    window: AnalyticsWindow,
    intervalSeconds: number,
  ): Promise<BaseStationAvailabilityUI> {
    const availability = await operationsRpc.getBaseStationAvailability(
      bsEui,
      window,
      intervalSeconds,
    );
    return mapBaseStationAvailability(availability, window.startTime);
  },

  ping(bsEui: string): Promise<PingResultUI> {
    return operationsRpc.initiatePing(bsEui);
  },
};
