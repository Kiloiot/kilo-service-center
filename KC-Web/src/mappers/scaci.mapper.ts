/**
 * SCACI Mappers
 *
 * Control-plane DTOs to UI shapes, with the SCACI connect fields named as
 * the specification names them (SCACI §3.3.1/§3.3.2).
 */

import type {
  ScaciSessionDTO,
  ScaciSessionUI,
  ScaciStatusDTO,
  ScaciStatusUI,
} from "@api-types/system";

import { SCACI_SESSION_STATUS } from "@constants/app";

export function mapScaciStatus(dto: ScaciStatusDTO): ScaciStatusUI {
  return {
    serviceOnline: dto.serviceOnline,
    protocolVersion: dto.protocolVersion,
    scEui: dto.scEui,
    lastPingAt: dto.lastPingAt?.toISOString(),
    lastPingRttMs: dto.lastPingAt ? dto.lastPingRttMs : undefined,
    missedPings: dto.missedPings,
    lastConnectResult: dto.lastConnectResult,
  };
}

export function mapScaciSession(dto: ScaciSessionDTO): ScaciSessionUI {
  return {
    acEui: dto.acEui,
    status: dto.status,
    snResume: dto.status === SCACI_SESSION_STATUS.RESUMED,
    version: dto.protocolVersion,
    snAcUuid: dto.snAcUuid,
    snScUuid: dto.snScUuid,
    maxAcOpId: dto.lastOpIdAc,
    minScOpId: dto.lastOpIdSc,
  };
}
