/**
 * A text field bound to one End Point form field: its value, its error, its
 * required marker and its validation when the administrator leaves it.
 */

import React from "react";

import { TextField, type TextFieldProps } from "@mui/material";
import Grid, { type GridSize } from "@mui/material/Grid";

import { componentSpacing } from "@theme/index";

import type { EndpointFormState } from "../hooks";
import {
  ENDPOINT_REQUIRED_FIELDS,
  type EndpointField,
} from "../utils/endpoint-validation";

export type FieldSize = { xs: GridSize; md: GridSize } | GridSize;

const HALF_WIDTH: FieldSize = componentSpacing.gridSpan.half;

export interface EndpointTextFieldProps {
  form: EndpointFormState;
  field: EndpointField;
  label: string;
  helper: string;
  size?: FieldSize;
  inputProps?: TextFieldProps["inputProps"];
  InputProps?: TextFieldProps["InputProps"];
  type?: string;
  placeholder?: string;
  disabled?: boolean;
  /** Shown instead of the form value, e.g. a revealed stored key. */
  value?: string;
}

export const EndpointTextField: React.FC<EndpointTextFieldProps> = ({
  form,
  field,
  label,
  helper,
  size = HALF_WIDTH,
  ...textFieldProps
}) => (
  <Grid size={size}>
    <TextField
      fullWidth
      label={label}
      value={form.values[field]}
      onChange={form.handleInput(field)}
      onBlur={form.handleBlur(field)}
      error={!!form.errors[field]}
      helperText={form.errors[field] || helper}
      required={
        ENDPOINT_REQUIRED_FIELDS.has(field) && !form.storedKeys.has(field)
      }
      {...textFieldProps}
    />
  </Grid>
);
