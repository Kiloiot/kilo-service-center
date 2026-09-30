import React from "react";

import type { OrganizationUI } from "@api-types/api";
import {
  Autocomplete,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  FormControl,
  IconButton,
  InputLabel,
  MenuItem,
  Select,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Tooltip,
  Typography,
} from "@mui/material";

import { memberStatusChip } from "@utils/chipMappings";
import { ORG_ROLE } from "@constants/app";
import { ORG_USERS_PAGE, USER_FORM } from "@constants/messages";
import { AddIcon, DeleteIcon } from "@theme/icons";
import { componentSpacing } from "@theme/index";

export interface UserOrgMembership {
  orgId: string;
  orgName: string;
  role: string;
  status: string;
}

interface UserMembershipsCardProps {
  memberships: UserOrgMembership[];
  availableOrgs: OrganizationUI[];
  isOrgsLoading: boolean;
  isRemovePending: boolean;
  isAddPending: boolean;
  selectedOrg: OrganizationUI | null;
  selectedRole: string;
  onSelectedOrgChange: (org: OrganizationUI | null) => void;
  onSelectedRoleChange: (role: string) => void;
  onAddMembership: () => void;
  onRemoveMembership: (orgId: string) => void;
}

const UserMembershipsCard: React.FC<UserMembershipsCardProps> = ({
  memberships,
  availableOrgs,
  isOrgsLoading,
  isRemovePending,
  isAddPending,
  selectedOrg,
  selectedRole,
  onSelectedOrgChange,
  onSelectedRoleChange,
  onAddMembership,
  onRemoveMembership,
}) => (
  <Card>
    <CardContent>
      <Typography variant="h6" gutterBottom>
        {USER_FORM.SECTION_ORG_MEMBERSHIPS}
      </Typography>

      {isOrgsLoading ? (
        <Box display="flex" justifyContent="center" py={2}>
          <CircularProgress size={componentSpacing.spinner.section} />
        </Box>
      ) : memberships.length === 0 ? (
        <Typography color="text.secondary" sx={{ py: 2 }}>
          {USER_FORM.EMPTY_NO_MEMBERSHIPS}
        </Typography>
      ) : (
        <TableContainer>
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>{USER_FORM.COL_ORG_NAME}</TableCell>
                <TableCell>{ORG_USERS_PAGE.COL_ROLE}</TableCell>
                <TableCell>{ORG_USERS_PAGE.COL_STATUS}</TableCell>
                <TableCell>{ORG_USERS_PAGE.COL_ACTIONS}</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {memberships.map((m) => (
                <TableRow key={m.orgId}>
                  <TableCell>
                    <Typography variant="body2">{m.orgName}</Typography>
                  </TableCell>
                  <TableCell>
                    <Chip label={m.role} size="small" />
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={m.status}
                      size="small"
                      color={memberStatusChip(m.status).color}
                    />
                  </TableCell>
                  <TableCell>
                    <Tooltip title={ORG_USERS_PAGE.TOOLTIP_REMOVE}>
                      <IconButton
                        size="small"
                        color="error"
                        onClick={() => onRemoveMembership(m.orgId)}
                        disabled={isRemovePending}
                      >
                        <DeleteIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}

      <Box sx={{ mt: 3, pt: 2, borderTop: 1, borderColor: "divider" }}>
        <Typography variant="subtitle2" gutterBottom>
          {USER_FORM.LABEL_ADD_TO_ORG}
        </Typography>
        <Box display="flex" gap={2} alignItems="flex-start">
          <Autocomplete
            options={availableOrgs}
            getOptionLabel={(option) => option.name}
            value={selectedOrg}
            onChange={(_, newValue) => onSelectedOrgChange(newValue)}
            isOptionEqualToValue={(option, value) => option.id === value.id}
            sx={{
              minWidth: componentSpacing.selectField.wideMinWidth,
              flex: 1,
            }}
            renderInput={(params) => (
              <TextField
                {...params}
                label={USER_FORM.LABEL_SELECT_ORG}
                size="small"
              />
            )}
          />
          <FormControl
            size="small"
            sx={{ minWidth: componentSpacing.selectField.minWidth }}
          >
            <InputLabel>{USER_FORM.LABEL_SELECT_ROLE}</InputLabel>
            <Select
              value={selectedRole}
              onChange={(e) => onSelectedRoleChange(e.target.value)}
              label={USER_FORM.LABEL_SELECT_ROLE}
            >
              <MenuItem value={ORG_ROLE.MEMBER}>
                {ORG_USERS_PAGE.ROLE_MEMBER}
              </MenuItem>
              <MenuItem value={ORG_ROLE.ADMIN}>
                {ORG_USERS_PAGE.ROLE_ADMIN}
              </MenuItem>
              <MenuItem value={ORG_ROLE.OWNER}>
                {ORG_USERS_PAGE.ROLE_OWNER}
              </MenuItem>
            </Select>
          </FormControl>
          <Button
            variant="contained"
            size="small"
            startIcon={<AddIcon />}
            onClick={onAddMembership}
            disabled={!selectedOrg || isAddPending}
            sx={{ mt: 0.5 }}
          >
            {USER_FORM.ACTION_ADD_MEMBERSHIP}
          </Button>
        </Box>
      </Box>
    </CardContent>
  </Card>
);

export default UserMembershipsCard;
