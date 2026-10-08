import {
  ENDPOINT_ACTIVITY,
  ENDPOINT_ATTACH_STATUS,
  EVENT_SEVERITY,
  ORG_MEMBER_STATUS,
  ORG_ROLE,
  ORG_STATE,
} from "@constants/app";
import {
  ENDPOINT_DETAILS,
  ENDPOINTS_PAGE,
  ORG_USERS_PAGE,
  ORGANIZATIONS_PAGE,
} from "@constants/messages";

export type ChipColor = "success" | "info" | "warning" | "error" | "default";

export interface ChipPresentation {
  label: string;
  color: ChipColor;
}

const organizationStateChips: Record<string, ChipPresentation> = {
  [ORG_STATE.ACTIVE]: {
    label: ORGANIZATIONS_PAGE.STATE_ACTIVE,
    color: "success",
  },
  [ORG_STATE.SUSPENDED]: {
    label: ORGANIZATIONS_PAGE.STATE_SUSPENDED,
    color: "warning",
  },
  [ORG_STATE.ARCHIVED]: {
    label: ORGANIZATIONS_PAGE.STATE_ARCHIVED,
    color: "error",
  },
};

const memberStatusChips: Record<string, ChipPresentation> = {
  [ORG_MEMBER_STATUS.ACTIVE]: {
    label: ORG_USERS_PAGE.STATUS_ACTIVE,
    color: "success",
  },
  [ORG_MEMBER_STATUS.INVITED]: {
    label: ORG_USERS_PAGE.STATUS_INVITED,
    color: "warning",
  },
  [ORG_MEMBER_STATUS.REMOVED]: {
    label: ORG_USERS_PAGE.STATUS_REMOVED,
    color: "error",
  },
};

const attachStatusChips: Record<string, ChipPresentation> = {
  [ENDPOINT_ATTACH_STATUS.ATTACHED]: {
    label: ENDPOINT_DETAILS.ATTACH_STATUS_ATTACHED,
    color: "success",
  },
  [ENDPOINT_ATTACH_STATUS.ATTACHING]: {
    label: ENDPOINT_DETAILS.ATTACH_STATUS_ATTACHING,
    color: "info",
  },
  [ENDPOINT_ATTACH_STATUS.PENDING]: {
    label: ENDPOINT_DETAILS.ATTACH_STATUS_PENDING,
    color: "info",
  },
  [ENDPOINT_ATTACH_STATUS.DETACHED]: {
    label: ENDPOINT_DETAILS.ATTACH_STATUS_DETACHED,
    color: "warning",
  },
  [ENDPOINT_ATTACH_STATUS.UNKNOWN]: {
    label: ENDPOINT_DETAILS.ATTACH_STATUS_UNKNOWN,
    color: "default",
  },
};

const endpointActivityChips: Record<string, ChipPresentation> = {
  [ENDPOINT_ACTIVITY.ACTIVE]: {
    label: ENDPOINTS_PAGE.ACTIVE,
    color: "success",
  },
  [ENDPOINT_ACTIVITY.INACTIVE]: {
    label: ENDPOINTS_PAGE.INACTIVE,
    color: "default",
  },
};

const memberRoleLabels: Record<string, string> = {
  [ORG_ROLE.OWNER]: ORG_USERS_PAGE.ROLE_OWNER,
  [ORG_ROLE.ADMIN]: ORG_USERS_PAGE.ROLE_ADMIN,
  [ORG_ROLE.MEMBER]: ORG_USERS_PAGE.ROLE_MEMBER,
};

/** Chip label and color for an organization state; unknown states render as-is. */
export const organizationStateChip = (state: string): ChipPresentation =>
  organizationStateChips[state] ?? { label: state, color: "default" };

/** Chip label and color for a membership status; unknown statuses render as-is. */
export const memberStatusChip = (status: string): ChipPresentation =>
  memberStatusChips[status] ?? { label: status, color: "default" };

/** Display label for a membership role; unknown roles render as-is. */
export const memberRoleLabel = (role: string): string =>
  memberRoleLabels[role] ?? role;

const severityColors: Record<string, ChipColor> = {
  [EVENT_SEVERITY.CRITICAL]: "error",
  [EVENT_SEVERITY.ERROR]: "error",
  [EVENT_SEVERITY.WARNING]: "warning",
  [EVENT_SEVERITY.INFO]: "info",
};

/** Chip for a system event severity, labelled with the stored severity name. */
export const eventSeverityChip = (severity: string): ChipPresentation => ({
  label: severity,
  color: severityColors[severity] ?? "default",
});

/** Chip label and color for an end point attach state; anything else is unknown. */
export const attachStatusChip = (status?: string): ChipPresentation =>
  attachStatusChips[status ?? ""] ??
  attachStatusChips[ENDPOINT_ATTACH_STATUS.UNKNOWN];

/** Chip label and color for an end point's activity; anything not active reads as inactive. */
export const endpointActivityChip = (status?: string): ChipPresentation =>
  endpointActivityChips[status ?? ""] ??
  endpointActivityChips[ENDPOINT_ACTIVITY.INACTIVE];
