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
