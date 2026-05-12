import React, { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";

import { useBaseStation, useDeleteBaseStation } from "@hooks";
import {
  Alert,
  Box,
  Card,
  CardContent,
  Divider,
  Grid,
  IconButton,
  Snackbar,
  Tooltip,
} from "@mui/material";

import { realtimeService } from "@services/realtime";
import {
  calculateDaysUntilExpiry,
  formatEUIWithDashes,
} from "@utils/formatters";
import { BS_DETAIL_LAYOUT } from "@constants/app";
import {
  ACTION_DELETE,
  ACTION_EDIT,
  ERR_DELETE_BASE_STATION,
  MSG_BS_EUI_UPDATED,
} from "@constants/messages";
import { DeleteIcon, EditIcon } from "@theme/icons";

import BaseStationDeleteDialog from "./BaseStationDeleteDialog";
import BaseStationEditDialog from "./BaseStationEditDialog";
import BaseStationInfoPanel from "./BaseStationInfoPanel";
import BaseStationLocationMap from "./BaseStationLocationMap";
import BaseStationMessages from "./BaseStationMessages";

interface BaseStationDetailsProps {
  baseStation: {
    id: string;
    eui: string;
    name?: string;
    status: "online" | "offline";
    connectionType: "BSSCI" | "MQTT";
    lastSeen: string;
    serviceCenterUrl: string;
    certificateExpiryDate?: string;
  };
  onDelete?: (id: string) => void;
}

const BaseStationDetails: React.FC<BaseStationDetailsProps> = ({
  baseStation,
  onDelete,
}) => {
  const navigate = useNavigate();
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  const deleteBaseStationMutation = useDeleteBaseStation();

  const {
    data: baseStationDetails,
    isLoading: loadingDetails,
    error: detailsError,
  } = useBaseStation(baseStation.eui);

  useEffect(() => {
    realtimeService.connectBaseStationStream(baseStation.eui);
    return () => realtimeService.disconnectBaseStationStream();
  }, [baseStation.eui]);

  const daysUntilExpiry = calculateDaysUntilExpiry(
    baseStation.certificateExpiryDate,
  );
  const isExpiringSoon = daysUntilExpiry !== null && daysUntilExpiry <= 30;
  const isExpired = daysUntilExpiry !== null && daysUntilExpiry < 0;

  const handleEditSuccess = (newEui?: string) => {
    if (newEui) {
      setSuccessMessage(MSG_BS_EUI_UPDATED);
      setEditDialogOpen(false);
      navigate(`/base-stations/${formatEUIWithDashes(newEui)}`);
    } else {
      setEditDialogOpen(false);
    }
  };

  const handleEditError = (message: string) => {
    if (message) setErrorMessage(message);
  };

  const handleDeleteConfirm = () => {
    deleteBaseStationMutation.mutate(baseStation.eui, {
      onSuccess: () => {
        setDeleteDialogOpen(false);
        if (onDelete) {
          onDelete(baseStation.id);
        }
      },
      onError: () => {
        setErrorMessage(ERR_DELETE_BASE_STATION);
      },
    });
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
            <Tooltip title={ACTION_EDIT}>
              <IconButton
                size="small"
                onClick={() => setEditDialogOpen(true)}
              >
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
          <Grid size={{ xs: 12, md: 3 }}>
            <BaseStationLocationMap
              latitude={baseStationDetails?.latitude}
              longitude={baseStationDetails?.longitude}
              altitude={baseStationDetails?.altitude}
              locationSource={baseStationDetails?.locationSource}
            />
          </Grid>

          <Grid size={{ xs: 12, md: 9 }}>
            <BaseStationInfoPanel
              baseStation={baseStation}
              baseStationDetails={baseStationDetails}
              loadingDetails={loadingDetails}
              detailsError={detailsError}
              isExpiringSoon={isExpiringSoon}
              isExpired={isExpired}
            />
          </Grid>

          <Grid size={12}>
            <Divider sx={{ my: 1 }} />
          </Grid>

          <Grid size={12}>
            <BaseStationMessages
              bsEui={baseStation.eui}
              basestationName={
                baseStation.name || formatEUIWithDashes(baseStation.eui)
              }
            />
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
        onSuccess={handleEditSuccess}
        onError={handleEditError}
      />

      <Snackbar
        open={!!errorMessage}
        autoHideDuration={6000}
        onClose={() => setErrorMessage(null)}
        anchorOrigin={{ vertical: "bottom", horizontal: "center" }}
      >
        <Alert severity="error" onClose={() => setErrorMessage(null)}>
          {errorMessage}
        </Alert>
      </Snackbar>

      <Snackbar
        open={!!successMessage}
        autoHideDuration={4000}
        onClose={() => setSuccessMessage(null)}
        anchorOrigin={{ vertical: "bottom", horizontal: "center" }}
      >
        <Alert severity="success" onClose={() => setSuccessMessage(null)}>
          {successMessage}
        </Alert>
      </Snackbar>
    </Card>
  );
};

export default BaseStationDetails;
