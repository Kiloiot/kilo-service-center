/**
 * ExclusiveLock backed by the Web Locks API, which every tab of the origin shares.
 */

import type { ExclusiveLock } from "./refreshCoordinator";

export function webExclusiveLock(): ExclusiveLock {
  const locks = typeof navigator === "undefined" ? undefined : navigator.locks;
  // Web Locks exist only in secure contexts; without them no primitive can exclude other tabs.
  if (!locks) {
    return { run: (_name, task) => task() };
  }
  return {
    async run<T>(name: string, task: () => Promise<T>): Promise<T> {
      return locks.request(name, task);
    },
  };
}
