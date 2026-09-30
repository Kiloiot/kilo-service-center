/**
 * Base Station Operation Mappers
 */

import type {
  BaseStationAvailabilityDTO,
  BaseStationAvailabilityUI,
} from "@api-types/base-station-operations";

import { MS_PER_SECOND } from "@constants/app";

/** Buckets are UTC-aligned, so the first one starts at the window start floored to the bucket width. */
export function mapBaseStationAvailability(
  dto: BaseStationAvailabilityDTO,
  windowStart: Date,
): BaseStationAvailabilityUI {
  const bucketMs = dto.intervalSeconds * MS_PER_SECOND;
  const firstBucket = Math.floor(windowStart.getTime() / bucketMs) * bucketMs;
  const buckets = dto.availability.map((availability, index) => ({
    start: new Date(firstBucket + index * bucketMs).toISOString(),
    availability,
  }));
  const overall =
    buckets.length > 0
      ? buckets.reduce((sum, bucket) => sum + bucket.availability, 0) /
        buckets.length
      : 0;
  return { buckets, overall };
}
