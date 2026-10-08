/**
 * Conversions between protobuf wire shapes and the plain values the client
 * methods build requests from and read responses into.
 */

import type { ListPage, PageRequest } from "@api-types/pagination";
import * as google_protobuf_timestamp_pb from "google-protobuf/google/protobuf/timestamp_pb";

const textEncoder = new TextEncoder();
const textDecoder = new TextDecoder();

export function stringToBytes(value: string): Uint8Array {
  return textEncoder.encode(value);
}

export function bytesToString(value: Uint8Array): string {
  return textDecoder.decode(value);
}

export function dateToTimestamp(
  date: Date,
): google_protobuf_timestamp_pb.Timestamp {
  const timestamp = new google_protobuf_timestamp_pb.Timestamp();
  timestamp.fromDate(date);
  return timestamp;
}

/** Pagination and time-range filter params shared across listing methods. */
export interface PaginationTimeRangeParams {
  pageSize?: number;
  pageToken?: string;
  startTime?: Date;
  endTime?: Date;
}

/**
 * Applies pagination (pageSize, pageToken) and time-range (startTime, endTime)
 * fields to a protobuf request that exposes the standard setters.
 */
export function applyPaginationAndTimeRange(
  request: {
    setPageSize(v: number): void;
    setPageToken(v: string): void;
    setStartTime(v: google_protobuf_timestamp_pb.Timestamp): void;
    setEndTime(v: google_protobuf_timestamp_pb.Timestamp): void;
  },
  params?: PaginationTimeRangeParams,
): void {
  if (params?.pageSize) request.setPageSize(params.pageSize);
  if (params?.pageToken) request.setPageToken(params.pageToken);
  if (params?.startTime)
    request.setStartTime(dateToTimestamp(params.startTime));
  if (params?.endTime) request.setEndTime(dateToTimestamp(params.endTime));
}

/**
 * The page token of an offset-paginated list: the backend reads it as the
 * row offset, and the first page sends none.
 */
export function offsetPageToken(page: PageRequest): string {
  const offset = page.page * page.pageSize;
  return offset > 0 ? String(offset) : "";
}

/** Wraps one page of a list response. */
export function pageOf<T>(
  items: T[],
  totalCount: number,
  nextPageToken: string,
): ListPage<T> {
  return { items, totalCount, nextPageToken: nextPageToken || undefined };
}

/** Reads a protobuf string map into a plain record. */
export function tagsMapToRecord(map: {
  toObject(): Array<[string, string]>;
}): Record<string, string> {
  return map.toObject().reduce(
    (acc, [k, v]) => {
      acc[k] = v;
      return acc;
    },
    {} as Record<string, string>,
  );
}

// A 64-bit field generated with jstype=JS_STRING reads "0" when it was never set.
const UNSET_INT64_STRING = "0";

/** Reads a string-typed 64-bit field, treating the unset value as absent. */
export function optionalInt64String(value: string): string | undefined {
  return value === UNSET_INT64_STRING ? undefined : value;
}

/** Unwraps an optional protobuf wrapper (DoubleValue, Int64Value, ...). */
export function wrapperValue<T>(
  wrapper: { getValue(): T } | undefined,
): T | undefined {
  return wrapper?.getValue() ?? undefined;
}
