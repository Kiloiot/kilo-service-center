import React from "react";

import { Box, Typography } from "@mui/material";

import { BackButton } from "@components/common/BackButton";
import { USERS_PAGE } from "@constants/messages";

interface UserDetailHeaderProps {
  onBack: () => void;
}

const UserDetailHeader: React.FC<UserDetailHeaderProps> = ({ onBack }) => (
  <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
    <Typography variant="h4" component="h1">
      {USERS_PAGE.DETAILS_TITLE}
    </Typography>
    <BackButton label={USERS_PAGE.BACK_TO_LIST} onClick={onBack} />
  </Box>
);

export default UserDetailHeader;
