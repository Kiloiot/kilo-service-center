/**
 * Organization Users Table Component
 *
 * Displays organization members by delegating to UsersTableBase.
 * Provides edit/remove action handlers.
 */

import React from "react";

import type { OrganizationUserUI, SystemUserUI } from "@api-types/api";

import { ORG_USERS_PAGE } from "@constants/messages";

import UsersTableBase, { type OrderBy } from "./UsersTableBase";

type OrgOrderBy = "email" | "role" | "status" | "createdAt";
type OrderDirection = "asc" | "desc";

export interface OrganizationUsersTableProps {
  users: OrganizationUserUI[];
  orderBy: OrgOrderBy;
  orderDirection: OrderDirection;
  onSort: (field: OrgOrderBy) => void;
  onEdit: (user: OrganizationUserUI) => void;
  onRemove: (user: OrganizationUserUI) => void;
  emptyMessage: string;
}

const OrganizationUsersTable: React.FC<OrganizationUsersTableProps> = ({
  users,
  orderBy,
  orderDirection,
  onSort,
  onEdit,
  onRemove,
  emptyMessage = ORG_USERS_PAGE.NO_USERS,
}) => {
  return (
    <UsersTableBase
      users={users}
      orderBy={orderBy}
      orderDirection={orderDirection}
      onSort={onSort as (field: OrderBy) => void}
      onEdit={onEdit as (user: SystemUserUI | OrganizationUserUI) => void}
      onRemove={onRemove as (user: SystemUserUI | OrganizationUserUI) => void}
      emptyMessage={emptyMessage}
    />
  );
};

export default OrganizationUsersTable;
