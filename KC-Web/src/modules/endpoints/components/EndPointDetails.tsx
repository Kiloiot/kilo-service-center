import React, { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";

import {
  useAttachEndpoint,
  useDeleteEndpoint,
  useDetachEndpoint,
  useEndpoint,
  useEndpointActivity,
} from "@hooks";
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  Divider,
  IconButton,
  Tab,
  Tabs,
  Tooltip,
  Typography,
} from "@mui/material";
import Grid from "@mui/material/Grid";

import { ActivityTimeline } from "@components/common/ActivityTimeline";
import { PaginationControls } from "@components/common/PaginationControls";
import { api } from "@services/api";
import { formatDateTime } from "@utils/formatters";
import { logger } from "@utils/logger";
import { getMonoBody1 } from "@utils/typography";
import { ROUTES } from "@constants/app";
import {
  ACTION_CANCEL,
  ACTION_DELETE,
  ENDPOINT_DETAILS,
  ERR_ATTACH_ENDPOINT,
  ERR_DELETE_ENDPOINT,
  ERR_DETACH_ENDPOINT,
  LOG_ATTACH_FAILED,
  LOG_DELETE_FAILED,
  LOG_DETACH_FAILED,
  LOG_PRE_DELETE_DETACH_FAILED,
} from "@constants/messages";
import {
  BlueprintIcon,
  DeleteIcon,
  EditIcon,
  LinkIcon,
  LinkOffIcon,
  MessageIcon,
  SendIcon,
} from "@theme/icons";

import { DownlinkTab } from "./DownlinkTab";
import EditEndPointDialog from "./EditEndPointDialog";

interface EndPointDetailsProps {
  endPoint: {
    id: string;
    epEui: string;
    name?: string;
    lastSeen?: string;
    status: "active" | "inactive";
    attachStatus?:
      | "attached"
      | "detached"
      | "attaching"
      | "pending"
      | "unknown";
    // Roaming fields.
    ownerTenantId?: number;
    isRoaming?: boolean;
  };
  onDelete?: (id: string) => void;
}

const EndPointDetails: React.FC<EndPointDetailsProps> = ({
  endPoint,
  onDelete,
}) => {
  const navigate = useNavigate();
  const [actionError, setActionError] = useState<string | null>(null);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [activeTab, setActiveTab] = useState(0);

  // React Query mutations for attach/detach/delete with cache invalidation
  const attachMutation = useAttachEndpoint();
  const detachMutation = useDetachEndpoint();
  const deleteMutation = useDeleteEndpoint();

  // Derive loading states from mutations (separate per action)
  const isAttaching = attachMutation.isPending;
  const isDetaching = detachMutation.isPending;
  const isDeleting = deleteMutation.isPending;

  const { data: endpointDetails } = useEndpoint(endPoint.epEui);
  const endpointData = endpointDetails || endPoint;

  // Activity feed pagination (token-based, mirroring BaseStationMessages)
  const [activityPage, setActivityPage] = useState(0);
  const [activityRowsPerPage, setActivityRowsPerPage] = useState(50);
  const [activityPageTokens, setActivityPageTokens] = useState<string[]>([""]);
  const currentActivityToken = activityPageTokens[activityPage] || "";

  const { data: activityData, isLoading: activityLoading } =
    useEndpointActivity(
      endPoint.epEui,
      currentActivityToken,
      activityRowsPerPage,
    );

  // Track next page token for forward navigation
  useEffect(() => {
    if (
      activityData?.nextPageToken &&
      activityPage === activityPageTokens.length - 1
    ) {
      setActivityPageTokens((prev) => [...prev, activityData.nextPageToken!]);
    }
  }, [activityData?.nextPageToken, activityPage, activityPageTokens.length]);

  const error = actionError;

  // Blueprint device model info (read-only display)
  const [blueprintInfo, setBlueprintInfo] = useState<{
    manufacturerName: string;
    modelName: string;
  } | null>(null);

  useEffect(() => {
    const deviceModelId = endpointDetails?.deviceModelId;
    if (!deviceModelId) {
      setBlueprintInfo(null);
      return;
    }
    let cancelled = false;
    (async () => {
      try {
        const model = await api.getDeviceModel(deviceModelId);
        if (cancelled || !model) return;
        const mfg = await api.getManufacturer(model.manufacturerId);
        if (cancelled) return;
        setBlueprintInfo({
          manufacturerName: mfg?.name || deviceModelId,
          modelName: model.name,
        });
      } catch {
        if (!cancelled) {
          setBlueprintInfo({
            manufacturerName: deviceModelId,
            modelName: deviceModelId,
          });
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [endpointDetails?.deviceModelId]);

  const handleAttach = async () => {
    try {
      await attachMutation.mutateAsync(endPoint.epEui);
      // React Query cache invalidation handles refresh
    } catch (err) {
      logger.error(LOG_ATTACH_FAILED, err);
      setActionError(ERR_ATTACH_ENDPOINT);
    }
  };

  const handleDetach = async () => {
    try {
      await detachMutation.mutateAsync(endPoint.epEui);
      // React Query cache invalidation handles refresh
    } catch (err) {
      logger.error(LOG_DETACH_FAILED, err);
      setActionError(ERR_DETACH_ENDPOINT);
    }
  };

  const handleEdit = () => {
    setEditDialogOpen(true);
  };

  const handleEditClose = () => {
    setEditDialogOpen(false);
  };

  const handleDelete = async () => {
    // Close dialog FIRST to ensure error is visible
    setDeleteDialogOpen(false);

    try {
      // Pre-delete detach if attached (handle detach failure explicitly)
      if (endpointData.attachStatus === "attached") {
        try {
          await detachMutation.mutateAsync(endPoint.epEui);
        } catch (detachErr) {
          // Log but continue with delete - backend may handle attached endpoints
          logger.warn(LOG_PRE_DELETE_DETACH_FAILED, detachErr);
        }
      }

      // Delete endpoint (uses EUI, matches backend route /endpoints/{eui})
      await deleteMutation.mutateAsync(endPoint.epEui);

      // Navigate away on success
      if (onDelete) {
        onDelete(endPoint.epEui);
      } else {
        // Fallback: navigate to endpoints list when no callback provided
        navigate(ROUTES.ENDPOINTS);
      }
    } catch (err) {
      logger.error(LOG_DELETE_FAILED, err);
      // Error visible because dialog is already closed
      setActionError(ERR_DELETE_ENDPOINT);
    }
  };

  return (
    <>
      <Card sx={{ mt: 2, mb: 2 }}>
        <CardContent>
          <Box
            display="flex"
            justifyContent="flex-end"
            alignItems="center"
            mb={2}
          >
            <Box>
              <Tooltip title={ENDPOINT_DETAILS.TOOLTIP_EDIT}>
                <IconButton size="small" onClick={handleEdit}>
                  <EditIcon />
                </IconButton>
              </Tooltip>
              <Tooltip title={ENDPOINT_DETAILS.TOOLTIP_DELETE}>
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

          <Grid container spacing={3}>
            <Grid size={{ xs: 12, md: 6 }}>
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
                  {endPoint.epEui}
                </Typography>
              </Box>
            </Grid>

            <Grid size={{ xs: 12, md: 6 }}>
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
                  <Chip
                    label={
                      endpointData.attachStatus || ENDPOINT_DETAILS.TERM_UNKNOWN
                    }
                    color={
                      endpointData.attachStatus === "attached"
                        ? "success"
                        : endpointData.attachStatus === "detached"
                          ? "warning"
                          : "default"
                    }
                    size="small"
                  />
                </Box>
              </Box>
              {/* Display roaming status. */}
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
                {endpointData.attachStatus !== "attached" ? (
                  <Button
                    variant="contained"
                    color="primary"
                    startIcon={<LinkIcon />}
                    onClick={handleAttach}
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
                    onClick={handleDetach}
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

            {/* Blueprint Configuration (read-only) */}
            {blueprintInfo && (
              <Grid size={12}>
                <Typography
                  variant="subtitle2"
                  color="text.secondary"
                  gutterBottom
                  sx={{
                    mt: 1,
                    display: "flex",
                    alignItems: "center",
                    gap: 0.5,
                  }}
                >
                  <BlueprintIcon fontSize="small" />
                  {ENDPOINT_DETAILS.SECTION_BLUEPRINT_INFO}
                </Typography>
                <Box display="flex" gap={4}>
                  <Box>
                    <Typography variant="body2" color="text.secondary">
                      {ENDPOINT_DETAILS.OPTION_SELECT_MANUFACTURER}
                    </Typography>
                    <Typography variant="body1">
                      {blueprintInfo.manufacturerName}
                    </Typography>
                  </Box>
                  <Box>
                    <Typography variant="body2" color="text.secondary">
                      {ENDPOINT_DETAILS.OPTION_SELECT_MODEL}
                    </Typography>
                    <Typography variant="body1">
                      {blueprintInfo.modelName}
                    </Typography>
                  </Box>
                </Box>
              </Grid>
            )}

            <Grid size={12}>
              <Divider sx={{ my: 1 }} />
            </Grid>

            <Grid size={12}>
              <Box sx={{ borderBottom: 1, borderColor: "divider" }}>
                <Tabs
                  value={activeTab}
                  onChange={(_, v) => setActiveTab(v)}
                  aria-label={ENDPOINT_DETAILS.ARIA_ENDPOINT_TABS}
                >
                  <Tab
                    icon={<MessageIcon />}
                    iconPosition="start"
                    label={ENDPOINT_DETAILS.TAB_MESSAGES}
                  />
                  <Tab
                    icon={<SendIcon />}
                    iconPosition="start"
                    label={ENDPOINT_DETAILS.TAB_DOWNLINK}
                  />
                </Tabs>
              </Box>
            </Grid>

            {activeTab === 0 && (
              <Grid size={12}>
                <Box sx={{ pt: 2 }}>
                  {error && (
                    <Alert severity="error" sx={{ mb: 2 }}>
                      {error}
                    </Alert>
                  )}

                  <ActivityTimeline
                    variant="detailed"
                    items={activityData?.items ?? []}
                    loading={activityLoading}
                  />
                  <PaginationControls
                    page={activityPage}
                    rowsPerPage={activityRowsPerPage}
                    totalCount={activityData?.totalCount ?? 0}
                    onPageChange={(newPage) => setActivityPage(newPage)}
                    onRowsPerPageChange={(newSize) => {
                      setActivityRowsPerPage(newSize);
                      setActivityPage(0);
                      setActivityPageTokens([""]);
                    }}
                  />
                </Box>
              </Grid>
            )}

            {activeTab === 1 && (
              <Grid size={12}>
                <Box sx={{ pt: 2 }}>
                  <DownlinkTab epEui={endPoint.epEui} />
                </Box>
              </Grid>
            )}
          </Grid>
        </CardContent>
      </Card>

      {/* Delete Confirmation Dialog */}
      <Dialog
        open={deleteDialogOpen}
        onClose={() => setDeleteDialogOpen(false)}
      >
        <DialogTitle>{ENDPOINT_DETAILS.DIALOG_DELETE_TITLE}</DialogTitle>
        <DialogContent>
          <DialogContentText>
            {ENDPOINT_DETAILS.DIALOG_DELETE_CONFIRM_PREFIX} (
            {endPoint.name || endPoint.epEui})?
            {endpointData.attachStatus === "attached" && (
              <>
                <br />
                <br />
                <strong>
                  {ENDPOINT_DETAILS.DIALOG_DELETE_NOTE_PREFIX}
                </strong>{" "}
                {ENDPOINT_DETAILS.DIALOG_DELETE_NOTE}
              </>
            )}
            <br />
            <br />
            {ENDPOINT_DETAILS.DIALOG_DELETE_WARNING}
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteDialogOpen(false)} color="primary">
            {ACTION_CANCEL}
          </Button>
          <Button
            onClick={handleDelete}
            color="error"
            variant="contained"
            disabled={isDeleting}
          >
            {isDeleting ? ENDPOINT_DETAILS.ACTION_DELETING : ACTION_DELETE}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Edit Endpoint Dialog - Full MIOTY fields */}
      <EditEndPointDialog
        open={editDialogOpen}
        onClose={handleEditClose}
        endpoint={endpointDetails ?? null}
      />
    </>
  );
};

export default EndPointDetails;
