import { useMemo, useState } from "react";

import { useBaseStations, useScopedFilters } from "@hooks";

import { paginate } from "@utils/formatters";
import type { BaseStationStatus } from "@constants/app";
import { SORT_DIRECTION } from "@constants/app";

import {
  arrangeBaseStations,
  type BaseStationSortField,
  getBaseStationStats,
  type StatusOverlay,
} from "../utils/base-station-list";

export function useBaseStationListView() {
  const { filters, setSearch, setSort, setPage, setPageSize } =
    useScopedFilters("baseStations");
  const [certExpiryOverlay, setCertExpiryOverlay] = useState(false);
  const [statusOverlay, setStatusOverlay] = useState<StatusOverlay>(null);
  const { data: stations = [], isLoading, isError, error } = useBaseStations();

  const rows = useMemo(
    () =>
      arrangeBaseStations(stations, filters, {
        certExpiry: certExpiryOverlay,
        status: statusOverlay,
      }),
    [stations, filters, certExpiryOverlay, statusOverlay],
  );
  const pageRows = useMemo(
    () => paginate(rows, filters.pagination.page, filters.pagination.pageSize),
    [rows, filters.pagination.page, filters.pagination.pageSize],
  );
  const stats = useMemo(() => getBaseStationStats(stations), [stations]);

  const sortByColumn = (field: BaseStationSortField) => {
    const isAsc =
      filters.sort.field === field &&
      filters.sort.direction === SORT_DIRECTION.ASC;
    setSort({
      field,
      direction: isAsc ? SORT_DIRECTION.DESC : SORT_DIRECTION.ASC,
    });
    setCertExpiryOverlay(false);
    setStatusOverlay(null);
  };

  const toggleCertExpiryOverlay = () => {
    setCertExpiryOverlay((active) => !active);
    setStatusOverlay(null);
  };

  const toggleStatusOverlay = (status: BaseStationStatus) => {
    setStatusOverlay((current) => (current === status ? null : status));
    setCertExpiryOverlay(false);
  };

  return {
    stations,
    rows,
    pageRows,
    stats,
    filters,
    overlay: { certExpiry: certExpiryOverlay, status: statusOverlay },
    sortByColumn,
    toggleCertExpiryOverlay,
    toggleStatusOverlay,
    setSearch,
    setPage,
    setPageSize,
    isLoading,
    isError,
    error,
  };
}
