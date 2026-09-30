/**
 * Operator actions on the downlink queue: edit a pending entry, revoke one
 * entry (SCACI §3.11) or revoke every revocable entry the filter matches.
 */

import { useState } from "react";

import type {
  DownlinkQueueFilter,
  SCACIDownlinkQueueDTO,
} from "@api-types/api";
import { useFlushDownlinkQueue, useRevokeDownlink } from "@hooks";

import { useFeedback } from "@contexts/feedback";
import {
  EDIT_DOWNLINK_DIALOG,
  ERR_FLUSH_PARTIAL,
  MSG_DOWNLINK_REVOKED,
  MSG_QUEUE_FLUSH_EMPTY,
  MSG_QUEUE_FLUSHED,
} from "@constants/messages";

export function useDownlinkQueueActions(filter: DownlinkQueueFilter) {
  const [editing, setEditing] = useState<SCACIDownlinkQueueDTO | null>(null);
  const [revoking, setRevoking] = useState<SCACIDownlinkQueueDTO | null>(null);
  const [flushing, setFlushing] = useState(false);
  const revoke = useRevokeDownlink();
  const flush = useFlushDownlinkQueue();
  const feedback = useFeedback();

  const confirmRevoke = async () => {
    if (!revoking) return;
    await revoke.mutateAsync({
      epEui: revoking.epEui,
      queueId: revoking.queId,
    });
    setRevoking(null);
    feedback.success(MSG_DOWNLINK_REVOKED);
  };

  const confirmFlush = async () => {
    const { revoked, failed } = await flush.mutateAsync(filter);
    const withCounts = (text: string) =>
      text
        .replace("{revoked}", String(revoked))
        .replace("{failed}", String(failed));
    if (failed > 0) throw new Error(withCounts(ERR_FLUSH_PARTIAL));
    setFlushing(false);
    feedback.success(
      revoked > 0 ? withCounts(MSG_QUEUE_FLUSHED) : MSG_QUEUE_FLUSH_EMPTY,
    );
  };

  const saved = () => {
    setEditing(null);
    feedback.success(EDIT_DOWNLINK_DIALOG.MSG_UPDATED);
  };

  return {
    editing,
    startEdit: setEditing,
    cancelEdit: () => setEditing(null),
    saved,
    revoking,
    startRevoke: setRevoking,
    cancelRevoke: () => setRevoking(null),
    confirmRevoke,
    flushing,
    startFlush: () => setFlushing(true),
    cancelFlush: () => setFlushing(false),
    confirmFlush,
  };
}
