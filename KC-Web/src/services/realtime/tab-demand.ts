/**
 * The streams this tab asks for: the context its signed-in user's roles
 * open streams in, and the base stations its pages watch.
 */

import type { StreamContext, TabDemand } from "./stream-demand";

export class TabDemandState {
  private context: StreamContext | null = null;
  private readonly watchers = new Map<string, number>();

  /** Replaces the context; false when it is the one already held. */
  setContext(next: StreamContext | null): boolean {
    const current = this.context;
    if (
      current?.organizationId === next?.organizationId &&
      current?.userId === next?.userId &&
      current?.uplinks === next?.uplinks
    ) {
      return false;
    }
    this.context = next;
    return true;
  }

  watch(bsEui: string): void {
    this.watchers.set(bsEui, (this.watchers.get(bsEui) ?? 0) + 1);
  }

  unwatch(bsEui: string): void {
    const remaining = (this.watchers.get(bsEui) ?? 0) - 1;
    if (remaining > 0) {
      this.watchers.set(bsEui, remaining);
    } else {
      this.watchers.delete(bsEui);
    }
  }

  current(): TabDemand | null {
    if (!this.context) return null;
    return { ...this.context, stations: [...this.watchers.keys()].sort() };
  }
}
