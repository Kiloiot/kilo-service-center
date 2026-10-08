import { useState } from "react";

import type { PageRequest } from "@api-types/pagination";

import { ALERT_TABLE_PAGE_SIZE } from "@constants/app";

/** Page and page size of a server-paginated table; a new size restarts at the first page. */
export function usePageState(initialPageSize: number = ALERT_TABLE_PAGE_SIZE) {
  const [request, setRequest] = useState<PageRequest>({
    page: 0,
    pageSize: initialPageSize,
  });
  const setPage = (page: number) =>
    setRequest((current) => ({ ...current, page }));
  const setPageSize = (pageSize: number) => setRequest({ page: 0, pageSize });
  return {
    request,
    resetPage: () => setPage(0),
    paging: {
      ...request,
      onPageChange: setPage,
      onPageSizeChange: setPageSize,
    },
  };
}
