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

interface EndPointInput {
  id: string;
  epEui: string;
  name?: string;
  lastSeen?: string;
  status: "active" | "inactive";
  attachStatus?: "attached" | "detached" | "attaching" | "pending" | "unknown";
  ownerTenantId?: number;
  isRoaming?: boolean;
}

interface BlueprintInfo {
  manufacturerName: string;
  modelName: string;
}

interface EndPointDetailsProps {
  endPoint: EndPointInput;
  onDelete?: (id: string) => void;
}

interface EndpointOverviewPanelProps {
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

const EndpointOverviewPanel: React.FC<EndpointOverviewPanelProps> = ({
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
              label={endpointData.attachStatus || ENDPOINT_DETAILS.TERM_UNKNOWN}
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

      {blueprintInfo && (
        <Grid size={12}>
          <Typography
            variant="subtitle2"
            color="text.secondary"
            gutterBottom
            sx={{ mt: 1, display: "flex", alignItems: "center", gap: 0.5 }}
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
              <Typography variant="body1">{blueprintInfo.modelName}</Typography>
            </Box>
          </Box>
        </Grid>
      )}
    </Grid>
  </>
);

interface EndpointMessagesTabProps {
  error: string | null;
  items: React.ComponentProps<typeof ActivityTimeline>["items"];
  loading: boolean;
  page: number;
  rowsPerPage: number;
  totalCount: number;
  onPageChange: (page: number) => void;
  onRowsPerPageChange: (size: number) => void;
}

const EndpointMessagesTab: React.FC<EndpointMessagesTabProps> = ({
  error,
  items,
  loading,
  page,
  rowsPerPage,
  totalCount,
  onPageChange,
  onRowsPerPageChange,
}) => (
  <Box sx={{ pt: 2 }}>
    {error && (
      <Alert severity="error" sx={{ mb: 2 }}>
        {error}
      </Alert>
    )}
    <ActivityTimeline variant="detailed" items={items} loading={loading} />
    <PaginationControls
      page={page}
      rowsPerPage={rowsPerPage}
      totalCount={totalCount}
      onPageChange={onPageChange}
      onRowsPerPageChange={onRowsPerPageChange}
    />
  </Box>
);

const EndpointDownlinkTabPanel: React.FC<{ epEui: string }> = ({ epEui }) => (
  <Box sx={{ pt: 2 }}>
    <DownlinkTab epEui={epEui} />
  </Box>
);

const EndPointDetails: React.FC<EndPointDetailsProps> = ({
  endPoint,
  onDelete,
}) => {
  const navigate = useNavigate();
  const [actionError, setActionError] = useState<string | null>(null);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [activeTab, setActiveTab] = useState(0);

  const attachMutation = useAttachEndpoint();
  const detachMutation = useDetachEndpoint();
  const deleteMutation = useDeleteEndpoint();

  const isAttaching = attachMutation.isPending;
  const isDetaching = detachMutation.isPending;
  const isDeleting = deleteMutation.isPending;

  const { data: endpointDetails } = useEndpoint(endPoint.epEui);
  const endpointData = endpointDetails || endPoint;

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

  useEffect(() => {
    if (
      activityData?.nextPageToken &&
      activityPage === activityPageTokens.length - 1
    ) {
      setActivityPageTokens((prev) => [...prev, activityData.nextPageToken!]);
    }
  }, [activityData?.nextPageToken, activityPage, activityPageTokens.length]);

  const [blueprintInfo, setBlueprintInfo] = useState<BlueprintInfo | null>(
    null,
  );

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
    } catch (err) {
      logger.error(LOG_ATTACH_FAILED, err);
      setActionError(ERR_ATTACH_ENDPOINT);
    }
  };

  const handleDetach = async () => {
    try {
      await detachMutation.mutateAsync(endPoint.epEui);
    } catch (err) {
      logger.error(LOG_DETACH_FAILED, err);
      setActionError(ERR_DETACH_ENDPOINT);
    }
  };

  const handleEdit = () => setEditDialogOpen(true);
  const handleEditClose = () => setEditDialogOpen(false);

  const handleDelete = async () => {
    setDeleteDialogOpen(false);
    try {
      if (endpointData.attachStatus === "attached") {
        try {
          await detachMutation.mutateAsync(endPoint.epEui);
        } catch (detachErr) {
          logger.warn(LOG_PRE_DELETE_DETACH_FAILED, detachErr);
        }
      }
      await deleteMutation.mutateAsync(endPoint.epEui);
      if (onDelete) {
        onDelete(endPoint.epEui);
      } else {
        navigate(ROUTES.ENDPOINTS);
      }
    } catch (err) {
      logger.error(LOG_DELETE_FAILED, err);
      setActionError(ERR_DELETE_ENDPOINT);
    }
  };

  return (
    <>
      <Card sx={{ mt: 2, mb: 2 }}>
        <CardContent>
          <EndpointOverviewPanel
            endPoint={endPoint}
            endpointData={endpointData}
            blueprintInfo={blueprintInfo}
            isAttaching={isAttaching}
            isDetaching={isDetaching}
            onAttach={handleAttach}
            onDetach={handleDetach}
            onEdit={handleEdit}
            onDeleteClick={() => setDeleteDialogOpen(true)}
          />

          <Grid container spacing={3}>
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
                <EndpointMessagesTab
                  error={actionError}
                  items={activityData?.items ?? []}
                  loading={activityLoading}
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
              </Grid>
            )}

            {activeTab === 1 && (
              <Grid size={12}>
                <EndpointDownlinkTabPanel epEui={endPoint.epEui} />
              </Grid>
            )}
          </Grid>
        </CardContent>
      </Card>

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

      <EditEndPointDialog
        open={editDialogOpen}
        onClose={handleEditClose}
        endpoint={endpointDetails ?? null}
      />
    </>
  );
};

export default EndPointDetails;
