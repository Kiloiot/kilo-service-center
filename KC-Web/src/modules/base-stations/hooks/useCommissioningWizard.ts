import { useState } from "react";

import {
  useCommissionBaseStationWithCerts,
  useRetryCertificateGeneration,
} from "@hooks";

import type { CommissioningPhase } from "@constants/app";
import {
  COMMISSION_RESULT_STATUS,
  COMMISSIONING_PHASE,
  FRACTION_DIGITS,
} from "@constants/app";

import {
  mapCommissionError,
  mapRetryCertsError,
} from "../utils/commissioning-errors";
import {
  buildCommissionPayload,
  type CertificateData,
  type CommissioningErrors,
  type CommissioningFormData,
  EMPTY_COMMISSIONING_FORM,
  validateCommissioningForm,
} from "../utils/commissioning-form";

/**
 * Drives the add-base-station wizard: the base station record is created
 * first so the certificates generated afterwards can be persisted against it.
 * A certificate failure after creation lands in the partial phase, where a
 * retry regenerates certificates without creating a second base station.
 */
export function useCommissioningWizard() {
  const [values, setValues] = useState<CommissioningFormData>(
    EMPTY_COMMISSIONING_FORM,
  );
  const [errors, setErrors] = useState<CommissioningErrors>({});
  const [phase, setPhase] = useState<CommissioningPhase>(
    COMMISSIONING_PHASE.INPUT,
  );
  const [certificateData, setCertificateData] =
    useState<CertificateData | null>(null);
  const [retryToken, setRetryToken] = useState<string | null>(null);
  const commission = useCommissionBaseStationWithCerts();
  const retryCertificates = useRetryCertificateGeneration();
  const isPending = commission.isPending || retryCertificates.isPending;

  const setField = (field: keyof CommissioningFormData, value: string) => {
    setValues((prev) => ({ ...prev, [field]: value }));
    setErrors((prev) => (prev[field] ? { ...prev, [field]: undefined } : prev));
  };

  const setLocation = (latitude: number, longitude: number) => {
    setValues((prev) => ({
      ...prev,
      latitude: latitude.toFixed(FRACTION_DIGITS.COORDINATE),
      longitude: longitude.toFixed(FRACTION_DIGITS.COORDINATE),
    }));
  };

  const submit = async () => {
    const validation = validateCommissioningForm(values);
    setErrors(validation);
    if (Object.keys(validation).length > 0) return;
    try {
      const result = await commission.mutateAsync(
        buildCommissionPayload(values),
      );
      if (
        result.status === COMMISSION_RESULT_STATUS.COMPLETE &&
        result.certData
      ) {
        setCertificateData({
          bsEui: result.bsEui,
          serviceCenterUrl: result.certData.serviceCenterUrl,
          downloadUrls: result.certData.downloadUrls,
          expiresAt: result.certData.expiryDate || "",
        });
        setPhase(COMMISSIONING_PHASE.SUCCESS);
      } else if (result.status === COMMISSION_RESULT_STATUS.PARTIAL) {
        setRetryToken(result.retryToken || result.bsEui);
        setPhase(COMMISSIONING_PHASE.PARTIAL);
      }
    } catch (err) {
      setErrors({ general: mapCommissionError(err) });
    }
  };

  const retryCerts = async () => {
    if (!retryToken) return;
    setErrors({});
    try {
      const result = await retryCertificates.mutateAsync({ bsEui: retryToken });
      setCertificateData({
        bsEui: result.bsEui,
        serviceCenterUrl: result.serviceCenterUrl,
        downloadUrls: result.downloadUrls,
        expiresAt: result.expiryDate || "",
      });
      setRetryToken(null);
      setPhase(COMMISSIONING_PHASE.SUCCESS);
    } catch (err) {
      setErrors({ general: mapRetryCertsError(err) });
    }
  };

  const failWith = (message: string) => setErrors({ general: message });

  const reset = () => {
    setValues(EMPTY_COMMISSIONING_FORM);
    setErrors({});
    setPhase(COMMISSIONING_PHASE.INPUT);
    setCertificateData(null);
    setRetryToken(null);
  };

  return {
    phase,
    values,
    errors,
    isPending,
    certificateData,
    setField,
    setLocation,
    submit,
    retryCerts,
    failWith,
    reset,
  };
}
