/**
 * Paging over a forward-only cursor: each page the server returns carries the
 * token of the next one, so the pages reached so far and the page after the
 * current one are the only pages that can be opened. A new page size or a
 * restart (a new filter) returns to the first page.
 */

import { useCallback, useState } from "react";

import type { DataTablePaging } from "@components/common/DataTable";
import { CURSOR_FIRST_PAGE_TOKEN, PAGINATION } from "@constants/app";

const FIRST_PAGE: readonly string[] = [CURSOR_FIRST_PAGE_TOKEN];

export function useCursorPager() {
  const [pageSize, setPageSizeState] = useState<number>(
    PAGINATION.DEFAULT_PAGE_SIZE,
  );
  const [tokens, setTokens] = useState<readonly string[]>(FIRST_PAGE);
  const [page, setPage] = useState(0);

  const restart = useCallback(() => {
    setTokens(FIRST_PAGE);
    setPage(0);
  }, []);

  /** The table paging, given the next-page token of the page on screen. */
  const paging = (nextPageToken: string | undefined): DataTablePaging => {
    const lastKnown = tokens.length - 1;
    const lastReachablePage = nextPageToken
      ? Math.max(lastKnown, page + 1)
      : lastKnown;
    return {
      page,
      pageSize,
      lastReachablePage,
      onPageChange: (target) => {
        if (target > lastReachablePage) return;
        if (target > lastKnown && nextPageToken) {
          setTokens([...tokens, nextPageToken]);
        }
        setPage(target);
      },
      onPageSizeChange: (size) => {
        setPageSizeState(size);
        restart();
      },
    };
  };

  return { pageToken: tokens[page], pageSize, restart, paging };
}
