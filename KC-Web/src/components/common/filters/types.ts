/**
 * Declarative filter fields: each field reads its value from the listing's
 * filter object and turns an edit into a filter patch.
 */

export interface SelectOption<V extends string> {
  value: V;
  label: string;
}

interface FilterFieldBase {
  /** Filter property the field edits; also names the scope key that hides it. */
  id: string;
  label: string;
}

export interface SelectFilterField<
  F,
  V extends string = string,
> extends FilterFieldBase {
  kind: "select";
  options: readonly SelectOption<V>[];
  /** When set, an extra first option clears the predicate. */
  anyLabel?: string;
  get(filter: F): V | undefined;
  set(value: V | undefined): Partial<F>;
}

export interface TextFilterField<F> extends FilterFieldBase {
  kind: "text";
  /** Returns the error shown for an entry that cannot be applied, else null. */
  validate?: (value: string) => string | null;
  get(filter: F): string | undefined;
  set(value: string | undefined): Partial<F>;
}

export interface ToggleFilterField<F> extends FilterFieldBase {
  kind: "toggle";
  get(filter: F): boolean;
  set(value: boolean): Partial<F>;
}

export type FilterField<F> =
  | SelectFilterField<F>
  | TextFilterField<F>
  | ToggleFilterField<F>;
