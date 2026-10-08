/**
 * The Traffic views (uplinks, downlink queue, downlink results), each
 * rendered for the same device scope; an empty scope is the whole tenant.
 * The uplink source decides which listing, and so which role, serves the
 * uplink view.
 */

import type { ReactElement, ReactNode } from "react";

import type { DeviceScope } from "@api-types/api";

import { RoleGate } from "@components/common/RoleGate";
import { ViewTabs } from "@components/common/ViewTabs";
import {
  ROLE_REQUIREMENT,
  type RoleRequirement,
  TRAFFIC_VIEW,
  TRAFFIC_VIEWS,
  type TrafficView,
  UPLINK_SOURCE,
  type UplinkSource,
} from "@constants/app";
import { TRAFFIC_PAGE, TRAFFIC_VIEW_LABELS } from "@constants/messages";
import {
  DownlinkQueueIcon,
  DownlinkResultIcon,
  UplinkIcon,
} from "@theme/icons";

import { DownlinkQueuePanel } from "./DownlinkQueuePanel";
import { DownlinkResultsPanel } from "./DownlinkResultsPanel";
import { UplinkPanel } from "./UplinkPanel";

const VIEW_PANELS: Record<
  TrafficView,
  (scope: DeviceScope, source: UplinkSource) => ReactNode
> = {
  uplinks: (scope, source) => <UplinkPanel scope={scope} source={source} />,
  downlinkQueue: (scope) => <DownlinkQueuePanel scope={scope} />,
  downlinkResults: (scope) => <DownlinkResultsPanel scope={scope} />,
};

const VIEW_ICONS: Record<TrafficView, ReactElement> = {
  uplinks: <UplinkIcon />,
  downlinkQueue: <DownlinkQueueIcon />,
  downlinkResults: <DownlinkResultIcon />,
};

// A station's own listing serves base station managers; every other view carries endpoint data.
const UPLINK_SOURCE_REQUIRES: Record<UplinkSource, RoleRequirement> = {
  [UPLINK_SOURCE.TENANT]: ROLE_REQUIREMENT.ENDPOINT_MANAGER,
  [UPLINK_SOURCE.STATION]: ROLE_REQUIREMENT.BASE_STATION_MANAGER,
};

function viewRequires(
  view: TrafficView,
  source: UplinkSource,
): RoleRequirement {
  return view === TRAFFIC_VIEW.UPLINKS
    ? UPLINK_SOURCE_REQUIRES[source]
    : ROLE_REQUIREMENT.ENDPOINT_MANAGER;
}

export function TrafficViews({
  scope,
  uplinkSource = UPLINK_SOURCE.TENANT,
}: {
  scope: DeviceScope;
  uplinkSource?: UplinkSource;
}) {
  return (
    <ViewTabs
      views={TRAFFIC_VIEWS}
      labels={TRAFFIC_VIEW_LABELS}
      icons={VIEW_ICONS}
      ariaLabel={TRAFFIC_PAGE.ARIA_VIEWS}
    >
      {(view) => (
        <RoleGate requires={viewRequires(view, uplinkSource)}>
          {VIEW_PANELS[view](scope, uplinkSource)}
        </RoleGate>
      )}
    </ViewTabs>
  );
}
