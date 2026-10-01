import React, {
  type ReactElement,
  type ReactNode,
  useEffect,
  useState,
} from "react";

import { useBaseStation, useDeleteBaseStation } from "@hooks";
import {
  Box,
  Card,
  CardContent,
  Divider,
  Grid,
  IconButton,
  Tooltip,
} from "@mui/material";

import { BaseStationTrafficTab } from "@modules/traffic";
import { ViewTabs } from "@components/common/ViewTabs";
import { realtimeStreams } from "@services/realtime";
import { useFeedback } from "@contexts/feedback";
import type { BaseStationStatus, BsConnectionTypeLabel } from "@constants/app";
import {
  BASE_STATION_DETAIL_VIEWS,
  BASE_STATION_STATUS,
  type BaseStationDetailView,
  BS_DETAIL_LAYOUT,
} from "@constants/app";
import {
  ACTION_DELETE,
  ACTION_EDIT,
  BASE_STATION_DETAIL_VIEW_LABELS,
  DEVICE_DETAIL_TABS,
  MSG_BS_UPDATED,
} from "@constants/messages";
import { DeleteIcon, EditIcon, MessageIcon, TrafficIcon } from "@theme/icons";
import { componentSpacing } from "@theme/index";

import { BaseStationActivity } from "./BaseStationActivity";
import BaseStationDeleteDialog from "./BaseStationDeleteDialog";
import BaseStationEditDialog from "./BaseStationEditDialog";
import BaseStationInfoPanel from "./BaseStationInfoPanel";
import BaseStationLocationMap from "./BaseStationLocationMap";
import { BaseStationOperationsPanel } from "./BaseStationOperationsPanel";
import { BaseStationStatusRequestButton } from "./BaseStationStatusRequestButton";

const BASE_STATION_DETAIL_ICONS: Record<BaseStationDetailView, ReactElement> = {
  activity: <MessageIcon />,
  traffic: <TrafficIcon />,
};

interface BaseStationDetailsProps {
  baseStation: {
    id: string;
    eui: string;
    name?: string;
    status: BaseStationStatus;
    connectionType: BsConnectionTypeLabel;
    lastSeen: string;
    serviceCenterUrl: string;
    certificateExpiryDate?: string;
  };
  onDelete?: (id: string) => void;
  onEuiChange: (newEui: string, changesSaved: boolean) => void;
}

const BaseStationDetails: React.FC<BaseStationDetailsProps> = ({
  baseStation,
  onDelete,
  onEuiChange,
}) => {
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const feedback = useFeedback();

  const deleteBaseStationMutation = useDeleteBaseStation();

  const {
    data: baseStationDetails,
    isLoading: loadingDetails,
    error: detailsError,
  } = useBaseStation(baseStation.eui);

  useEffect(
    () => realtimeStreams.watchBaseStation(baseStation.eui),
    [baseStation.eui],
  );

  const viewPanels: Record<BaseStationDetailView, () => ReactNode> = {
    activity: () => <BaseStationActivity bsEui={baseStation.eui} />,
    traffic: () => <BaseStationTrafficTab bsEui={baseStation.eui} />,
  };

  const handleEuiChange = (newEui: string, changesSaved: boolean) => {
    setEditDialogOpen(false);
    onEuiChange(newEui, changesSaved);
  };

  const handleEditSaved = () => {
    setEditDialogOpen(false);
    feedback.success(MSG_BS_UPDATED);
  };

  const handleDeleteConfirm = async () => {
    await deleteBaseStationMutation.mutateAsync(baseStation.eui);
    setDeleteDialogOpen(false);
    onDelete?.(baseStation.id);
  };

  return (
    <Card sx={{ mt: 2, mb: 2 }}>
      <CardContent>
        <Box
          display="flex"
          justifyContent="flex-end"
          alignItems="center"
          mb={2}
        >
          <Box>
            <BaseStationStatusRequestButton bsEui={baseStation.eui} />
            <Tooltip title={ACTION_EDIT}>
              <IconButton size="small" onClick={() => setEditDialogOpen(true)}>
                <EditIcon />
              </IconButton>
            </Tooltip>
            <Tooltip title={ACTION_DELETE}>
              <IconButton
                size="small"
                color="error"
                onClick={() => setDeleteDialogOpen(true)}
              >
                <DeleteIcon />
              </IconButton>
            </Tooltip>
          </Box>
        </Box>

        <Grid container spacing={BS_DETAIL_LAYOUT.GRID_SPACING}>
          <Grid size={componentSpacing.gridSpan.quarter}>
            <BaseStationLocationMap
              latitude={baseStationDetails?.latitude}
              longitude={baseStationDetails?.longitude}
              altitude={baseStationDetails?.altitude}
              locationSource={baseStationDetails?.locationSource}
            />
          </Grid>

          <Grid size={componentSpacing.gridSpan.threeQuarters}>
            <BaseStationInfoPanel
              baseStation={baseStation}
              baseStationDetails={baseStationDetails}
              loadingDetails={loadingDetails}
              detailsError={detailsError}
            />
          </Grid>

          <Grid size={componentSpacing.gridSpan.full}>
            <BaseStationOperationsPanel
              bsEui={baseStation.eui}
              online={baseStation.status === BASE_STATION_STATUS.ONLINE}
              certificateExpiresAt={baseStationDetails?.certificateExpiryDate}
              certificateFingerprint={
                baseStationDetails?.certificateFingerprint
              }
              lastHandshake={baseStationDetails?.lastHandshake}
            />
          </Grid>

          <Grid size={componentSpacing.gridSpan.full}>
            <Divider sx={{ my: 1 }} />
          </Grid>

          <Grid size={componentSpacing.gridSpan.full}>
            <ViewTabs
              views={BASE_STATION_DETAIL_VIEWS}
              labels={BASE_STATION_DETAIL_VIEW_LABELS}
              icons={BASE_STATION_DETAIL_ICONS}
              ariaLabel={DEVICE_DETAIL_TABS.ARIA_BASE_STATION_TABS}
            >
              {(view) => viewPanels[view]()}
            </ViewTabs>
          </Grid>
        </Grid>
      </CardContent>

      <BaseStationDeleteDialog
        open={deleteDialogOpen}
        onClose={() => setDeleteDialogOpen(false)}
        baseStationName={baseStation.name}
        eui={baseStation.eui}
        onConfirm={handleDeleteConfirm}
        isPending={deleteBaseStationMutation.isPending}
      />

      <BaseStationEditDialog
        open={editDialogOpen}
        onClose={() => setEditDialogOpen(false)}
        baseStation={baseStation}
        baseStationDetails={baseStationDetails}
        onSuccess={handleEditSaved}
        onEuiChange={handleEuiChange}
      />
    </Card>
  );
};

export default BaseStationDetails;
