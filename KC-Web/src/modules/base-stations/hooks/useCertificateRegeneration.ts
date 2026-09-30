import { useEffect, useState } from "react";

import type { GenerateCertificateResponse } from "@api-types/api";
import { useGenerateCertificate } from "@hooks";

import { useFeedback } from "@contexts/feedback";
import { CERT_VALIDITY_DAYS } from "@constants/app";
import { BASE_STATION_DETAILS } from "@constants/messages";

/**
 * Regenerates a base station's certificate behind a confirmation. The issued
 * bundle stays available until the edit dialog opens again or for another
 * station, so a refresh of the station's details never hides its downloads.
 */
export function useCertificateRegeneration(
  bsEui: string,
  open: boolean,
  onError: (message: string | null) => void,
) {
  const generateCertificate = useGenerateCertificate();
  const feedback = useFeedback();
  const [confirming, setConfirming] = useState(false);
  const [issued, setIssued] = useState<GenerateCertificateResponse | null>(
    null,
  );

  useEffect(() => {
    if (open) {
      setIssued(null);
      setConfirming(false);
    }
  }, [open, bsEui]);

  const regenerate = async () => {
    try {
      const response = await generateCertificate.mutateAsync({
        bsEui,
        validityDays: CERT_VALIDITY_DAYS.THREE_YEARS,
      });
      setIssued(response);
      setConfirming(false);
      onError(null);
      feedback.success(BASE_STATION_DETAILS.REGENERATE_CERTS_SUCCESS);
    } catch {
      onError(BASE_STATION_DETAILS.REGENERATE_CERTS_ERROR);
    }
  };

  return {
    issued,
    confirming,
    isRegenerating: generateCertificate.isPending,
    askToConfirm: () => setConfirming(true),
    cancel: () => setConfirming(false),
    regenerate,
  };
}
