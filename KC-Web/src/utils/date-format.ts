/**
 * Date and time formatting on the single MIOTY locale (day-first, 24-hour).
 */

import {
  DATE_INPUT_EU,
  DAYS_PER_WEEK,
  FILE_TIMESTAMP,
  HOURS_PER_DAY,
  MIOTY_LOCALE,
  MS_PER_HOUR,
  MS_PER_MINUTE,
  NANOSECONDS_PER_MILLISECOND,
  SECONDS_PER_DAY,
  SECONDS_PER_HOUR,
  SECONDS_PER_MINUTE,
  TIME_24H_FORMAT,
  TIME_RANGE_LOOKBACK_MS,
  type TimeRange,
} from "@constants/app";
import {
  DURATION_UNIT_SUFFIX,
  RELATIVE_TIME_UNIT,
  UI_COMMON,
} from "@constants/messages";

/** Format a time as HH:mm:ss on the MIOTY locale. */
export const formatTime24h = (date: Date | string): string => {
  const d = typeof date === "string" ? new Date(date) : date;
  return d.toLocaleTimeString(MIOTY_LOCALE, TIME_24H_FORMAT);
};

/** Missing timestamps (an empty string, undefined, an unparsable value) render as "never". */
const toValidDate = (
  date: Date | string | number | undefined | null,
): Date | null => {
  if (date === undefined || date === null || date === "") return null;
  const d = new Date(date);
  return isNaN(d.getTime()) ? null : d;
};

// Full date and time in 24-hour format (DD/MM/YYYY HH:mm:ss)
export const formatDateTime = (
  date: Date | string | number | undefined | null,
): string => {
  const d = toValidDate(date);
  if (!d) return UI_COMMON.TIME_NEVER;
  return `${d.toLocaleDateString(MIOTY_LOCALE)} ${formatTime24h(d)}`;
};

/** Full date and time of a MIOTY Unix-nanosecond timestamp (rxTime, txTime), carried as a string. */
export const formatUnixNanos = (nanos: string | undefined): string => {
  if (!nanos || !/^-?\d+$/.test(nanos)) return formatDateTime(undefined);
  return formatDateTime(
    Number(BigInt(nanos) / BigInt(NANOSECONDS_PER_MILLISECOND)),
  );
};

/** A span of whole seconds as "1d 2h 3m 4s"; the day part only when a day has passed. */
export const formatDurationSeconds = (seconds: number): string => {
  const days = Math.floor(seconds / SECONDS_PER_DAY);
  const hours = Math.floor((seconds % SECONDS_PER_DAY) / SECONDS_PER_HOUR);
  const minutes = Math.floor((seconds % SECONDS_PER_HOUR) / SECONDS_PER_MINUTE);
  const secs = seconds % SECONDS_PER_MINUTE;
  const parts = [
    `${hours}${DURATION_UNIT_SUFFIX.HOUR}`,
    `${minutes}${DURATION_UNIT_SUFFIX.MINUTE}`,
    `${secs}${DURATION_UNIT_SUFFIX.SECOND}`,
  ];
  if (days > 0) parts.unshift(`${days}${DURATION_UNIT_SUFFIX.DAY}`);
  return parts.join(DURATION_UNIT_SUFFIX.SEPARATOR);
};

/** Start of a relative time window; undefined for the unbounded window. */
export const timeRangeStart = (
  range: TimeRange,
  now: Date,
): Date | undefined => {
  const lookback = TIME_RANGE_LOOKBACK_MS[range];
  return lookback === undefined
    ? undefined
    : new Date(now.getTime() - lookback);
};

// Date only (DD/MM/YYYY)
export const formatDate = (
  date: Date | string | number | undefined | null,
): string => {
  const d = toValidDate(date);
  return d ? d.toLocaleDateString(MIOTY_LOCALE) : UI_COMMON.TIME_NEVER;
};

type RelativeTimeUnit =
  (typeof RELATIVE_TIME_UNIT)[keyof typeof RELATIVE_TIME_UNIT];

const timeAgo = (count: number, unit: RelativeTimeUnit): string =>
  `${count} ${count === 1 ? unit.ONE : unit.MANY} ${UI_COMMON.TIME_AGO}`;

/** Relative age of a timestamp: "5 minutes ago", "2 hours ago"; a week or older renders as a date. */
export const formatRelativeDuration = (
  timestamp: string | undefined,
): string => {
  if (!timestamp) return UI_COMMON.TIME_NEVER;

  const timestampDate = new Date(timestamp);
  const now = new Date();
  const diffMs = now.getTime() - timestampDate.getTime();
  const diffMinutes = Math.floor(diffMs / MS_PER_MINUTE);
  const diffHours = Math.floor(diffMs / MS_PER_HOUR);
  const diffDays = Math.floor(diffHours / HOURS_PER_DAY);

  if (diffMinutes < 1) return UI_COMMON.TIME_NOW;
  if (diffHours < 1) return timeAgo(diffMinutes, RELATIVE_TIME_UNIT.MINUTE);
  if (diffHours < HOURS_PER_DAY)
    return timeAgo(diffHours, RELATIVE_TIME_UNIT.HOUR);
  if (diffDays < DAYS_PER_WEEK)
    return timeAgo(diffDays, RELATIVE_TIME_UNIT.DAY);
  return formatDate(timestampDate);
};

/**
 * Parse a typed DD/MM/YYYY date into local midnight of that day; an
 * incomplete text or a day the calendar lacks (31/02) is undefined.
 */
export const parseDateInputEU = (text: string): Date | undefined => {
  const match = text.match(/^(\d{2})\/(\d{2})\/(\d{4})$/);
  if (!match) return undefined;
  const [day, month, year] = match.slice(1).map(Number);
  const date = new Date(year, month - 1, day);
  const sameDay =
    date.getFullYear() === year &&
    date.getMonth() === month - 1 &&
    date.getDate() === day;
  return sameDay ? date : undefined;
};

/** A UTC timestamp safe in a file name on every file system (no colon). */
export const formatFileTimestamp = (date: Date): string =>
  date.toISOString().replace(FILE_TIMESTAMP.UNSAFE, FILE_TIMESTAMP.SEPARATOR);

/** Local midnight of the day after date: the exclusive end of that day. */
export const nextLocalMidnight = (date: Date): Date =>
  new Date(date.getFullYear(), date.getMonth(), date.getDate() + 1);

/** Keystrokes of a DD/MM/YYYY field: digits only, separators inserted. */
export const maskDateInputEU = (text: string): string => {
  const digits = text.replace(/\D/g, "").slice(0, DATE_INPUT_EU.DIGITS);
  return [
    digits.slice(0, DATE_INPUT_EU.DAY_END),
    digits.slice(DATE_INPUT_EU.DAY_END, DATE_INPUT_EU.MONTH_END),
    digits.slice(DATE_INPUT_EU.MONTH_END),
  ]
    .filter(Boolean)
    .join(DATE_INPUT_EU.SEPARATOR);
};
