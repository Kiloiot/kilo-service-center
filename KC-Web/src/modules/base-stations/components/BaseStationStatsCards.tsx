import type { ReactElement } from "react";

import {
  StatCard,
  type StatCardColor,
  StatCardRow,
} from "@components/common/StatCard";
import type { BaseStationStatus } from "@constants/app";
import { BASE_STATION_STATUS } from "@constants/app";
import { BASE_STATIONS_PAGE } from "@constants/messages";
import {
  CheckCircleIcon,
  ErrorIcon,
  RouterIcon,
  WarningIcon,
} from "@theme/icons";

import type { BaseStationStats, ListOverlay } from "../utils/base-station-list";

interface BaseStationStatsCardsProps {
  stats: BaseStationStats;
  overlay: ListOverlay;
  onToggleStatus: (status: BaseStationStatus) => void;
  onToggleCertExpiry: () => void;
}

function expiryCard(stats: BaseStationStats): {
  color: StatCardColor;
  icon: ReactElement;
} {
  if (stats.expiringCritical > 0) {
    return { color: "error", icon: <ErrorIcon /> };
  }
  if (stats.expiringSoon > 0) {
    return { color: "warning", icon: <WarningIcon /> };
  }
  return { color: "success", icon: <CheckCircleIcon /> };
}

export default function BaseStationStatsCards({
  stats,
  overlay,
  onToggleStatus,
  onToggleCertExpiry,
}: BaseStationStatsCardsProps) {
  const expiry = expiryCard(stats);

  return (
    <StatCardRow>
      <StatCard
        label={BASE_STATIONS_PAGE.TOTAL_BASE_STATIONS}
        value={stats.total}
        icon={<RouterIcon />}
        color="primary"
      />
      <StatCard
        label={BASE_STATIONS_PAGE.ONLINE}
        value={stats.online}
        icon={<CheckCircleIcon />}
        color="success"
        caption={
          overlay.status === BASE_STATION_STATUS.ONLINE
            ? BASE_STATIONS_PAGE.SORTED_BY_ONLINE
            : undefined
        }
        onClick={() => onToggleStatus(BASE_STATION_STATUS.ONLINE)}
      />
      <StatCard
        label={BASE_STATIONS_PAGE.OFFLINE}
        value={stats.offline}
        icon={<ErrorIcon />}
        color="error"
        caption={
          overlay.status === BASE_STATION_STATUS.OFFLINE
            ? BASE_STATIONS_PAGE.SORTED_BY_OFFLINE
            : undefined
        }
        onClick={() => onToggleStatus(BASE_STATION_STATUS.OFFLINE)}
      />
      <StatCard
        label={BASE_STATIONS_PAGE.EXPIRING_CERTIFICATES}
        value={stats.expiringSoon}
        icon={expiry.icon}
        color={expiry.color}
        caption={
          overlay.certExpiry ? BASE_STATIONS_PAGE.SORTED_BY_EXPIRY : undefined
        }
        onClick={onToggleCertExpiry}
      />
    </StatCardRow>
  );
}
