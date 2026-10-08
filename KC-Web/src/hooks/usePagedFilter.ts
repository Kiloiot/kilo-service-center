/**
 * View state of a server-paged, filtered listing: changing the filter or the
 * page size returns to the first page.
 */

import { useCallback, useState } from "react";

import { PAGINATION } from "@constants/app";

export function usePagedFilter<F>(initial: F) {
  const [filter, setFilter] = useState<F>(initial);
  const [page, setPage] = useState(0);
  const [pageSize, setPageSizeState] = useState<number>(
    PAGINATION.DEFAULT_PAGE_SIZE,
  );

  const updateFilter = useCallback((patch: Partial<F>) => {
    setFilter((prev) => ({ ...prev, ...patch }));
    setPage(0);
  }, []);

  const setPageSize = useCallback((size: number) => {
    setPageSizeState(size);
    setPage(0);
  }, []);

  return {
    filter,
    updateFilter,
    paging: {
      page,
      pageSize,
      onPageChange: setPage,
      onPageSizeChange: setPageSize,
    },
  };
}
