import { useState } from "react";

import type { DecodePreviewResponse } from "@api-types/api";

import { getErrorMessage } from "@utils/error-message";

import { useDecodePreview } from "./useBlueprints";

export function useDecodePlayground(blueprintId: string | undefined) {
  const [payload, setPayload] = useState("");
  const [formatId, setFormatId] = useState(0);
  const [result, setResult] = useState<DecodePreviewResponse | null>(null);
  const decode = useDecodePreview(blueprintId);

  const run = () => {
    if (!payload) return;
    decode.mutate(
      { userData: payload, formatId },
      {
        onSuccess: setResult,
        onError: (err: Error) =>
          setResult({
            success: false,
            errorDetail: getErrorMessage(err),
            formatId,
          }),
      },
    );
  };

  return {
    payload,
    setPayload,
    formatId,
    setFormatId,
    result,
    run,
    isPending: decode.isPending,
  };
}
