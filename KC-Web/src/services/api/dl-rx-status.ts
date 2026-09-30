/**
 * DL RX status of an end point in the shapes the UI consumes.
 */

import type {
  DlRxStatusQueriesResponse,
  DlRxStatusResponse,
  QueryDlRxStatusResponse,
} from "@api-types/api";

import * as dlRxStatusRpc from "@services/grpc/rpc/dl-rx-status";

export const dlRxStatusApi = {
  async getStatuses(epEui: string, limit: number): Promise<DlRxStatusResponse> {
    return dlRxStatusRpc.getDlRxStatus({ epEui, limit });
  },

  async getQueries(
    epEui: string,
    limit: number,
  ): Promise<DlRxStatusQueriesResponse> {
    return dlRxStatusRpc.getDlRxStatusQueries({ epEui, limit });
  },

  async query(epEui: string): Promise<QueryDlRxStatusResponse> {
    return dlRxStatusRpc.queryDlRxStatus(epEui);
  },
};
