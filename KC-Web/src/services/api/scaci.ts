/**
 * SCACI control plane in the shapes the UI consumes.
 */

import type { ListPage, PageRequest } from "@api-types/pagination";
import type { ScaciSessionUI, ScaciStatusUI } from "@api-types/system";
import { mapListPage, mapScaciSession, mapScaciStatus } from "@mappers";

import * as scaciRpc from "@services/grpc/rpc/scaci";

export const scaciApi = {
  async getStatus(): Promise<ScaciStatusUI | null> {
    const status = await scaciRpc.getScaciStatus();
    return status ? mapScaciStatus(status) : null;
  },

  async listSessions(page: PageRequest): Promise<ListPage<ScaciSessionUI>> {
    return mapListPage(await scaciRpc.listScaciSessions(page), mapScaciSession);
  },
};
