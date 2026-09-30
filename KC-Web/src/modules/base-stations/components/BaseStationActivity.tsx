/**
 * Activity tab of the base station page: the station's events and the
 * uplinks it heard, each uplink scoped to its endpoint, with the export of
 * the uplinks in the chosen date range.
 */

import { useBaseStationActivity } from "@hooks";

import { ActivityPanel } from "@components/common/activity/ActivityPanel";
import { uplinkEndpointScope } from "@utils/uplink-summary";

import { UplinkExportButtons } from "./UplinkExportButtons";

export function BaseStationActivity({ bsEui }: { bsEui: string }) {
  return (
    <ActivityPanel
      eui={bsEui}
      useFeed={useBaseStationActivity}
      uplinkScope={uplinkEndpointScope}
      renderActions={(filter) => (
        <UplinkExportButtons bsEui={bsEui} filter={filter} />
      )}
    />
  );
}
