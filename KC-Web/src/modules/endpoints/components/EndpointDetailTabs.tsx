/**
 * The named tabs of the end point page: Activity, Downlink, Traffic and
 * Configuration (the last once the end point's details have loaded).
 */

import type { ReactElement, ReactNode } from "react";

import type { EndpointUI } from "@api-types/api";

import { EndpointTrafficTab } from "@modules/traffic";
import { ViewTabs } from "@components/common/ViewTabs";
import { ENDPOINT_DETAIL_VIEWS, type EndpointDetailView } from "@constants/app";
import {
  ENDPOINT_DETAIL_VIEW_LABELS,
  ENDPOINT_DETAILS,
} from "@constants/messages";
import {
  ConfigurationIcon,
  MessageIcon,
  SendIcon,
  TrafficIcon,
} from "@theme/icons";

import { DownlinkTab } from "./DownlinkTab";
import { EndpointActivity } from "./EndpointActivity";
import { EndpointConfigurationPanel } from "./EndpointConfigurationPanel";

const ENDPOINT_DETAIL_ICONS: Record<EndpointDetailView, ReactElement> = {
  activity: <MessageIcon />,
  downlink: <SendIcon />,
  traffic: <TrafficIcon />,
  configuration: <ConfigurationIcon />,
};

interface EndpointDetailTabsProps {
  epEui: string;
  endpoint?: EndpointUI | null;
}

export function EndpointDetailTabs({
  epEui,
  endpoint,
}: EndpointDetailTabsProps) {
  const viewPanels: Record<EndpointDetailView, () => ReactNode> = {
    activity: () => <EndpointActivity epEui={epEui} />,
    downlink: () => (
      <DownlinkTab epEui={epEui} lastPacketCnt={endpoint?.lastPacketCnt} />
    ),
    traffic: () => <EndpointTrafficTab epEui={epEui} />,
    configuration: () =>
      endpoint && <EndpointConfigurationPanel endpoint={endpoint} />,
  };

  return (
    <ViewTabs
      views={ENDPOINT_DETAIL_VIEWS}
      labels={ENDPOINT_DETAIL_VIEW_LABELS}
      icons={ENDPOINT_DETAIL_ICONS}
      ariaLabel={ENDPOINT_DETAILS.ARIA_ENDPOINT_TABS}
    >
      {(view) => viewPanels[view]()}
    </ViewTabs>
  );
}
