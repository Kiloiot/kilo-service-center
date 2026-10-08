/**
 * Lossless decoding of the free-form JSON the backend stores: system event
 * details and blueprint output. An integer beyond Number.MAX_SAFE_INTEGER
 * (an int64 opId, queId or nanosecond time) keeps its exact decimal digits as
 * a string; every other value decodes as JSON.parse decodes it.
 */

import { JSON_INTEGER_SOURCE_REGEX } from "@constants/app";

/** The third reviver argument of the TC39 JSON.parse source text access proposal. */
interface ReviverContext {
  source?: string;
}

const textDecoder = new TextDecoder();

function keepExactInteger(
  _key: string,
  value: unknown,
  context?: ReviverContext,
): unknown {
  if (typeof value !== "number" || Number.isSafeInteger(value)) return value;
  const source = context?.source;
  return source !== undefined && JSON_INTEGER_SOURCE_REGEX.test(source)
    ? source
    : value;
}

/** Decodes UTF-8 JSON bytes; throws a SyntaxError on invalid JSON, as JSON.parse does. */
export function decodeExactJson(bytes: Uint8Array): unknown {
  return JSON.parse(textDecoder.decode(bytes), keepExactInteger);
}

/** A decoded JSON object, as opposed to an array, a primitive or null. */
export function isJsonObject(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

/** Decodes a JSON object blob; empty, invalid or non-object JSON yields undefined. */
export function decodeExactJsonObject(
  bytes: Uint8Array,
): Record<string, unknown> | undefined {
  if (bytes.length === 0) return undefined;
  try {
    const decoded = decodeExactJson(bytes);
    return isJsonObject(decoded) ? decoded : undefined;
  } catch {
    return undefined;
  }
}
