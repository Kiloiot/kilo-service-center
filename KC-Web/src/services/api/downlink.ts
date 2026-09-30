/**
 * Downlink queue and results in the shapes the UI consumes.
 */

import type {
  DownlinkFlushResult,
  DownlinkQueueFilter,
  DownlinkResultFilter,
  Page,
  RevokeDownlinkResponse,
  SCACIDownlinkQueueDTO,
  SendDownlinkRequest,
  SendDownlinkResponse,
  UpdatePendingDownlinkRequest,
} from "@api-types/api";

import * as downlinkRpc from "@services/grpc/rpc/downlink";
import { timeRangeStart } from "@utils/date-format";
import { isRevocableDownlink } from "@utils/downlink-queue";
import { PAGINATION } from "@constants/app";

import { euiParam, offsetPageToken, toPage } from "./paging";

export const downlinkApi = {
  sendDownlink(params: SendDownlinkRequest): Promise<SendDownlinkResponse> {
    return downlinkRpc.sendDownlink(params);
  },

  updatePendingDownlink(
    params: UpdatePendingDownlinkRequest,
  ): Promise<SCACIDownlinkQueueDTO> {
    return downlinkRpc.updatePendingDownlink(params);
  },

  revokeDownlink(
    epEui: string,
    queueId: string,
  ): Promise<RevokeDownlinkResponse> {
    return downlinkRpc.revokeDownlink({ epEui, queueId });
  },

  async listQueue(
    filter: DownlinkQueueFilter,
    page: number,
    pageSize: number,
  ): Promise<Page<SCACIDownlinkQueueDTO>> {
    const result = await downlinkRpc.listDownlinkQueue({
      epEui: euiParam(filter.epEui),
      bsEui: euiParam(filter.bsEui),
      status: filter.status,
      priority:
        filter.priority === undefined ? undefined : Number(filter.priority),
      queId: filter.queId,
      pageSize,
      pageToken: offsetPageToken(page, pageSize),
    });
    return toPage(result);
  },

  /**
   * Revokes every revocable downlink the filter matches: pages through the
   * whole listing first, so revocations cannot shift the pages being read.
   */
  async flushQueue(filter: DownlinkQueueFilter): Promise<DownlinkFlushResult> {
    const pageSize = PAGINATION.DOWNLINK_FLUSH_PAGE_SIZE;
    const revocable: SCACIDownlinkQueueDTO[] = [];
    let seen = 0;
    for (let page = 0; ; page++) {
      const { items, totalCount } = await downlinkApi.listQueue(
        filter,
        page,
        pageSize,
      );
      revocable.push(...items.filter(isRevocableDownlink));
      seen += items.length;
      if (items.length === 0 || seen >= totalCount) break;
    }
    const results = await Promise.allSettled(
      revocable.map((m) => downlinkApi.revokeDownlink(m.epEui, m.queId)),
    );
    return {
      revoked: results.filter((r) => r.status === "fulfilled").length,
      failed: results.filter((r) => r.status === "rejected").length,
    };
  },

  async listResults(
    filter: DownlinkResultFilter,
    page: number,
    pageSize: number,
  ): Promise<Page<SCACIDownlinkQueueDTO>> {
    const result = await downlinkRpc.getDownlinkResults({
      epEui: euiParam(filter.epEui),
      bsEui: euiParam(filter.bsEui),
      result: filter.result,
      queId: filter.queId,
      timeFrom: timeRangeStart(filter.timeRange, new Date()),
      pageSize,
      pageToken: offsetPageToken(page, pageSize),
    });
    return toPage(result);
  },
};
