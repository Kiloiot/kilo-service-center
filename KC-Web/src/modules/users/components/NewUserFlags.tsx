import React from "react";

import { Box, FormControlLabel, Switch } from "@mui/material";

import { USER_FORM } from "@constants/messages";

import type { NewUserFlag, NewUserForm } from "../utils/new-user-form";

const FLAG_LABELS: ReadonlyArray<[NewUserFlag, string]> = [
  ["isActive", USER_FORM.LABEL_IS_ACTIVE],
  ["isAdmin", USER_FORM.LABEL_IS_ADMIN],
  ["isTenantManager", USER_FORM.LABEL_IS_TENANT_MGR],
  ["isBaseStationManager", USER_FORM.LABEL_IS_BS_MGR],
  ["isEndpointManager", USER_FORM.LABEL_IS_EP_MGR],
];

interface NewUserFlagsProps {
  form: NewUserForm;
  onChange: (flag: NewUserFlag, value: boolean) => void;
}

/** The status and role switches of a new user. */
const NewUserFlags: React.FC<NewUserFlagsProps> = ({ form, onChange }) => (
  <Box sx={{ display: "flex", flexDirection: "column", gap: 1 }}>
    {FLAG_LABELS.map(([flag, label]) => (
      <FormControlLabel
        key={flag}
        control={
          <Switch
            checked={form[flag]}
            onChange={(e) => onChange(flag, e.target.checked)}
          />
        }
        label={label}
      />
    ))}
  </Box>
);

export default NewUserFlags;
