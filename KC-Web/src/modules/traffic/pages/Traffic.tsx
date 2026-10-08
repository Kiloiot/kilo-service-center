/**
 * Traffic (NAV_STRUCTURE §4): the organization's uplinks, downlink queue and
 * downlink results.
 */

import React from "react";

import { Box, Typography } from "@mui/material";

import { TRAFFIC_PAGE } from "@constants/messages";

import { TrafficViews } from "../components/TrafficViews";

const TENANT_SCOPE = {};

const Traffic: React.FC = () => (
  <Box>
    <Typography variant="h4" component="h1" gutterBottom>
      {TRAFFIC_PAGE.TITLE}
    </Typography>
    <TrafficViews scope={TENANT_SCOPE} />
  </Box>
);

export default Traffic;
