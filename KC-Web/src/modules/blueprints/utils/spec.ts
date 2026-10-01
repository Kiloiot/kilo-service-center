import { BLUEPRINT_LABELS } from "@constants/messages";

/**
 * A pasted blueprint may be the bare decoder spec or a full record nesting it
 * under `spec`; unwrap so the backend receives the decoder spec, not the record.
 */
export function unwrapBlueprintSpec(parsed: unknown): unknown {
  if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
    const rec = parsed as Record<string, unknown>;
    if (rec.spec && typeof rec.spec === "object") return rec.spec;
  }
  return parsed;
}

/**
 * Returns the parsed blueprint spec object when text is valid JSON, or throws
 * an Error with the localized invalid-JSON message otherwise. The blueprint
 * spec contract requires an object (not array, not primitive); primitives
 * round-trip through JSON.parse and would be rejected by the backend.
 */
export function validateBlueprintSpecJson(text: string): object {
  let parsed: unknown;
  try {
    parsed = JSON.parse(text);
  } catch {
    throw new Error(BLUEPRINT_LABELS.ERR_INVALID_JSON);
  }
  if (parsed === null || typeof parsed !== "object") {
    throw new Error(BLUEPRINT_LABELS.ERR_INVALID_JSON);
  }
  return unwrapBlueprintSpec(parsed) as object;
}

/** The field error of a blueprint specification: required when empty, invalid when not a JSON object. */
export function specJsonError(text: string): string | null {
  if (!text.trim()) return BLUEPRINT_LABELS.ERR_SPEC_JSON_REQUIRED;
  try {
    validateBlueprintSpecJson(text);
    return null;
  } catch {
    return BLUEPRINT_LABELS.ERR_INVALID_JSON;
  }
}
