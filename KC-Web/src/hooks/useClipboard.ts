import { useCallback, useEffect, useRef, useState } from "react";

import { TIMING_COPY_FEEDBACK } from "@constants/app";

/**
 * Copy text to the clipboard and remember which field was copied for a short
 * feedback window.
 */
export function useClipboard() {
  const [copiedField, setCopiedField] = useState<string | null>(null);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(
    () => () => {
      if (timer.current) clearTimeout(timer.current);
    },
    [],
  );

  const copy = useCallback((text: string, field: string) => {
    void navigator.clipboard.writeText(text);
    setCopiedField(field);
    if (timer.current) clearTimeout(timer.current);
    timer.current = setTimeout(
      () => setCopiedField(null),
      TIMING_COPY_FEEDBACK,
    );
  }, []);

  const reset = useCallback(() => setCopiedField(null), []);

  return { copiedField, copy, reset };
}

export type ClipboardControls = ReturnType<typeof useClipboard>;
