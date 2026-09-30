/**
 * RealtimeService
 *
 * One tab of the origin, elected through a Web Lock held for its life, owns
 * the gRPC-web server streams and relays everything they carry to every tab
 * over a BroadcastChannel, so the streams do not multiply with the tabs: over
 * HTTP/1.1 each stream holds one of the few connections a browser allows a
 * host. Every tab states its demand, the context its user's roles open
 * streams in and the stations its pages watch, and consumes the relayed
 * streams that demand names. When the leader closes, the next tab takes over.
 */

import type { ExclusiveLock } from "@services/grpc/refreshCoordinator";
import { webExclusiveLock } from "@services/grpc/webLock";
import { normalizeEui } from "@utils/eui";
import { generateRandomKey } from "@utils/formatters";
import { storageService } from "@utils/storage";
import {
  BROADCAST_CHANNEL_NAMES,
  LOCK_NAMES,
  REALTIME_RELAY_MESSAGE,
  REALTIME_TAB_ID_BYTES,
  STORAGE_KEYS,
} from "@constants/app";

import { holdForLife, TabLiveness, tabLockName } from "./leadership";
import {
  broadcastRelayPort,
  type DemandMessage,
  type RelayMessage,
  type RelayPort,
} from "./relay";
import { StreamHub } from "./stream-hub";
import { grpcSpecOpener, type SpecOpener } from "./stream-openers";
import { TabDemandState } from "./tab-demand";
import { TabView } from "./tab-view";
import type {
  ConnectionError,
  ConnectionEventListener,
  ConnectionState,
  ErrorListener,
  EventHandler,
  RealtimeLifecycle,
  RealtimeStreams,
  RealtimeStreamSet,
  RealtimeSubscriptions,
  StateChangeListener,
} from "./types";

interface RealtimeServiceDeps {
  tabId: string;
  port: RelayPort;
  lock: ExclusiveLock;
  open: SpecOpener;
  /** True while the session every tab shares is signed in. */
  hasSession(): boolean;
}

class RealtimeService
  implements RealtimeLifecycle, RealtimeSubscriptions, RealtimeStreams
{
  private readonly view = new TabView();
  private readonly demand = new TabDemandState();
  private readonly liveness: TabLiveness;
  private readonly deps: RealtimeServiceDeps;
  private hub: StreamHub | null = null;

  constructor(deps: RealtimeServiceDeps) {
    this.deps = deps;
    this.liveness = new TabLiveness(deps.lock);
    deps.port.listen((message) => this.fromChannel(message));
    holdForLife(deps.lock, tabLockName(deps.tabId));
    holdForLife(deps.lock, LOCK_NAMES.REALTIME_LEADER, () => this.lead());
  }

  reconnectWithOrg(
    organizationId: string,
    userId: string,
    streams: RealtimeStreamSet,
  ): void {
    const context = { organizationId, userId, uplinks: streams.uplinks };
    if (this.demand.setContext(context)) this.announce();
  }

  reset(): void {
    if (this.demand.setContext(null)) this.announce();
  }

  watchBaseStation(bsEui: string): () => void {
    const eui = normalizeEui(bsEui);
    this.demand.watch(eui);
    this.announce();
    return () => {
      this.demand.unwatch(eui);
      this.announce();
    };
  }

  getState(): ConnectionState {
    return this.view.getState();
  }

  getLastError(): ConnectionError | null {
    return this.view.getLastError();
  }

  subscribeAll(handler: EventHandler): () => void {
    return this.view.subscribeAll(handler);
  }

  onStateChange(listener: StateChangeListener): () => void {
    return this.view.onStateChange(listener);
  }

  onErrorChange(listener: ErrorListener): () => void {
    return this.view.onErrorChange(listener);
  }

  onConnectionEvent(listener: ConnectionEventListener): () => void {
    return this.view.onConnectionEvent(listener);
  }

  private announce(): void {
    const demand = this.demand.current();
    this.view.setDemand(demand);
    if (this.hub) {
      this.hub.setDemand(this.deps.tabId, demand);
      return;
    }
    this.deps.port.post({
      type: REALTIME_RELAY_MESSAGE.DEMAND,
      tabId: this.deps.tabId,
      demand,
    });
  }

  private fromChannel(message: RelayMessage): void {
    const hub = this.hub;
    // A leader consumes only its own streams; without Web Locks every tab leads and serves only itself.
    if (!hub) {
      this.deliver(message);
      return;
    }
    if (message.type === REALTIME_RELAY_MESSAGE.DEMAND)
      this.serve(hub, message);
  }

  private serve(hub: StreamHub, message: DemandMessage): void {
    this.liveness.watch(message.tabId, () =>
      hub.setDemand(message.tabId, null),
    );
    hub.setDemand(message.tabId, message.demand);
  }

  private deliver(message: RelayMessage): void {
    switch (message.type) {
      // Another tab's demand is the leader's to serve.
      case REALTIME_RELAY_MESSAGE.DEMAND:
        return;
      case REALTIME_RELAY_MESSAGE.CENSUS:
        this.view.leaderLost();
        this.announce();
        return;
      default:
        this.view.receive(message);
    }
  }

  private lead(): void {
    this.hub = new StreamHub({
      open: this.deps.open,
      hasSession: () => this.deps.hasSession(),
      publish: (message) => this.publish(message),
    });
    this.publish({ type: REALTIME_RELAY_MESSAGE.CENSUS });
  }

  /** Relays to the other tabs and delivers to this one alike. */
  private publish(message: RelayMessage): void {
    this.deps.port.post(message);
    this.deliver(message);
  }
}

const realtimeService = new RealtimeService({
  tabId: generateRandomKey(REALTIME_TAB_ID_BYTES),
  port: broadcastRelayPort(BROADCAST_CHANNEL_NAMES.REALTIME),
  lock: webExclusiveLock(),
  open: grpcSpecOpener,
  hasSession: () => !!storageService.getItem(STORAGE_KEYS.AUTH_TOKEN),
});

/** Connection lifecycle and state listeners. */
export const realtimeLifecycle: RealtimeLifecycle = realtimeService;

/** Event subscription for cache invalidation. */
export const realtimeSubscriptions: RealtimeSubscriptions = realtimeService;

/** The live updates of the base station pages open. */
export const realtimeStreams: RealtimeStreams = realtimeService;
