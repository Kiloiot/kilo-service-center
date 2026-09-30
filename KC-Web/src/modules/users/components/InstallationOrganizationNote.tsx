import React from "react";

import { Alert } from "@mui/material";

import { useOrganization } from "@contexts/OrganizationContext";
import { USER_FORM } from "@constants/messages";

/** Names the installation's organization, which every user belongs to. */
const InstallationOrganizationNote: React.FC = () => {
  const { organizationName } = useOrganization();
  return (
    <Alert severity="info">
      {USER_FORM.INFO_JOINS_INSTALLATION_ORG}
      {organizationName}
    </Alert>
  );
};

export default InstallationOrganizationNote;
