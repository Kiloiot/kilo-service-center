import React, { useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import {
  Alert,
  Box,
  CircularProgress,
  InputAdornment,
  TextField,
  Typography,
  useTheme,
} from "@mui/material";

import { BackButton } from "@components/common/BackButton";
import { DetailNotFound } from "@components/common/DetailNotFound";
import { PaginationControls } from "@components/common/PaginationControls";
import { useFeedback } from "@contexts/feedback";
import { useCapabilities } from "@hooks/useCapabilities";
import { getErrorMessage } from "@utils/error-message";
import { normalizeEui } from "@utils/eui";
import { ROUTES } from "@constants/app";
import {
  BASE_STATIONS_PAGE,
  ERR_BS_NOT_FOUND,
  ERR_LOAD_BASE_STATIONS,
  ERR_UPDATE_BS_NAME_PARTIAL,
  MSG_BS_DELETED,
  MSG_BS_EUI_UPDATED,
} from "@constants/messages";
import { baseStationDetailPath } from "@router/paths";
import { SearchIcon } from "@theme/icons";
import { componentSpacing } from "@theme/index";

import BaseStationCommissioningDialog from "../components/BaseStationCommissioningDialog";
import BaseStationDetails from "../components/BaseStationDetails";
import BaseStationListHeader from "../components/BaseStationListHeader";
import { BaseStationMap } from "../components/BaseStationMap";
import BaseStationStatsCards from "../components/BaseStationStatsCards";
import BaseStationTable from "../components/BaseStationTable";
import { useBaseStationListView } from "../hooks";

const BaseStations: React.FC = () => {
  const navigate = useNavigate();
  const { id: euiParam } = useParams();
  const theme = useTheme();
  const view = useBaseStationListView();
  const { isServerAdmin } = useCapabilities();
  const [commissioningOpen, setCommissioningOpen] = useState(false);
  const feedback = useFeedback();

  const selectedBaseStation = euiParam
    ? view.stations.find(
        (bs) => normalizeEui(bs.eui) === normalizeEui(euiParam),
      )
    : undefined;

  const openBaseStation = (id: string) => {
    const baseStation = view.stations.find((bs) => bs.id === id);
    if (baseStation) navigate(baseStationDetailPath(baseStation.eui));
  };
  const backToList = () => navigate(ROUTES.BASE_STATIONS);
  const confirmDeletion = () => {
    backToList();
    feedback.success(MSG_BS_DELETED);
  };
  const followEuiChange = (newEui: string, changesSaved: boolean) => {
    navigate(baseStationDetailPath(newEui));
    if (changesSaved) feedback.success(MSG_BS_EUI_UPDATED);
    else feedback.warning(ERR_UPDATE_BS_NAME_PARTIAL);
  };

  const loadingIndicator = (
    <Box
      display="flex"
      justifyContent="center"
      alignItems="center"
      minHeight={componentSpacing.stateView.listMinHeight}
    >
      <CircularProgress />
    </Box>
  );

  if (euiParam && view.isLoading) return loadingIndicator;

  if (euiParam && !selectedBaseStation) {
    return (
      <DetailNotFound
        message={getErrorMessage(view.error, ERR_BS_NOT_FOUND)}
        backLabel={BASE_STATIONS_PAGE.BACK_TO_LIST}
        onBack={backToList}
      />
    );
  }

  if (selectedBaseStation) {
    return (
      <Box sx={{ pt: theme.spacing(4) }}>
        <Box
          display="flex"
          justifyContent="space-between"
          alignItems="center"
          mb={theme.spacing(3)}
        >
          <Typography variant="h4" component="h1">
            {BASE_STATIONS_PAGE.DETAILS_TITLE}
          </Typography>
          <BackButton
            label={BASE_STATIONS_PAGE.BACK_TO_LIST}
            onClick={backToList}
          />
        </Box>
        <BaseStationDetails
          baseStation={selectedBaseStation}
          onDelete={confirmDeletion}
          onEuiChange={followEuiChange}
        />
      </Box>
    );
  }

  return (
    <Box
      data-testid="base-stations-page"
      sx={{ p: theme.spacing(3), pt: theme.spacing(4) }}
    >
      <BaseStationListHeader onAdd={() => setCommissioningOpen(true)} />

      <BaseStationStatsCards
        stats={view.stats}
        overlay={view.overlay}
        onToggleStatus={view.toggleStatusOverlay}
        onToggleCertExpiry={view.toggleCertExpiryOverlay}
      />

      {/* The map reads every tenant's stations, so it is for administrators only */}
      {isServerAdmin && (
        <Box mb={3}>
          <BaseStationMap />
        </Box>
      )}

      <Box mb={3}>
        <TextField
          fullWidth
          variant="outlined"
          placeholder={BASE_STATIONS_PAGE.SEARCH_PLACEHOLDER}
          value={view.filters.search}
          onChange={(e) => view.setSearch(e.target.value)}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon />
              </InputAdornment>
            ),
          }}
        />
      </Box>

      {view.isError && (
        <Alert severity="error" sx={{ mb: 3 }}>
          {getErrorMessage(view.error, ERR_LOAD_BASE_STATIONS)}
        </Alert>
      )}

      {view.isLoading
        ? loadingIndicator
        : !view.isError && (
            <>
              <BaseStationTable
                rows={view.pageRows}
                sort={view.filters.sort}
                columnSortActive={
                  !view.overlay.certExpiry && view.overlay.status === null
                }
                onSort={view.sortByColumn}
                onRowClick={openBaseStation}
              />

              <PaginationControls
                page={view.filters.pagination.page}
                rowsPerPage={view.filters.pagination.pageSize}
                totalCount={view.rows.length}
                onPageChange={view.setPage}
                onRowsPerPageChange={view.setPageSize}
              />
            </>
          )}

      <BaseStationCommissioningDialog
        open={commissioningOpen}
        onClose={() => setCommissioningOpen(false)}
      />
    </Box>
  );
};

export default BaseStations;
