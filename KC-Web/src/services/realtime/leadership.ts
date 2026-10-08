/**
 * Locks a tab holds for its whole life. The browser releases them when the
 * tab closes or navigates away, which elects the next leader and tells the
 * leader that a tab is gone.
 */

import type { ExclusiveLock } from "@services/grpc/refreshCoordinator";
import { LOCK_NAMES } from "@constants/app";

/** Holds the lock until the tab ends, calling onHeld once it is granted. */
export function holdForLife(
  lock: ExclusiveLock,
  name: string,
  onHeld?: () => void,
): void {
  void lock.run(name, () => {
    onHeld?.();
    return new Promise<never>(() => {});
  });
}

/** The lock a tab holds for its life. */
export function tabLockName(tabId: string): string {
  return `${LOCK_NAMES.REALTIME_TAB_PREFIX}${tabId}`;
}

/** Tells the leader when a tab it serves closes. */
export class TabLiveness {
  private readonly watched = new Set<string>();
  private readonly lock: ExclusiveLock;

  constructor(lock: ExclusiveLock) {
    this.lock = lock;
  }

  /** Calls gone once the tab's life lock is free, that is once the tab closed. */
  watch(tabId: string, gone: () => void): void {
    if (this.watched.has(tabId)) return;
    this.watched.add(tabId);
    void this.lock.run(tabLockName(tabId), async () => {
      this.watched.delete(tabId);
      gone();
    });
  }
}
