/**
 * Shown in place of every page to a signed-in user who holds no role.
 */

import React from "react";
import { useNavigate } from "react-router-dom";

import { EmptyState } from "@ui/EmptyState";
import { ROUTES } from "@constants/app";
import { NO_ACCESS } from "@constants/messages";
import { SecurityIcon } from "@theme/icons";
import { componentSpacing } from "@theme/index";

const NoAccessState: React.FC = () => {
  const navigate = useNavigate();

  return (
    <EmptyState
      icon={
        <SecurityIcon sx={{ fontSize: componentSpacing.stateView.iconSize }} />
      }
      title={NO_ACCESS.TITLE}
      description={NO_ACCESS.DESCRIPTION}
      action={{
        label: NO_ACCESS.CHANGE_PASSWORD,
        onClick: () => navigate(ROUTES.MY_PASSWORD),
      }}
    />
  );
};

export default NoAccessState;
