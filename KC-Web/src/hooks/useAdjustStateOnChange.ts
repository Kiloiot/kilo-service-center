import { useState } from "react";

const changed = (seen: readonly unknown[], deps: readonly unknown[]): boolean =>
  seen.length !== deps.length || deps.some((dep, i) => !Object.is(dep, seen[i]));

/**
 * Runs adjust during render on mount and whenever a dep changes, so state
 * derived from props is corrected before the stale value reaches the screen.
 * adjust may only set state of the calling component, and deps must keep their
 * identity between renders of that component.
 */
export function useAdjustStateOnChange(
  deps: readonly unknown[],
  adjust: () => void,
): void {
  const [seen, setSeen] = useState<readonly unknown[] | null>(null);
  if (seen === null || changed(seen, deps)) {
    setSeen(deps);
    adjust();
  }
}
