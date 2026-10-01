/**
 * Errors Center (NAV_STRUCTURE §5): failures only, grouped per bucket by
 * event type, code and related object (ListErrorGroups).
 */

import type { ErrorGroupFilter, ErrorGroupUI } from "@api-types/api";
import { useErrorGroups } from "@hooks";
import { Box } from "@mui/material";

import { DataTable } from "@components/common/DataTable";
import { FilterBar } from "@components/common/filters/FilterBar";
import { ViewTabs } from "@components/common/ViewTabs";
import { usePagedFilter } from "@hooks/usePagedFilter";
import {
  ERROR_BUCKETS,
  type ErrorBucket,
  ROW_KEY_SEPARATOR,
  TIME_RANGE,
} from "@constants/app";
import {
  ERROR_BUCKET_LABELS,
  ERROR_GROUP_TABLE,
  LOGS_PAGE,
} from "@constants/messages";

import { ERROR_GROUP_COLUMNS } from "./logColumns";
import { ERROR_GROUP_FIELDS } from "./logFields";

// SCACI failures group by command, code and error token, so the message tells two groups apart.
const errorGroupKey = (group: ErrorGroupUI) =>
  [group.eventType, group.code, group.sourceName, group.message].join(
    ROW_KEY_SEPARATOR,
  );

function ErrorGroupsPanel({ bucket }: { bucket: ErrorBucket }) {
  const { filter, updateFilter, paging } = usePagedFilter<ErrorGroupFilter>({
    bucket,
    timeRange: TIME_RANGE.LAST_DAY,
  });
  const { data, isLoading, error } = useErrorGroups(
    filter,
    paging.page,
    paging.pageSize,
  );

  return (
    <Box>
      <FilterBar
        fields={ERROR_GROUP_FIELDS}
        filter={filter}
        onChange={updateFilter}
      />
      <DataTable
        columns={ERROR_GROUP_COLUMNS}
        rows={data?.items ?? []}
        rowKey={errorGroupKey}
        isLoading={isLoading}
        error={error}
        emptyMessage={ERROR_GROUP_TABLE.EMPTY}
        paging={paging}
        totalCount={data?.totalCount ?? 0}
      />
    </Box>
  );
}

export function ErrorsCenter() {
  return (
    <ViewTabs
      views={ERROR_BUCKETS}
      labels={ERROR_BUCKET_LABELS}
      ariaLabel={LOGS_PAGE.ARIA_ERROR_BUCKETS}
    >
      {(bucket) => <ErrorGroupsPanel key={bucket} bucket={bucket} />}
    </ViewTabs>
  );
}
