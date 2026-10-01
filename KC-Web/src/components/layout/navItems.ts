import type { ElementType } from "react";

import {
  NAV_ITEMS,
  ROLE_REQUIREMENT,
  type RoleRequirement,
  ROUTES,
} from "@constants/app";
import {
  BlueprintIcon,
  BusinessIcon,
  DashboardIcon,
  DeviceHubIcon,
  LogsIcon,
  PeopleIcon,
  RouterIcon,
  SecurityIcon,
  TrafficIcon,
  VpnKeyIcon,
} from "@theme/icons";

/** The role an entry requires per edition; an edition without one hides it. */
interface EditionRequirement {
  community?: RoleRequirement;
  enterprise?: RoleRequirement;
}

interface NavItem {
  title: string;
  path: string;
  icon: ElementType;
  requires: EditionRequirement;
}

const inEveryEdition = (role: RoleRequirement): EditionRequirement => ({
  community: role,
  enterprise: role,
});

/**
 * Users & Roles manages organization members in the enterprise edition, which
 * tenant managers may do; in the community edition it manages the server's
 * users, an administrator's task. Organizations exist in the enterprise
 * edition only.
 */
const NAV_ENTRIES: readonly NavItem[] = [
  {
    title: NAV_ITEMS.DASHBOARD,
    path: ROUTES.HOME,
    icon: DashboardIcon,
    requires: inEveryEdition(ROLE_REQUIREMENT.ANY_ROLE),
  },
  {
    title: NAV_ITEMS.BASE_STATIONS,
    path: ROUTES.BASE_STATIONS,
    icon: RouterIcon,
    requires: inEveryEdition(ROLE_REQUIREMENT.BASE_STATION_MANAGER),
  },
  {
    title: NAV_ITEMS.ENDPOINTS,
    path: ROUTES.ENDPOINTS,
    icon: DeviceHubIcon,
    requires: inEveryEdition(ROLE_REQUIREMENT.ENDPOINT_MANAGER),
  },
  {
    title: NAV_ITEMS.TRAFFIC,
    path: ROUTES.TRAFFIC,
    icon: TrafficIcon,
    requires: inEveryEdition(ROLE_REQUIREMENT.ENDPOINT_MANAGER),
  },
  {
    title: NAV_ITEMS.LOGS,
    path: ROUTES.LOGS,
    icon: LogsIcon,
    requires: inEveryEdition(ROLE_REQUIREMENT.DEVICE_MANAGER),
  },
  {
    title: NAV_ITEMS.BLUEPRINTS,
    path: ROUTES.BLUEPRINTS,
    icon: BlueprintIcon,
    requires: inEveryEdition(ROLE_REQUIREMENT.ENDPOINT_MANAGER),
  },
  {
    title: NAV_ITEMS.USERS,
    path: ROUTES.USERS,
    icon: PeopleIcon,
    requires: {
      community: ROLE_REQUIREMENT.ADMIN,
      enterprise: ROLE_REQUIREMENT.TENANT_MANAGER,
    },
  },
  {
    title: NAV_ITEMS.API_KEYS,
    path: ROUTES.API_KEYS,
    icon: VpnKeyIcon,
    requires: inEveryEdition(ROLE_REQUIREMENT.ADMIN),
  },
  {
    title: NAV_ITEMS.ORGANIZATIONS,
    path: ROUTES.ORGANIZATIONS,
    icon: BusinessIcon,
    requires: { enterprise: ROLE_REQUIREMENT.ADMIN },
  },
  {
    title: NAV_ITEMS.CERTIFICATES,
    path: ROUTES.CERTIFICATES,
    icon: SecurityIcon,
    requires: inEveryEdition(ROLE_REQUIREMENT.BASE_STATION_MANAGER),
  },
];

/** Navigation entries the signed-in user's roles allow in this edition. */
export const buildNavItems = (
  can: (requirement: RoleRequirement) => boolean,
  enterprise: boolean,
): NavItem[] =>
  NAV_ENTRIES.filter((entry) => {
    const role = enterprise
      ? entry.requires.enterprise
      : entry.requires.community;
    return role !== undefined && can(role);
  });
