import { BACKOFF_FACTOR } from "@constants/app";

/** Delay before retry number `attempt` (0-based): baseMs grown by BACKOFF_FACTOR, capped at maxMs. */
export const exponentialBackoffMs = (
  attempt: number,
  baseMs: number,
  maxMs: number,
): number => Math.min(baseMs * BACKOFF_FACTOR ** attempt, maxMs);
