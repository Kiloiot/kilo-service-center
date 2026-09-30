/**
 * Endpoint overview: identity, attach state, blueprint and the attach/detach/edit/delete actions.
 */

import React from "react";

import {
  Box,
  Button,
  Chip,
  IconButton,
  Tooltip,
  Typography,
} from "@mui/material";
import Grid from "@mui/material/Grid";

import { formatDateTime } from "@utils/date-format";
import { formatEui } from "@utils/eui";
import { getMonoBody1 } from "@utils/typography";
import type { EndpointActivity, EndpointAttachStatus } from "@constants/app";
import { ENDPOINT_ATTACH_STATUS } from "@constants/app";
import { ENDPOINT_DETAILS } from "@constants/messages";
import { DeleteIcon, EditIcon, LinkIcon, LinkOffIcon } from "@theme/icons";
import { componentSpacing } from "@theme/index";

import AttachStatusChip from "./AttachStatusChip";
import {
  type BlueprintInfo,
  EndpointBlueprintSection,
} from "./EndpointBlueprintSection";
import ReattachPendingBadge from "./ReattachPendingBadge";

export interface EndPointInput {
  id: string;
  epEui: string;
  name?: string;
  lastSeen?: string;
  status: EndpointActivity;
  attachStatus?: EndpointAttachStatus;
  reattachPending?: boolean;
  ownerTenantId?: number;
  isRoaming?: boolean;
}

export interface EndpointOverviewPanelProps {
  endPoint: EndPointInput;
  endpointData: EndPointInput;
  blueprintInfo: BlueprintInfo | null;
  isAttaching: boolean;
  isDetaching: boolean;
  onAttach: () => void;
  onDetach: () => void;
  onEdit: () => void;
  onDeleteClick: () => void;
}

export const EndpointOverviewPanel: React.FC<EndpointOverviewPanelProps> = ({
  endPoint,
  endpointData,
  blueprintInfo,
  isAttaching,
  isDetaching,
  onAttach,
  onDetach,
  onEdit,
  onDeleteClick,
}) => (
  <>
    <Box display="flex" justifyContent="flex-end" alignItems="center" mb={2}>
      <Box>
        <Tooltip title={ENDPOINT_DETAILS.TOOLTIP_EDIT}>
          <IconButton size="small" onClick={onEdit}>
            <EditIcon />
          </IconButton>
        </Tooltip>
        <Tooltip title={ENDPOINT_DETAILS.TOOLTIP_DELETE}>
          <IconButton size="small" color="error" onClick={onDeleteClick}>
            <DeleteIcon />
          </IconButton>
        </Tooltip>
      </Box>
    </Box>

    <Grid container spacing={3}>
      <Grid size={componentSpacing.gridSpan.half}>
        <Typography
          variant="subtitle2"
          color="text.secondary"
          gutterBottom
          sx={{ mb: 2 }}
        >
          {ENDPOINT_DETAILS.SECTION_DEVICE_INFO}
        </Typography>
        {endPoint.name && (
          <Box mb={1}>
            <Typography variant="body2" color="text.secondary">
              {ENDPOINT_DETAILS.LABEL_DEVICE_NAME}
            </Typography>
            <Typography variant="body1">{endPoint.name}</Typography>
          </Box>
        )}
        <Box mb={1}>
          <Typography variant="body2" color="text.secondary">
            {ENDPOINT_DETAILS.LABEL_DEVICE_EUI}
          </Typography>
          <Typography variant="body1" sx={(theme) => getMonoBody1(theme)}>
            {formatEui(endPoint.epEui)}
          </Typography>
        </Box>
      </Grid>

      <Grid size={componentSpacing.gridSpan.half}>
        <Typography
          variant="subtitle2"
          color="text.secondary"
          gutterBottom
          sx={{ mb: 2 }}
        >
          {ENDPOINT_DETAILS.SECTION_STATUS_INFO}
        </Typography>
        <Box mb={1}>
          <Typography variant="body2" color="text.secondary">
            {ENDPOINT_DETAILS.LABEL_ATTACH_STATUS}
          </Typography>
          <Box display="flex" alignItems="center" gap={1}>
            <AttachStatusChip status={endpointData.attachStatus} />
            <ReattachPendingBadge pending={endpointData.reattachPending} />
          </Box>
        </Box>
        {endpointData.isRoaming !== undefined && (
          <Box mb={1}>
            <Typography variant="body2" color="text.secondary">
              {ENDPOINT_DETAILS.LABEL_ROAMING_STATUS}
            </Typography>
            <Box display="flex" alignItems="center" gap={1}>
              <Chip
                label={
                  endpointData.isRoaming
                    ? ENDPOINT_DETAILS.STATUS_ROAMING
                    : ENDPOINT_DETAILS.STATUS_HOME_NETWORK
                }
                color={endpointData.isRoaming ? "info" : "default"}
                size="small"
                icon={endpointData.isRoaming ? <LinkIcon /> : undefined}
              />
              {endpointData.ownerTenantId && (
                <Typography variant="caption" color="text.secondary">
                  ({ENDPOINT_DETAILS.LABEL_OWNER_TENANT}{" "}
                  {endpointData.ownerTenantId})
                </Typography>
              )}
            </Box>
          </Box>
        )}
        <Box mb={1}>
          <Typography variant="body2" color="text.secondary">
            {ENDPOINT_DETAILS.LABEL_LAST_SEEN}
          </Typography>
          <Typography
            variant="body1"
            color={endPoint.lastSeen ? "inherit" : "error"}
          >
            {endPoint.lastSeen
              ? formatDateTime(endPoint.lastSeen)
              : ENDPOINT_DETAILS.STATUS_NEVER}
          </Typography>
        </Box>
        <Box mt={2}>
          {endpointData.attachStatus !== ENDPOINT_ATTACH_STATUS.ATTACHED ? (
            <Button
              variant="contained"
              color="primary"
              startIcon={<LinkIcon />}
              onClick={onAttach}
              disabled={isAttaching}
              size="small"
            >
              {isAttaching
                ? ENDPOINT_DETAILS.ACTION_ATTACHING
                : ENDPOINT_DETAILS.ACTION_ATTACH_EP}
            </Button>
          ) : (
            <Button
              variant="outlined"
              color="warning"
              startIcon={<LinkOffIcon />}
              onClick={onDetach}
              disabled={isDetaching}
              size="small"
            >
              {isDetaching
                ? ENDPOINT_DETAILS.ACTION_DETACHING
                : ENDPOINT_DETAILS.ACTION_DETACH_EP}
            </Button>
          )}
        </Box>
      </Grid>

      <Grid size={componentSpacing.gridSpan.full}>
        <EndpointBlueprintSection
          blueprintInfo={blueprintInfo}
          onAssign={onEdit}
        />
      </Grid>
    </Grid>
  </>
);
