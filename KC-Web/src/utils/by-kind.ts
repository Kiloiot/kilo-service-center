/**
 * Dispatch on the kind of a tagged union: the handler table names one
 * handler per kind, so a new kind is a new entry rather than a new branch.
 */

import type { KindHandlers, Tagged } from "@api-types/tagged";

export function byKind<M, K extends keyof M, R>(
  variant: Tagged<M, K>,
  handlers: KindHandlers<M, R>,
): R {
  return handlers[variant.kind](variant);
}
