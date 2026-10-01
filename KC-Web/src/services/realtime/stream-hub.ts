/**
 * The streams of the leader tab: one per distinct stream any tab demands,
 * held while the shared session lasts. Each stream's events, connection
 * notices, errors and state are relayed to every tab.
 */

import { REALTIME_RELAY_MESSAGE } from "@constants/app";

import type { StreamMessage } from "./relay";
import { type ChannelHost, StreamChannel } from "./stream-channel";
import { type StreamSpec, type TabDemand, unionSpecs } from "./stream-demand";
import { type SpecOpener, STREAM_WORDING } from "./stream-openers";

export interface StreamHubDeps {
  open: SpecOpener;
  publish(message: StreamMessage): void;
  /** True while the session every tab shares is signed in. */
  hasSession(): boolean;
}

export class StreamHub {
  private readonly demands = new Map<string, TabDemand>();
  private readonly channels = new Map<string, StreamChannel>();
  private readonly deps: StreamHubDeps;

  constructor(deps: StreamHubDeps) {
    this.deps = deps;
  }

  /** Records a tab's demand, null withdrawing it, and holds the streams every demand needs. */
  setDemand(tabId: string, demand: TabDemand | null): void {
    const known = this.demands.has(tabId);
    if (!demand && !known) return;
    if (demand) {
      this.demands.set(tabId, demand);
    } else {
      this.demands.delete(tabId);
    }
    this.apply();
    // A tab that joins learns which streams were connected before it came.
    if (demand && !known) {
      this.channels.forEach((channel) => channel.announceConnected());
    }
  }

  private apply(): void {
    const wanted = this.deps.hasSession()
      ? unionSpecs(this.demands.values())
      : new Map<string, StreamSpec>();
    this.channels.forEach((channel, key) => {
      if (wanted.has(key)) return;
      this.channels.delete(key);
      channel.close();
    });
    wanted.forEach((spec, key) => {
      if (!this.channels.has(key)) this.openChannel(key, spec);
    });
    this.publishStates();
  }

  private openChannel(key: string, spec: StreamSpec): void {
    const channel = new StreamChannel(
      spec.kind,
      STREAM_WORDING[spec.kind],
      this.hostFor(key),
    );
    this.channels.set(key, channel);
    channel.open((callbacks) =>
      this.deps.open(spec, callbacks, (event) => {
        // A message the closed stream's request still delivers is dropped.
        if (this.channels.get(key) !== channel) return;
        this.deps.publish({
          type: REALTIME_RELAY_MESSAGE.EVENT,
          streamKey: key,
          event,
        });
      }),
    );
  }

  private hostFor(streamKey: string): ChannelHost {
    return {
      hasContext: () => this.deps.hasSession() && this.channels.has(streamKey),
      emit: (event) =>
        this.deps.publish({
          type: REALTIME_RELAY_MESSAGE.CONNECTION,
          streamKey,
          event: { ...event, timestamp: new Date().toISOString() },
        }),
      reportError: (error) =>
        this.deps.publish({
          type: REALTIME_RELAY_MESSAGE.ERROR,
          streamKey,
          error,
        }),
      stateChanged: () => this.publishStates(),
    };
  }

  private publishStates(): void {
    const states = Object.fromEntries(
      [...this.channels].map(([key, channel]) => [key, channel.getState()]),
    );
    this.deps.publish({ type: REALTIME_RELAY_MESSAGE.STATES, states });
  }
}
