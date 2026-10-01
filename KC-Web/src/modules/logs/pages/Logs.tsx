/**
 * Logs (NAV_STRUCTURE §5): every recorded operation, the failures grouped in
 * the Errors Center, and the operators' own actions in the Audit Log.
 */

import React, { type ReactElement, type ReactNode } from "react";

import { Box, Typography } from "@mui/material";

import {
  EVENT_CELLS,
  eventColumns,
} from "@components/common/events/eventColumns";
import { RoleGate } from "@components/common/RoleGate";
import { ViewTabs } from "@components/common/ViewTabs";
import {
  EVENT_CATEGORY,
  LOG_VIEWS,
  type LogView,
  ROLE_REQUIREMENT,
  type RoleRequirement,
} from "@constants/app";
import { LOG_VIEW_LABELS, LOGS_PAGE } from "@constants/messages";
import { AuditIcon, ErrorsCenterIcon, EventLogIcon } from "@theme/icons";

import { ErrorsCenter } from "../components/ErrorsCenter";
import { EventLogPanel } from "../components/EventLogPanel";
import { AUDIT_COLUMNS } from "../components/logColumns";
import { AUDIT_FIELDS, EVENT_FIELDS } from "../components/logFields";

const TENANT_SCOPE = {};
const EVENT_COLUMNS = eventColumns(EVENT_CELLS);
const AUDIT_ONLY = { category: EVENT_CATEGORY.AUDIT };

const VIEW_PANELS: Record<LogView, () => ReactNode> = {
  events: () => (
    <EventLogPanel
      scope={TENANT_SCOPE}
      fields={EVENT_FIELDS}
      columns={EVENT_COLUMNS}
    />
  ),
  errors: () => <ErrorsCenter />,
  audit: () => (
    <EventLogPanel
      scope={TENANT_SCOPE}
      fixed={AUDIT_ONLY}
      fields={AUDIT_FIELDS}
      columns={AUDIT_COLUMNS}
    />
  ),
};

// Device events reach base station and endpoint managers, filtered to their categories; audit events only admins.
const VIEW_REQUIRES: Record<LogView, RoleRequirement> = {
  events: ROLE_REQUIREMENT.DEVICE_MANAGER,
  errors: ROLE_REQUIREMENT.DEVICE_MANAGER,
  audit: ROLE_REQUIREMENT.ADMIN,
};

const VIEW_ICONS: Record<LogView, ReactElement> = {
  events: <EventLogIcon />,
  errors: <ErrorsCenterIcon />,
  audit: <AuditIcon />,
};

const Logs: React.FC = () => (
  <Box>
    <Typography variant="h4" component="h1" gutterBottom>
      {LOGS_PAGE.TITLE}
    </Typography>
    <ViewTabs
      views={LOG_VIEWS}
      labels={LOG_VIEW_LABELS}
      icons={VIEW_ICONS}
      ariaLabel={LOGS_PAGE.ARIA_VIEWS}
    >
      {(view) => (
        <RoleGate requires={VIEW_REQUIRES[view]}>
          {VIEW_PANELS[view]()}
        </RoleGate>
      )}
    </ViewTabs>
  </Box>
);

export default Logs;
