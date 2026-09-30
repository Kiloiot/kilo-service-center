/**
 * Base station operation shapes: availability buckets and the ping result.
 */

export interface BaseStationAvailabilityDTO {
  availability: number[];
  intervalSeconds: number;
}

export interface AvailabilityBucket {
  start: string;
  /** Online fraction of the bucket, 0..1. */
  availability: number;
}

export interface BaseStationAvailabilityUI {
  buckets: AvailabilityBucket[];
  /** Mean online fraction of the window, 0..1. */
  overall: number;
}

/** BSSCI §5.4: the ping was queued to the station under opId. */
export interface PingResultUI {
  success: boolean;
  message: string;
  opId: string;
}
