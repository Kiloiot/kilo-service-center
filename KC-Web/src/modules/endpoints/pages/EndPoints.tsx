import React, { useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import { useEndpoint } from "@hooks";
import {
  Alert,
  Box,
  Button,
  CircularProgress,
  Typography,
} from "@mui/material";

import { BackButton } from "@components/common/BackButton";
import { DetailNotFound } from "@components/common/DetailNotFound";
import { PaginationControls } from "@components/common/PaginationControls";
import { getErrorMessage } from "@utils/error-message";
import { ROUTES, SORT_DIRECTION } from "@constants/app";
import { ENDPOINTS_PAGE, ERR_LOAD_ENDPOINTS } from "@constants/messages";
import { endpointDetailPath } from "@router/paths";
import { AddIcon } from "@theme/icons";
import { componentSpacing } from "@theme/index";

import AddEndPointDialog from "../components/AddEndPointDialog";
import EndPointDetails from "../components/EndPointDetails";
import { EndpointsFiltersBar } from "../components/EndpointsFiltersBar";
import { EndpointsListScope } from "../components/EndpointsListScope";
import { EndpointsStatsCards } from "../components/EndpointsStatsCards";
import {
  type EndpointsOrderBy,
  EndpointsTable,
} from "../components/EndpointsTable";
import { useEndpointList } from "../hooks";

const Centered: React.FC<{ children: React.ReactNode }> = ({ children }) => (
  <Box
    sx={{
      display: "flex",
      justifyContent: "center",
      alignItems: "center",
      minHeight: componentSpacing.stateView.listMinHeight,
    }}
  >
    {children}
  </Box>
);

const EndpointDetailView: React.FC<{
  epEui: string;
  onBack: () => void;
}> = ({ epEui, onBack }) => {
  const list = useEndpointList();
  const { data: detail, isLoading } = useEndpoint(epEui);
  const listed = list.all.find((ep) => ep.epEui === epEui);
  if (isLoading || (!detail && list.isLoading)) {
    return (
      <Centered>
        <CircularProgress />
      </Centered>
    );
  }
  const endpoint = detail || listed;
  if (!endpoint) {
    return (
      <DetailNotFound
        message={ENDPOINTS_PAGE.ERR_NOT_FOUND}
        backLabel={ENDPOINTS_PAGE.BACK_TO_LIST}
        onBack={onBack}
      />
    );
  }
  return (
    <Box sx={{ pt: 4 }}>
      <Box
        display="flex"
        justifyContent="space-between"
        alignItems="center"
        mb={3}
      >
        <Typography variant="h4" component="h1">
          {ENDPOINTS_PAGE.DETAILS_TITLE}
        </Typography>
        <BackButton label={ENDPOINTS_PAGE.BACK_TO_LIST} onClick={onBack} />
      </Box>
      <EndPointDetails endPoint={endpoint} onDelete={onBack} />
    </Box>
  );
};

const EndpointListView: React.FC<{ onOpen: (epEui: string) => void }> = ({
  onOpen,
}) => {
  const list = useEndpointList();
  const { filters } = list;
  const [addDialogOpen, setAddDialogOpen] = useState(false);

  const handleSort = (field: EndpointsOrderBy) => {
    const isAsc =
      filters.sort.field === field &&
      filters.sort.direction === SORT_DIRECTION.ASC;
    list.setSort({
      field,
      direction: isAsc ? SORT_DIRECTION.DESC : SORT_DIRECTION.ASC,
    });
  };

  const narrowed =
    !!filters.search ||
    filters.attachState.length > 0 ||
    filters.activity.length > 0;

  return (
    <Box data-testid="endpoints-page" sx={{ p: 3, pt: 4 }}>
      <Box
        display="flex"
        justifyContent="space-between"
        alignItems="center"
        mb={3}
      >
        <Typography variant="h4" component="h1">
          {ENDPOINTS_PAGE.TITLE}
        </Typography>
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={() => setAddDialogOpen(true)}
        >
          {ENDPOINTS_PAGE.ADD_ENDPOINT}
        </Button>
      </Box>

      <EndpointsStatsCards
        total={list.all.length}
        activeCount={list.activeCount}
      />

      <EndpointsFiltersBar
        search={filters.search}
        attachState={filters.attachState}
        activity={filters.activity}
        onSearchChange={list.setSearch}
        onAttachStateChange={(values) =>
          list.updateFilter("attachState", values)
        }
        onActivityChange={(values) => list.updateFilter("activity", values)}
      />

      <EndpointsListScope
        shown={list.shown.length}
        total={list.all.length}
        search={filters.search}
        attachState={filters.attachState}
        activity={filters.activity}
        onClear={list.reset}
      />

      {list.isLoading && (
        <Centered>
          <CircularProgress />
        </Centered>
      )}

      {list.isError && (
        <Alert severity="error" sx={{ mb: 3 }}>
          {getErrorMessage(list.error, ERR_LOAD_ENDPOINTS)}
        </Alert>
      )}

      {!list.isLoading && !list.isError && (
        <>
          <EndpointsTable
            endpoints={list.page}
            emptyMessage={
              narrowed ? ENDPOINTS_PAGE.NO_MATCH : ENDPOINTS_PAGE.NO_ENDPOINTS
            }
            orderBy={filters.sort.field}
            orderDirection={filters.sort.direction}
            onSort={handleSort}
            onRowClick={onOpen}
          />
          <PaginationControls
            page={filters.pagination.page}
            rowsPerPage={filters.pagination.pageSize}
            totalCount={list.shown.length}
            onPageChange={list.setPage}
            onRowsPerPageChange={list.setPageSize}
          />
        </>
      )}

      <AddEndPointDialog
        open={addDialogOpen}
        onClose={() => setAddDialogOpen(false)}
      />
    </Box>
  );
};

const EndPoints: React.FC = () => {
  const navigate = useNavigate();
  const { id } = useParams();

  if (!id) {
    return (
      <EndpointListView
        onOpen={(epEui) => navigate(endpointDetailPath(epEui))}
      />
    );
  }
  return (
    <EndpointDetailView epEui={id} onBack={() => navigate(ROUTES.ENDPOINTS)} />
  );
};

export default EndPoints;
