/**
 * Events, Audit and Errors Center logs in the shapes the UI consumes.
 */

import type {
  ErrorGroupFilter,
  ErrorGroupUI,
  EventLogEntryUI,
  EventLogFilter,
  Page,
} from "@api-types/api";
import { mapEventLogEntry } from "@mappers";

import * as eventsRpc from "@services/grpc/rpc/events";
import { timeRangeStart } from "@utils/date-format";
import { EVENT_OPERATION_TYPES } from "@constants/app";

import { euiParam, offsetPageToken, toPage } from "./paging";

export const logsApi = {
  async listEvents(
    filter: EventLogFilter,
    page: number,
    pageSize: number,
  ): Promise<Page<EventLogEntryUI>> {
    const response = await eventsRpc.listEvents({
      categories: filter.category ? [filter.category] : undefined,
      eventTypes: filter.operation
        ? EVENT_OPERATION_TYPES[filter.operation]
        : undefined,
      outcome: filter.outcome,
      opId: filter.opId,
      epEui: euiParam(filter.epEui),
      bsEui: euiParam(filter.bsEui),
      search: filter.search,
      startTime: timeRangeStart(filter.timeRange, new Date()),
      pageSize,
      pageToken: offsetPageToken(page, pageSize),
    });
    return {
      items: response.events.map(mapEventLogEntry),
      totalCount: response.totalCount,
    };
  },

  async listErrorGroups(
    filter: ErrorGroupFilter,
    page: number,
    pageSize: number,
  ): Promise<Page<ErrorGroupUI>> {
    const result = await eventsRpc.listErrorGroups({
      bucket: filter.bucket,
      startTime: timeRangeStart(filter.timeRange, new Date()),
      pageSize,
      pageToken: offsetPageToken(page, pageSize),
    });
    return toPage(result);
  },
};
