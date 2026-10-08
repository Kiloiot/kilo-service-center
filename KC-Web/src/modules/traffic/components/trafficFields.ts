/**
 * Filter fields of the Uplink Events, Downlink Queue and Downlink Results
 * listings (NAV_STRUCTURE §4).
 */

import type {
  DownlinkQueueFilter,
  DownlinkResultFilter,
  UplinkFilter,
} from "@api-types/api";

import {
  BS_EUI_FIELD,
  EP_EUI_FIELD,
  QUE_ID_FIELD,
  timeRangeField,
  validateNumber,
} from "@components/common/filters/fields";
import type { FilterField } from "@components/common/filters/types";
import {
  DOWNLINK_QUEUE_STATUS,
  DOWNLINK_RESULT,
  TIME_RANGES,
} from "@constants/app";
import {
  DOWNLINK_RESULT_LABELS,
  DOWNLINK_STATUS_LABELS,
  FILTER_LABELS,
  SCACI_FIELD,
  TABLE_HEADERS,
} from "@constants/messages";

export const UPLINK_FIELDS: readonly FilterField<UplinkFilter>[] = [
  timeRangeField(TIME_RANGES),
  EP_EUI_FIELD,
  BS_EUI_FIELD,
  {
    id: "profile",
    kind: "text",
    label: SCACI_FIELD.PROFILE,
    get: (filter) => filter.profile,
    set: (profile) => ({ profile }),
  },
  {
    id: "mode",
    kind: "text",
    label: SCACI_FIELD.MODE,
    get: (filter) => filter.mode,
    set: (mode) => ({ mode }),
  },
  {
    id: "duplicate",
    kind: "toggle",
    label: FILTER_LABELS.DUPLICATES_ONLY,
    get: (filter) => filter.duplicate === true,
    set: (on) => ({ duplicate: on || undefined }),
  },
  {
    id: "dlOpen",
    kind: "toggle",
    label: FILTER_LABELS.DL_OPEN_ONLY,
    get: (filter) => filter.dlOpen === true,
    set: (on) => ({ dlOpen: on || undefined }),
  },
];

export const DOWNLINK_QUEUE_FIELDS: readonly FilterField<DownlinkQueueFilter>[] =
  [
    {
      id: "status",
      kind: "select",
      label: TABLE_HEADERS.COL_STATUS,
      anyLabel: FILTER_LABELS.IN_FLIGHT,
      options: Object.values(DOWNLINK_QUEUE_STATUS).map((status) => ({
        value: status,
        label: DOWNLINK_STATUS_LABELS[status],
      })),
      get: (filter) => filter.status,
      set: (status) => ({ status }),
    },
    {
      id: "priority",
      kind: "text",
      label: SCACI_FIELD.PRIO,
      validate: validateNumber,
      get: (filter) => filter.priority,
      set: (priority) => ({ priority }),
    },
    EP_EUI_FIELD,
    BS_EUI_FIELD,
    QUE_ID_FIELD,
  ];

export const DOWNLINK_RESULT_FIELDS: readonly FilterField<DownlinkResultFilter>[] =
  [
    timeRangeField(TIME_RANGES),
    {
      id: "result",
      kind: "select",
      label: SCACI_FIELD.RESULT,
      anyLabel: FILTER_LABELS.ANY,
      options: Object.values(DOWNLINK_RESULT).map((result) => ({
        value: result,
        label: DOWNLINK_RESULT_LABELS[result],
      })),
      get: (filter) => filter.result,
      set: (result) => ({ result }),
    },
    EP_EUI_FIELD,
    BS_EUI_FIELD,
    QUE_ID_FIELD,
  ];
