/**
 * Renders a listing's filter fields in one wrapping row and reports each edit
 * as a filter patch.
 */

import {
  Box,
  FormControlLabel,
  MenuItem,
  Switch,
  TextField,
} from "@mui/material";

import { FILTER_CONTROL } from "@constants/app";

import { CommitTextField } from "./CommitTextField";
import type { FilterField, SelectFilterField } from "./types";

interface FilterBarProps<F> {
  fields: readonly FilterField<F>[];
  filter: F;
  onChange: (patch: Partial<F>) => void;
}

// The empty menu value stands for "no predicate".
const ANY_VALUE = "";

function SelectControl<F>({
  field,
  filter,
  onChange,
}: {
  field: SelectFilterField<F>;
  filter: F;
  onChange: (patch: Partial<F>) => void;
}) {
  const handleChange = (raw: string) =>
    onChange(field.set(field.options.find((o) => o.value === raw)?.value));

  return (
    <TextField
      select
      size="small"
      label={field.label}
      value={field.get(filter) ?? ANY_VALUE}
      onChange={(event) => handleChange(event.target.value)}
      sx={{ minWidth: FILTER_CONTROL.MIN_WIDTH }}
    >
      {field.anyLabel !== undefined && (
        <MenuItem value={ANY_VALUE}>{field.anyLabel}</MenuItem>
      )}
      {field.options.map((option) => (
        <MenuItem key={option.value} value={option.value}>
          {option.label}
        </MenuItem>
      ))}
    </TextField>
  );
}

export function FilterBar<F>({ fields, filter, onChange }: FilterBarProps<F>) {
  return (
    <Box
      sx={{
        display: "flex",
        flexWrap: "wrap",
        alignItems: "flex-start",
        gap: FILTER_CONTROL.GAP,
        mb: FILTER_CONTROL.GAP,
      }}
    >
      {fields.map((field) => {
        switch (field.kind) {
          case "select":
            return (
              <SelectControl
                key={field.id}
                field={field}
                filter={filter}
                onChange={onChange}
              />
            );
          case "text":
            return (
              <CommitTextField
                key={field.id}
                label={field.label}
                value={field.get(filter)}
                validate={field.validate}
                onCommit={(value) => onChange(field.set(value))}
              />
            );
          case "toggle":
            return (
              <FormControlLabel
                key={field.id}
                label={field.label}
                control={
                  <Switch
                    size="small"
                    checked={field.get(filter)}
                    onChange={(_, checked) => onChange(field.set(checked))}
                  />
                }
              />
            );
        }
      })}
    </Box>
  );
}
