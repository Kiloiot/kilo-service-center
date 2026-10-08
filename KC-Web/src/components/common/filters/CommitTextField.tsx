/**
 * Text filter that applies its value on Enter or blur, so a listing is
 * queried once per entry rather than once per keystroke.
 */

import { type KeyboardEvent, useState } from "react";

import { TextField } from "@mui/material";

import { FILTER_CONTROL } from "@constants/app";

interface CommitTextFieldProps {
  label: string;
  value: string | undefined;
  validate?: (value: string) => string | null;
  onCommit: (value: string | undefined) => void;
}

export function CommitTextField({
  label,
  value,
  validate,
  onCommit,
}: CommitTextFieldProps) {
  const [draft, setDraft] = useState(value ?? "");
  const [error, setError] = useState<string | null>(null);

  const commit = () => {
    const trimmed = draft.trim();
    const problem = trimmed ? (validate?.(trimmed) ?? null) : null;
    setError(problem);
    if (problem || trimmed === (value ?? "")) return;
    onCommit(trimmed || undefined);
  };

  const handleKeyDown = (event: KeyboardEvent) => {
    if (event.key === FILTER_CONTROL.COMMIT_KEY) commit();
  };

  return (
    <TextField
      label={label}
      size="small"
      value={draft}
      error={!!error}
      helperText={error ?? undefined}
      onChange={(event) => setDraft(event.target.value)}
      onBlur={commit}
      onKeyDown={handleKeyDown}
      sx={{ minWidth: FILTER_CONTROL.MIN_WIDTH }}
    />
  );
}
