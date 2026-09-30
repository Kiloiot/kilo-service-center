/**
 * Helpers for listings whose view fixes some filter values, such as a
 * device's own traffic or a failures-only log.
 */

/**
 * Drops the columns or filter fields whose id names a value the view already
 * fixes: a base station's own traffic needs no bsEui column or filter.
 */
export function omitFixed<T extends { id: string }>(
  items: readonly T[],
  fixed: object,
): T[] {
  const keys = new Set(
    Object.entries(fixed)
      .filter(([, value]) => value !== undefined && value !== "")
      .map(([key]) => key),
  );
  return items.filter((item) => !keys.has(item.id));
}
