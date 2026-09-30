/** The counter-dependent payload rows of a downlink form. */

import { useCallback, useState } from "react";

import { emptyPayloadRow, type PayloadRow } from "../utils/downlink-form";

export function usePayloadRows(initial: PayloadRow[]) {
  const [payloadRows, setPayloadRows] = useState<PayloadRow[]>(initial);

  const resetRows = useCallback(() => setPayloadRows([emptyPayloadRow()]), []);
  const addRow = useCallback(
    () => setPayloadRows((rows) => [...rows, emptyPayloadRow()]),
    [],
  );
  const removeRow = useCallback(
    (index: number) =>
      setPayloadRows((rows) => rows.filter((_, i) => i !== index)),
    [],
  );
  const updateRow = useCallback(
    (index: number, patch: Partial<PayloadRow>) =>
      setPayloadRows((rows) =>
        rows.map((row, i) => (i === index ? { ...row, ...patch } : row)),
      ),
    [],
  );

  return { payloadRows, resetRows, addRow, removeRow, updateRow };
}
