/**
 * Organizations a new user joins: a picker in the enterprise edition, the
 * installation's organization (joined automatically) in the community edition.
 */

import React, { useEffect } from "react";

import type { OrganizationUI } from "@api-types/api";
import { Autocomplete, Chip, TextField } from "@mui/material";

import { useOrganization } from "@contexts/OrganizationContext";
import { useOrganizations } from "@hooks/useOrganizations";
import { PAGINATION } from "@constants/app";
import { USER_FORM } from "@constants/messages";

import { useOrganizationAssignment } from "../hooks";
import InstallationOrganizationNote from "./InstallationOrganizationNote";

interface NewUserOrganizationsProps {
  open: boolean;
  value: OrganizationUI[];
  onChange: (orgs: OrganizationUI[]) => void;
}

const NewUserOrganizations: React.FC<NewUserOrganizationsProps> = ({
  open,
  value,
  onChange,
}) => {
  const assignable = useOrganizationAssignment();
  const { organizationId } = useOrganization();
  const { data } = useOrganizations(
    PAGINATION.ORG_PICKER_PAGE_SIZE,
    0,
    undefined,
    {
      enabled: open && assignable,
    },
  );
  const organizations = data?.organizations ?? [];

  // Preselect the organization the admin is working in.
  useEffect(() => {
    if (value.length > 0) return;
    const current = organizations.find((o) => o.id === organizationId);
    if (current) onChange([current]);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [organizations.length, organizationId]);

  if (!assignable) return <InstallationOrganizationNote />;

  return (
    <Autocomplete
      multiple
      options={organizations}
      getOptionLabel={(option) => option.name}
      value={value}
      onChange={(_, next) => onChange(next)}
      isOptionEqualToValue={(option, selected) => option.id === selected.id}
      renderTags={(selected, getTagProps) =>
        selected.map((option, index) => {
          const { key, ...chipProps } = getTagProps({ index });
          return (
            <Chip key={key} label={option.name} size="small" {...chipProps} />
          );
        })
      }
      renderInput={(params) => (
        <TextField
          {...params}
          label={USER_FORM.LABEL_ORGANIZATIONS}
          helperText={USER_FORM.HELPER_ORGANIZATIONS}
        />
      )}
    />
  );
};

export default NewUserOrganizations;
