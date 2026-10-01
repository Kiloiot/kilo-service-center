/**
 * Downlink Queue (SCACI §3.10.1 dlDataQue), tenant-wide or for one endpoint:
 * the in-flight downlinks with edit, revoke and flush actions.
 */

import type { DeviceScope, DownlinkQueueFilter } from "@api-types/api";
import { useDownlinkQueue } from "@hooks";
import { Box, Button } from "@mui/material";
import { ConfirmDialog } from "@ui";

import { DataTable } from "@components/common/DataTable";
import { FilterBar } from "@components/common/filters/FilterBar";
import { usePagedFilter } from "@hooks/usePagedFilter";
import { filterMatchesRevocable } from "@utils/downlink-queue";
import { omitFixed } from "@utils/fixed-filter";
import {
  ACTION_FLUSH_QUEUE,
  ACTION_FLUSHING_QUEUE,
  ACTION_REVOKE,
  ACTION_REVOKING,
  CONFIRM_FLUSH_QUEUE,
  CONFIRM_REVOKE_DOWNLINK,
  DOWNLINK_TABLE,
  TITLE_FLUSH_QUEUE,
  TITLE_REVOKE_DOWNLINK,
} from "@constants/messages";

import { useDownlinkQueueActions } from "../hooks";
import { downlinkQueueColumns } from "./downlinkColumns";
import { EditDownlinkDialog } from "./EditDownlinkDialog";
import { DOWNLINK_QUEUE_FIELDS } from "./trafficFields";

type QueueActions = ReturnType<typeof useDownlinkQueueActions>;

interface QueueToolbarProps {
  scope: DeviceScope;
  filter: DownlinkQueueFilter;
  onChange: (patch: Partial<DownlinkQueueFilter>) => void;
  canFlush: boolean;
  onFlush: () => void;
}

function DownlinkQueueToolbar(props: QueueToolbarProps) {
  return (
    <Box sx={{ display: "flex", alignItems: "flex-start", gap: 2 }}>
      <Box sx={{ flex: 1 }}>
        <FilterBar
          fields={omitFixed(DOWNLINK_QUEUE_FIELDS, props.scope)}
          filter={props.filter}
          onChange={props.onChange}
        />
      </Box>
      <Button
        size="small"
        color="error"
        variant="outlined"
        onClick={props.onFlush}
        disabled={!props.canFlush}
      >
        {ACTION_FLUSH_QUEUE}
      </Button>
    </Box>
  );
}

function DownlinkQueueDialogs({ actions }: { actions: QueueActions }) {
  return (
    <>
      {actions.editing && (
        <EditDownlinkDialog
          key={actions.editing.queId}
          downlink={actions.editing}
          onClose={actions.cancelEdit}
          onSaved={actions.saved}
        />
      )}
      <ConfirmDialog
        open={!!actions.revoking}
        onClose={actions.cancelRevoke}
        onConfirm={actions.confirmRevoke}
        title={TITLE_REVOKE_DOWNLINK}
        message={CONFIRM_REVOKE_DOWNLINK}
        confirmLabel={ACTION_REVOKE}
        pendingLabel={ACTION_REVOKING}
      />
      <ConfirmDialog
        open={actions.flushing}
        onClose={actions.cancelFlush}
        onConfirm={actions.confirmFlush}
        title={TITLE_FLUSH_QUEUE}
        message={CONFIRM_FLUSH_QUEUE}
        confirmLabel={ACTION_FLUSH_QUEUE}
        pendingLabel={ACTION_FLUSHING_QUEUE}
      />
    </>
  );
}

export function DownlinkQueuePanel({ scope }: { scope: DeviceScope }) {
  const { filter, updateFilter, paging } = usePagedFilter<DownlinkQueueFilter>(
    {},
  );
  const query = { ...filter, ...scope };
  const { data, isLoading, error } = useDownlinkQueue(
    query,
    paging.page,
    paging.pageSize,
  );
  const actions = useDownlinkQueueActions(query);
  const columns = omitFixed(
    downlinkQueueColumns({
      onEdit: actions.startEdit,
      onRevoke: actions.startRevoke,
    }),
    scope,
  );

  return (
    <Box>
      <DownlinkQueueToolbar
        scope={scope}
        filter={filter}
        onChange={updateFilter}
        canFlush={
          !!data && data.totalCount > 0 && filterMatchesRevocable(query)
        }
        onFlush={actions.startFlush}
      />
      <DataTable
        columns={columns}
        rows={data?.items ?? []}
        rowKey={(downlink) => downlink.queId}
        isLoading={isLoading}
        error={error}
        emptyMessage={DOWNLINK_TABLE.EMPTY_QUEUE}
        paging={paging}
        totalCount={data?.totalCount ?? 0}
      />
      <DownlinkQueueDialogs actions={actions} />
    </Box>
  );
}
