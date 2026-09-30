import type { AnalyticsWindow } from "@api-types/system";

import { MS_PER_HOUR } from "@constants/app";

/** The window of the given length that ends now; evaluated per fetch so a refetch moves it forward. */
export function windowEndingNow(
  hours: number,
  now: Date = new Date(),
): AnalyticsWindow {
  return {
    startTime: new Date(now.getTime() - hours * MS_PER_HOUR),
    endTime: now,
  };
}
