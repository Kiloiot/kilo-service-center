/**
 * Traffic tabs of the base station and endpoint detail pages
 * (NAV_STRUCTURE §2 and §3): the device's uplink totals above the Traffic
 * views, narrowed to that device (a base station's queue is the downlinks
 * it holds, its uplinks come from the station's own listing).
 */

import {
  useBaseStationTrafficSummary,
  useEndpointTrafficSummary,
} from "@hooks";
import { Box } from "@mui/material";

import { UPLINK_SOURCE } from "@constants/app";

import { TrafficSummaryCard } from "./TrafficSummaryCard";
import { TrafficViews } from "./TrafficViews";

export function BaseStationTrafficTab({ bsEui }: { bsEui: string }) {
  const summary = useBaseStationTrafficSummary(bsEui);
  return (
    <Box>
      <TrafficSummaryCard summary={summary.data} error={summary.error} />
      <TrafficViews scope={{ bsEui }} uplinkSource={UPLINK_SOURCE.STATION} />
    </Box>
  );
}

export function EndpointTrafficTab({ epEui }: { epEui: string }) {
  const summary = useEndpointTrafficSummary(epEui);
  return (
    <Box>
      <TrafficSummaryCard summary={summary.data} error={summary.error} />
      <TrafficViews scope={{ epEui }} />
    </Box>
  );
}
