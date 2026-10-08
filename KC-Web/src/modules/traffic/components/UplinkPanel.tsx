/**
 * Uplink Events (SCACI §3.8.1 ulData), tenant-wide or for one device: the
 * filters the scope leaves open, the server-paged rows and each row's
 * receptions on expand. The source names the listing the rows come from.
 */

import type { DeviceScope, UplinkFilter } from "@api-types/api";
import { useUplinks } from "@hooks";
import { Box } from "@mui/material";

import { DataTable } from "@components/common/DataTable";
import { FilterBar } from "@components/common/filters/FilterBar";
import { UplinkDetails } from "@components/common/uplinks/UplinkDetails";
import { usePagedFilter } from "@hooks/usePagedFilter";
import { omitFixed } from "@utils/fixed-filter";
import { TIME_RANGE, UPLINK_SOURCE, type UplinkSource } from "@constants/app";
import { UPLINK_TABLE } from "@constants/messages";

import { UPLINK_FIELDS } from "./trafficFields";
import { UPLINK_COLUMNS } from "./uplinkColumns";

export function UplinkPanel({
  scope,
  source = UPLINK_SOURCE.TENANT,
}: {
  scope: DeviceScope;
  source?: UplinkSource;
}) {
  const { filter, updateFilter, paging } = usePagedFilter<UplinkFilter>({
    timeRange: TIME_RANGE.ALL,
  });
  const { data, isLoading, error } = useUplinks(
    source,
    { ...filter, ...scope },
    paging.page,
    paging.pageSize,
  );

  return (
    <Box>
      <FilterBar
        fields={omitFixed(UPLINK_FIELDS, scope)}
        filter={filter}
        onChange={updateFilter}
      />
      <DataTable
        columns={omitFixed(UPLINK_COLUMNS, scope)}
        rows={data?.items ?? []}
        rowKey={(uplink) => uplink.id}
        isLoading={isLoading}
        error={error}
        emptyMessage={UPLINK_TABLE.EMPTY}
        paging={paging}
        totalCount={data?.totalCount ?? 0}
        renderDetails={(uplink) => <UplinkDetails uplink={uplink} />}
      />
    </Box>
  );
}
