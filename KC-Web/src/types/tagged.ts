/**
 * A tagged union built from a kind-to-payload map, and a handler per kind.
 * Declaring both from the one map lets a handler table be total over the
 * kinds and lets each handler receive its own variant without narrowing.
 */

export type Tagged<M, K extends keyof M = keyof M> = {
  [P in K]: { kind: P } & M[P];
}[K];

export type KindHandlers<M, R> = {
  [K in keyof M]: (variant: Tagged<M, K>) => R;
};
