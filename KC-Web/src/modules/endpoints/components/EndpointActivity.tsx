/**
 * Activity tab of the end point page: the end point's events and uplinks,
 * each uplink scoped to the base stations that received it.
 */

import { useEndpointActivity } from "@hooks";

import { ActivityPanel } from "@components/common/activity/ActivityPanel";
import { uplinkStationsScope } from "@utils/uplink-summary";

export function EndpointActivity({ epEui }: { epEui: string }) {
  return (
    <ActivityPanel
      eui={epEui}
      useFeed={useEndpointActivity}
      uplinkScope={uplinkStationsScope}
    />
  );
}
