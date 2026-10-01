/**
 * Search box and the Filters menu that narrows the End Points list by attach
 * state and activity.
 */

import React, { useState } from "react";

import {
  Badge,
  Box,
  Button,
  Checkbox,
  FormControlLabel,
  FormGroup,
  InputAdornment,
  Popover,
  TextField,
  Typography,
} from "@mui/material";

import { attachStatusChip } from "@utils/chipMappings";
import { ENDPOINT_ACTIVITY, ENDPOINT_ATTACH_STATUS } from "@constants/app";
import { ENDPOINTS_PAGE } from "@constants/messages";
import { FilterListIcon, SearchIcon } from "@theme/icons";

interface FilterOption {
  value: string;
  label: string;
}

const ATTACH_OPTIONS: FilterOption[] = Object.values(
  ENDPOINT_ATTACH_STATUS,
).map((value) => ({ value, label: attachStatusChip(value).label }));

const ACTIVITY_OPTIONS: FilterOption[] = [
  { value: ENDPOINT_ACTIVITY.ACTIVE, label: ENDPOINTS_PAGE.ACTIVE },
  { value: ENDPOINT_ACTIVITY.INACTIVE, label: ENDPOINTS_PAGE.INACTIVE },
];

export interface EndpointsFiltersBarProps {
  search: string;
  attachState: string[];
  activity: string[];
  onSearchChange: (value: string) => void;
  onAttachStateChange: (values: string[]) => void;
  onActivityChange: (values: string[]) => void;
}

const toggle = (values: string[], value: string): string[] =>
  values.includes(value)
    ? values.filter((v) => v !== value)
    : [...values, value];

function FilterGroup({
  title,
  options,
  selected,
  onChange,
}: {
  title: string;
  options: FilterOption[];
  selected: string[];
  onChange: (values: string[]) => void;
}) {
  return (
    <Box mb={1}>
      <Typography variant="subtitle2" color="text.secondary">
        {title}
      </Typography>
      <FormGroup>
        {options.map((option) => (
          <FormControlLabel
            key={option.value}
            label={option.label}
            control={
              <Checkbox
                size="small"
                checked={selected.includes(option.value)}
                onChange={() => onChange(toggle(selected, option.value))}
              />
            }
          />
        ))}
      </FormGroup>
    </Box>
  );
}

export const EndpointsFiltersBar: React.FC<EndpointsFiltersBarProps> = ({
  search,
  attachState,
  activity,
  onSearchChange,
  onAttachStateChange,
  onActivityChange,
}) => {
  const [anchor, setAnchor] = useState<HTMLElement | null>(null);
  return (
    <Box display="flex" gap={2} mb={2}>
      <TextField
        placeholder={ENDPOINTS_PAGE.SEARCH_PLACEHOLDER}
        value={search}
        onChange={(e) => onSearchChange(e.target.value)}
        sx={{ flexGrow: 1 }}
        slotProps={{
          input: {
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon />
              </InputAdornment>
            ),
          },
        }}
      />
      <Badge
        badgeContent={attachState.length + activity.length}
        color="primary"
      >
        <Button
          variant="outlined"
          startIcon={<FilterListIcon />}
          onClick={(e) => setAnchor(e.currentTarget)}
        >
          {ENDPOINTS_PAGE.FILTERS}
        </Button>
      </Badge>
      <Popover
        open={!!anchor}
        anchorEl={anchor}
        onClose={() => setAnchor(null)}
        anchorOrigin={{ vertical: "bottom", horizontal: "right" }}
        transformOrigin={{ vertical: "top", horizontal: "right" }}
      >
        <Box p={2}>
          <FilterGroup
            title={ENDPOINTS_PAGE.FILTER_ATTACH_STATE}
            options={ATTACH_OPTIONS}
            selected={attachState}
            onChange={onAttachStateChange}
          />
          <FilterGroup
            title={ENDPOINTS_PAGE.FILTER_ACTIVITY}
            options={ACTIVITY_OPTIONS}
            selected={activity}
            onChange={onActivityChange}
          />
        </Box>
      </Popover>
    </Box>
  );
};
