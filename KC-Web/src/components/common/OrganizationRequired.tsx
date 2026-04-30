/**
 * Organization Required Component
 *
 * Displayed when a user navigates to a view that requires an organization
 * to be selected, but no organization is currently active.
 *
 * Thin wrapper around EmptyState to avoid duplicating empty-state UI.
 */

import React from 'react';
import { useNavigate } from 'react-router-dom';

import { EmptyState } from '@ui/EmptyState';
import { ROUTES } from '@constants/app';
import { USERS_AND_ROLES } from '@constants/messages';
import { BusinessIcon } from '@theme/icons';

const OrganizationRequired: React.FC = () => {
  const navigate = useNavigate();

  return (
    <EmptyState
      icon={<BusinessIcon sx={{ fontSize: 64 }} />}
      title={USERS_AND_ROLES.ORG_REQUIRED.TITLE}
      description={USERS_AND_ROLES.ORG_REQUIRED.DESCRIPTION}
      action={{
        label: USERS_AND_ROLES.ORG_REQUIRED.ACTION,
        onClick: () => navigate(ROUTES.ORGANIZATIONS),
      }}
    />
  );
};

export default OrganizationRequired;
