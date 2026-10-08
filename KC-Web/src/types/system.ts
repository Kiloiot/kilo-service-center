/**
 * Service center shapes: the SCACI control plane, look-back windows and the
 * capabilities.
 * *DTO types are what the RPC layer returns; *UI types are what components
 * render.
 */

// SCACI control plane (SCACI §3.3 connect, §3.4 ping)

export interface ScaciStatusDTO {
  serviceOnline: boolean;
  protocolVersion: string;
  scEui: string;
  lastPingAt?: Date;
  lastPingRttMs: number;
  missedPings: number;
  lastConnectResult: string;
}

export interface ScaciStatusUI {
  serviceOnline: boolean;
  protocolVersion: string;
  scEui: string;
  lastPingAt?: string;
  lastPingRttMs?: number;
  missedPings: number;
  lastConnectResult: string;
}

export interface ScaciSessionDTO {
  acEui: string;
  status: string;
  protocolVersion: string;
  snAcUuid: string;
  snScUuid: string;
  lastOpIdAc: string;
  lastOpIdSc: string;
}

export interface ScaciSessionUI {
  acEui: string;
  status: string;
  /** conRsp snResume: the session's latest connect resumed a previous session. */
  snResume: boolean;
  version: string;
  snAcUuid: string;
  snScUuid: string;
  /** Highest AC-issued (positive) opId. */
  maxAcOpId: string;
  /** Lowest SC-issued (negative) opId. */
  minScOpId: string;
}

/** A look-back window ending now. */
export interface AnalyticsWindow {
  startTime: Date;
  endTime: Date;
}

// Capabilities

export interface Capability {
  name: string;
  enabled: boolean;
}
