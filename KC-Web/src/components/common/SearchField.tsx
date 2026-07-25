/**
 * Reusable search text field with a leading search icon.
 * Replaces the repeated InputAdornment + SearchIcon pattern
 * across Organizations, OrganizationUsers, and Users pages.
 */

import React from "react";

import { InputAdornment, TextField } from "@mui/material";

import { SearchIcon } from "@theme/icons";

interface SearchFieldProps {
  placeholder: string;
  value: string;
  onChange: (
    event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>,
  ) => void;
}

const SearchField: React.FC<SearchFieldProps> = ({
  placeholder,
  value,
  onChange,
}) => (
  <TextField
    placeholder={placeholder}
    value={value}
    onChange={onChange}
    sx={{ flexGrow: 1 }}
    InputProps={{
      startAdornment: (
        <InputAdornment position="start">
          <SearchIcon />
        </InputAdornment>
      ),
    }}
  />
);

export default SearchField;
