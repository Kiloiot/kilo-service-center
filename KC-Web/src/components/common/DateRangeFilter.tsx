/**
 * Start/end date filter of an activity feed. The fields keep what the admin
 * typed (DD/MM/YYYY, digits masked as they are typed); the range handed to
 * the caller covers whole local days, the end bound being the next midnight.
 */

import React, { useState } from "react";

import { Box, Button, FormHelperText, TextField } from "@mui/material";

import {
  maskDateInputEU,
  nextLocalMidnight,
  parseDateInputEU,
} from "@utils/date-format";
import { DATE_RANGE_FILTER } from "@constants/messages";
import { componentSpacing } from "@theme/index";

export interface DateRange {
  startTime?: string;
  endTime?: string;
}

type Bound = keyof DateRange;

const toBound: Record<Bound, (day: Date) => Date> = {
  startTime: (day) => day,
  endTime: nextLocalMidnight,
};

interface DateFieldProps {
  label: string;
  onCommit: (day: Date | undefined) => void;
  onReject: (text: string) => void;
  invalid: boolean;
  text: string;
  onText: (text: string) => void;
}

const DateField: React.FC<DateFieldProps> = ({
  label,
  onCommit,
  onReject,
  invalid,
  text,
  onText,
}) => {
  const commit = () => {
    const day = parseDateInputEU(text);
    if (text !== "" && !day) {
      onReject(text);
      return;
    }
    onCommit(day);
  };
  return (
    <TextField
      label={label}
      type="text"
      value={text}
      placeholder={DATE_RANGE_FILTER.DATE_PLACEHOLDER_EU}
      onChange={(e) => onText(maskDateInputEU(e.target.value))}
      onBlur={commit}
      error={invalid}
      size="small"
      slotProps={{
        inputLabel: { shrink: true },
        htmlInput: { inputMode: "numeric" },
      }}
      sx={{ minWidth: componentSpacing.dateInput.minWidth }}
    />
  );
};

interface DateRangeFilterProps {
  onChange: (range: DateRange) => void;
}

export const DateRangeFilter: React.FC<DateRangeFilterProps> = ({
  onChange,
}) => {
  const [texts, setTexts] = useState<Record<Bound, string>>({
    startTime: "",
    endTime: "",
  });
  const [range, setRange] = useState<DateRange>({});
  // The text of each field that failed to parse; its error clears once the text changes.
  const [rejected, setRejected] = useState<Record<Bound, string | null>>({
    startTime: null,
    endTime: null,
  });
  const isInvalid = (bound: Bound) =>
    rejected[bound] !== null && rejected[bound] === texts[bound];

  const commit = (bound: Bound) => (day: Date | undefined) => {
    const next = {
      ...range,
      [bound]: day && toBound[bound](day).toISOString(),
    };
    if (!next[bound]) delete next[bound];
    setRange(next);
    onChange(next);
  };

  const clear = () => {
    setTexts({ startTime: "", endTime: "" });
    setRejected({ startTime: null, endTime: null });
    setRange({});
    onChange({});
  };

  const field = (bound: Bound, label: string) => (
    <DateField
      label={label}
      text={texts[bound]}
      onText={(text) => setTexts((prev) => ({ ...prev, [bound]: text }))}
      onCommit={commit(bound)}
      onReject={(text) => setRejected((prev) => ({ ...prev, [bound]: text }))}
      invalid={isInvalid(bound)}
    />
  );

  // The error sits below the row so the fields and the button keep one height.
  return (
    <Box sx={{ mb: 2 }}>
      <Box sx={{ display: "flex", gap: 2 }}>
        {field("startTime", DATE_RANGE_FILTER.LABEL_START_DATE)}
        {field("endTime", DATE_RANGE_FILTER.LABEL_END_DATE)}
        <Button variant="outlined" onClick={clear}>
          {DATE_RANGE_FILTER.CLEAR_FILTERS}
        </Button>
      </Box>
      {(isInvalid("startTime") || isInvalid("endTime")) && (
        <FormHelperText error>
          {DATE_RANGE_FILTER.ERR_DATE_INVALID}
        </FormHelperText>
      )}
    </Box>
  );
};
