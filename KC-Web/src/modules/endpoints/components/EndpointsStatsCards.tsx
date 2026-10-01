/**
 * End Point totals: every registered end point, whatever the list shows.
 */

import React from "react";

import { StatCard, StatCardRow } from "@components/common/StatCard";
import { ENDPOINTS_PAGE } from "@constants/messages";
import { EndPointIcon, ErrorIcon, SuccessIcon } from "@theme/icons";

interface EndpointsStatsCardsProps {
  total: number;
  activeCount: number;
}

export const EndpointsStatsCards: React.FC<EndpointsStatsCardsProps> = ({
  total,
  activeCount,
}) => (
  <StatCardRow>
    <StatCard
      label={ENDPOINTS_PAGE.TOTAL_ENDPOINTS}
      value={total}
      icon={<EndPointIcon />}
      color="primary"
    />
    <StatCard
      label={ENDPOINTS_PAGE.ACTIVE}
      value={activeCount}
      icon={<SuccessIcon />}
      color="success"
    />
    <StatCard
      label={ENDPOINTS_PAGE.INACTIVE}
      value={total - activeCount}
      icon={<ErrorIcon />}
      color="error"
    />
  </StatCardRow>
);
