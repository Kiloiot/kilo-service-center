/**
 * Filter fields of the Events Log, Audit Log and Errors Center
 * (NAV_STRUCTURE §5).
 */

import type { ErrorGroupFilter, EventLogFilter } from "@api-types/api";

import {
  BS_EUI_FIELD,
  EP_EUI_FIELD,
  OP_ID_FIELD,
  timeRangeField,
} from "@components/common/filters/fields";
import type {
  FilterField,
  SelectFilterField,
  TextFilterField,
} from "@components/common/filters/types";
import {
  BOUNDED_TIME_RANGES,
  EVENT_CATEGORY,
  EVENT_OPERATION_TYPES,
  EVENT_OUTCOME,
  type EventOperation,
  type EventOutcome,
  TIME_RANGES,
} from "@constants/app";
import {
  EVENT_CATEGORY_LABELS,
  EVENT_OPERATION_LABELS,
  EVENT_OUTCOME_LABELS,
  FILTER_LABELS,
} from "@constants/messages";

const OPERATIONS = Object.keys(EVENT_OPERATION_TYPES).filter(
  (key): key is EventOperation => key in EVENT_OPERATION_LABELS,
);

const SEARCH_FIELD: TextFilterField<EventLogFilter> = {
  id: "search",
  kind: "text",
  label: FILTER_LABELS.SEARCH,
  get: (filter) => filter.search,
  set: (search) => ({ search }),
};

const OPERATION_FIELD: SelectFilterField<EventLogFilter, EventOperation> = {
  id: "operation",
  kind: "select",
  label: FILTER_LABELS.OPERATION,
  anyLabel: FILTER_LABELS.ANY,
  options: OPERATIONS.map((operation) => ({
    value: operation,
    label: EVENT_OPERATION_LABELS[operation],
  })),
  get: (filter) => filter.operation,
  set: (operation) => ({ operation }),
};

const OUTCOME_FIELD: SelectFilterField<EventLogFilter, EventOutcome> = {
  id: "outcome",
  kind: "select",
  label: FILTER_LABELS.OUTCOME,
  anyLabel: FILTER_LABELS.ANY,
  options: Object.values(EVENT_OUTCOME).map((outcome) => ({
    value: outcome,
    label: EVENT_OUTCOME_LABELS[outcome],
  })),
  get: (filter) => filter.outcome,
  set: (outcome) => ({ outcome }),
};

export const EVENT_FIELDS: readonly FilterField<EventLogFilter>[] = [
  timeRangeField(TIME_RANGES),
  OPERATION_FIELD,
  {
    id: "category",
    kind: "select",
    label: FILTER_LABELS.CATEGORY,
    anyLabel: FILTER_LABELS.ANY,
    options: Object.values(EVENT_CATEGORY).map((category) => ({
      value: category,
      label: EVENT_CATEGORY_LABELS[category],
    })),
    get: (filter) => filter.category,
    set: (category) => ({ category }),
  },
  OUTCOME_FIELD,
  OP_ID_FIELD,
  EP_EUI_FIELD,
  BS_EUI_FIELD,
  SEARCH_FIELD,
];

export const AUDIT_FIELDS: readonly FilterField<EventLogFilter>[] = [
  timeRangeField(TIME_RANGES),
  SEARCH_FIELD,
];

export const ERROR_GROUP_FIELDS: readonly FilterField<ErrorGroupFilter>[] = [
  timeRangeField(BOUNDED_TIME_RANGES),
];
