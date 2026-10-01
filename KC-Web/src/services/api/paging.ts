/**
 * Offset paging shared by the listing RPCs: their page token is the offset
 * of the first row, so any page is addressable by number.
 */

import type { ListResult, Page } from "@api-types/api";

import { normalizeEui } from "@utils/eui";

/** Page token of a 0-based page; the first page needs none. */
export function offsetPageToken(
  page: number,
  pageSize: number,
): string | undefined {
  return page > 0 ? String(page * pageSize) : undefined;
}

export function toPage<T>(result: ListResult<T>): Page<T> {
  return { items: result.items, totalCount: result.totalCount };
}

/** EUI filters travel in the canonical form the Service Center compares against. */
export function euiParam(eui: string | undefined): string | undefined {
  return eui ? normalizeEui(eui) : undefined;
}
