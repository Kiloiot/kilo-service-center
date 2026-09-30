/**
 * Says how many of all end points a narrowed list shows and by what, with a
 * way back to the whole list; renders nothing for an unnarrowed list.
 */

import React from "react";

import { Box, Button, Chip, Typography } from "@mui/material";

import { attachStatusChip } from "@utils/chipMappings";
import { ENDPOINT_ACTIVITY } from "@constants/app";
import { ENDPOINTS_PAGE } from "@constants/messages";

interface EndpointsListScopeProps {
  shown: number;
  total: number;
  search: string;
  attachState: string[];
  activity: string[];
  onClear: () => void;
}

const activityLabel = (value: string): string =>
  value === ENDPOINT_ACTIVITY.ACTIVE
    ? ENDPOINTS_PAGE.ACTIVE
    : ENDPOINTS_PAGE.INACTIVE;

export const EndpointsListScope: React.FC<EndpointsListScopeProps> = ({
  shown,
  total,
  search,
  attachState,
  activity,
  onClear,
}) => {
  const labels = [
    ...(search ? [`${ENDPOINTS_PAGE.SEARCH_CHIP_PREFIX} ${search}`] : []),
    ...attachState.map((state) => attachStatusChip(state).label),
    ...activity.map(activityLabel),
  ];
  if (labels.length === 0) return null;
  return (
    <Box
      data-testid="endpoints-list-scope"
      display="flex"
      alignItems="center"
      flexWrap="wrap"
      gap={1}
      mb={2}
    >
      <Typography variant="body2" color="text.secondary">
        {`${ENDPOINTS_PAGE.SHOWING} ${shown} ${ENDPOINTS_PAGE.OF} ${total}`}
      </Typography>
      {labels.map((label) => (
        <Chip key={label} label={label} size="small" variant="outlined" />
      ))}
      <Button size="small" onClick={onClear}>
        {ENDPOINTS_PAGE.CLEAR_FILTERS}
      </Button>
    </Box>
  );
};
