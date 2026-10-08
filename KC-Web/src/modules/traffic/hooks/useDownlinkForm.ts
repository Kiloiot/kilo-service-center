/**
 * Downlink form state: the single-shot payload or the counter-dependent
 * rows, the protocol flags, and the validation derived from them. Starts
 * empty for a new downlink or from a queued one being edited.
 */

import { useCallback, useMemo, useState } from "react";

import {
  buildDownlinkContent,
  type DownlinkFormInit,
  type DownlinkFormValues,
  isDownlinkFormValid,
  NEW_DOWNLINK_FORM,
  validateDownlinkForm,
} from "../utils/downlink-form";
import { usePayloadRows } from "./usePayloadRows";

export function useDownlinkForm(init: DownlinkFormInit = NEW_DOWNLINK_FORM) {
  const [values, setValues] = useState<DownlinkFormValues>(init.values);
  const { payloadRows, resetRows, addRow, removeRow, updateRow } =
    usePayloadRows(init.payloadRows);

  // Switching modes discards the other mode's payload input.
  const setField = useCallback(
    <K extends keyof DownlinkFormValues>(
      field: K,
      value: DownlinkFormValues[K],
    ) => {
      setValues((prev) => ({ ...prev, [field]: value }));
      if (field === "cntDepend") {
        if (value) resetRows();
        else setValues((prev) => ({ ...prev, payload: "" }));
      }
    },
    [resetRows],
  );

  const validation = useMemo(
    () => validateDownlinkForm({ ...values, payloadRows }),
    [values, payloadRows],
  );

  /** Clears the payload input after a successful send; flags stay. */
  const reset = useCallback(() => {
    setValues((prev) => ({ ...prev, payload: "" }));
    resetRows();
  }, [resetRows]);

  const buildContent = useCallback(
    () => buildDownlinkContent(values, payloadRows, validation),
    [values, payloadRows, validation],
  );

  return {
    values,
    setField,
    payloadRows,
    addRow,
    removeRow,
    updateRow,
    validation,
    isValid: isDownlinkFormValid(validation),
    reset,
    buildContent,
  };
}

export type DownlinkForm = ReturnType<typeof useDownlinkForm>;
