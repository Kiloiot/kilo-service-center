import React from "react";

import { Stack, Typography } from "@mui/material";

import { DataCard } from "@components/common/DataCard";
import { MiniBarChart } from "@components/common/MiniBarChart";
import { useBaseStationAvailability } from "@hooks/useBaseStationOperations";
import { formatDateTime } from "@utils/date-format";
import { formatFraction } from "@utils/formatters";
import { BS_OPERATIONS } from "@constants/messages";
import { componentSpacing } from "@theme/index";

const FULL_AVAILABILITY = 1;

/** GetBaseStationAvailability: the hourly share of the last day the station was connected. */
export const BaseStationAvailabilityCard: React.FC<{ bsEui: string }> = ({
  bsEui,
}) => {
  const { data, isLoading, error } = useBaseStationAvailability(bsEui);

  return (
    <DataCard
      title={BS_OPERATIONS.AVAILABILITY_TITLE}
      subtitle={BS_OPERATIONS.AVAILABILITY_HINT}
      isLoading={isLoading}
      error={error}
      errorFallback={BS_OPERATIONS.ERR_AVAILABILITY}
    >
      {data && (
        <Stack spacing={componentSpacing.infoGrid.rowGap}>
          <Typography variant="h4" component="p">
            {formatFraction(data.overall)}
          </Typography>
          <MiniBarChart
            max={FULL_AVAILABILITY}
            ariaLabel={BS_OPERATIONS.AVAILABILITY_TITLE}
            bars={data.buckets.map((bucket) => ({
              key: bucket.start,
              label: `${formatDateTime(bucket.start)}: ${formatFraction(bucket.availability)}`,
              value: bucket.availability,
            }))}
          />
        </Stack>
      )}
    </DataCard>
  );
};
