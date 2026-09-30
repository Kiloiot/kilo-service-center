/**
 * The End Points list as the page shows it: every end point from the server,
 * narrowed by the persisted search and filters, sorted and paged, with the
 * totals the stat cards count from the whole list.
 */

import { useMemo } from "react";

import { useEndpoints, useScopedFilters } from "@hooks";

import { paginate } from "@utils/formatters";
import { filterAndSortEndpoints } from "@utils/list-query";
import { ENDPOINT_ACTIVITY } from "@constants/app";

export function useEndpointList() {
  const scoped = useScopedFilters("endpoints");
  const { search, attachState, activity, sort, pagination } = scoped.filters;
  const { data: all = [], isLoading, isError, error } = useEndpoints();

  const shown = useMemo(
    () => filterAndSortEndpoints(all, { search, attachState, activity, sort }),
    [all, search, attachState, activity, sort],
  );
  const page = useMemo(
    () => paginate(shown, pagination.page, pagination.pageSize),
    [shown, pagination.page, pagination.pageSize],
  );
  const activeCount = useMemo(
    () => all.filter((ep) => ep.status === ENDPOINT_ACTIVITY.ACTIVE).length,
    [all],
  );

  return {
    ...scoped,
    all,
    shown,
    page,
    activeCount,
    isLoading,
    isError,
    error,
  };
}
