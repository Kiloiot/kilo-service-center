/**
 * Route Configuration
 *
 * Centralized route definitions with lazy loading support.
 * All routes are wrapped with FeatureProtectedRoute.
 */

import { lazy } from "react";

import type { FeatureFlagName } from "@contexts/FeatureFlagContext";
import {
  FEATURE_FLAG,
  ROLE_REQUIREMENT,
  type RoleRequirement,
  ROUTE_TITLES,
  ROUTES,
} from "@constants/app";

/**
 * Route configuration interface
 */
export interface RouteConfig {
  /** URL path */
  path: string;
  /** Lazy-loaded component (code splitting enabled) */
  element: React.LazyExoticComponent<React.FC>;
  /** Page title for display */
  title: string;
  /** Optional feature flag that must be enabled for this route */
  featureFlag?: FeatureFlagName;
  /** Requirement the route places on the signed-in user's roles; none for public pages */
  requires?: RoleRequirement;
}

// Lazy load page components from modules (code splitting)
const Dashboard = lazy(() =>
  import("@modules/dashboard").then((m) => ({ default: m.Dashboard })),
);

const BaseStations = lazy(() =>
  import("@modules/base-stations").then((m) => ({ default: m.BaseStations })),
);

const EndPoints = lazy(() =>
  import("@modules/endpoints").then((m) => ({ default: m.EndPoints })),
);

const Traffic = lazy(() =>
  import("@modules/traffic").then((m) => ({ default: m.Traffic })),
);

const Logs = lazy(() =>
  import("@modules/logs").then((m) => ({ default: m.Logs })),
);

const Certificates = lazy(() =>
  import("@modules/certificates").then((m) => ({ default: m.Certificates })),
);

const Login = lazy(() =>
  import("@modules/auth").then((m) => ({ default: m.Login })),
);
const Register = lazy(() =>
  import("@modules/auth").then((m) => ({ default: m.Register })),
);
const AuthCallback = lazy(() =>
  import("@modules/auth").then((m) => ({ default: m.AuthCallback })),
);
const MyPassword = lazy(() =>
  import("@modules/auth").then((m) => ({ default: m.MyPassword })),
);

// User management
const UserDetail = lazy(() =>
  import("@modules/users").then((m) => ({ default: m.UserDetail })),
);
const UserPassword = lazy(() =>
  import("@modules/users").then((m) => ({ default: m.UserPassword })),
);

// Users and Roles tabbed page + Organization Users
const UsersAndRoles = lazy(() =>
  import("@modules/users").then((m) => ({ default: m.UsersAndRoles })),
);
const OrganizationUsersPage = lazy(() =>
  import("@modules/users").then((m) => ({ default: m.OrganizationUsers })),
);

// Organization management
const Organizations = lazy(() =>
  import("@modules/organizations").then((m) => ({ default: m.Organizations })),
);
const OrganizationDetail = lazy(() =>
  import("@modules/organizations").then((m) => ({
    default: m.OrganizationDetail,
  })),
);

// API Keys management
const ApiKeys = lazy(() =>
  import("@modules/api-keys").then((m) => ({ default: m.ApiKeys })),
);

// Blueprint feature: Device catalog and payload decoding
const Blueprints = lazy(() =>
  import("@modules/blueprints").then((m) => ({ default: m.Blueprints })),
);
const BlueprintDetail = lazy(() =>
  import("@modules/blueprints").then((m) => ({ default: m.BlueprintDetail })),
);
const AddDeviceModel = lazy(() =>
  import("@modules/blueprints").then((m) => ({ default: m.AddDeviceModel })),
);

/**
 * Application routes configuration
 *
 * Routes are defined here and consumed by AppRouter.
 * Add new routes by adding to this array.
 */
export const routes: RouteConfig[] = [
  {
    path: ROUTES.HOME,
    requires: ROLE_REQUIREMENT.ANY_ROLE,
    element: Dashboard,
    title: ROUTE_TITLES.DASHBOARD,
  },
  {
    path: ROUTES.BASE_STATIONS,
    requires: ROLE_REQUIREMENT.BASE_STATION_MANAGER,
    element: BaseStations,
    title: ROUTE_TITLES.BASE_STATIONS,
  },
  {
    path: ROUTES.BASE_STATION_DETAIL,
    requires: ROLE_REQUIREMENT.BASE_STATION_MANAGER,
    element: BaseStations,
    title: ROUTE_TITLES.BASE_STATION_DETAIL,
  },
  {
    path: ROUTES.ENDPOINTS,
    requires: ROLE_REQUIREMENT.ENDPOINT_MANAGER,
    element: EndPoints,
    title: ROUTE_TITLES.ENDPOINTS,
  },
  {
    path: ROUTES.ENDPOINT_DETAIL,
    requires: ROLE_REQUIREMENT.ENDPOINT_MANAGER,
    element: EndPoints,
    title: ROUTE_TITLES.ENDPOINT_DETAIL,
  },
  {
    path: ROUTES.TRAFFIC,
    element: Traffic,
    title: ROUTE_TITLES.TRAFFIC,
    requires: ROLE_REQUIREMENT.ENDPOINT_MANAGER,
  },
  {
    path: ROUTES.LOGS,
    element: Logs,
    title: ROUTE_TITLES.LOGS,
    requires: ROLE_REQUIREMENT.DEVICE_MANAGER,
  },
  // Blueprint feature routes
  {
    path: ROUTES.BLUEPRINTS,
    requires: ROLE_REQUIREMENT.ENDPOINT_MANAGER,
    element: Blueprints,
    title: ROUTE_TITLES.BLUEPRINTS,
  },
  {
    path: ROUTES.BLUEPRINT_MANUFACTURER,
    requires: ROLE_REQUIREMENT.ENDPOINT_MANAGER,
    element: Blueprints,
    title: ROUTE_TITLES.BLUEPRINT_MANUFACTURER,
  },
  {
    path: ROUTES.BLUEPRINT_MODEL,
    requires: ROLE_REQUIREMENT.ENDPOINT_MANAGER,
    element: Blueprints,
    title: ROUTE_TITLES.BLUEPRINT_MODEL,
  },
  {
    path: ROUTES.BLUEPRINT_MODEL_NEW,
    requires: ROLE_REQUIREMENT.ENDPOINT_MANAGER,
    element: AddDeviceModel,
    title: ROUTE_TITLES.BLUEPRINT_MODEL_NEW,
  },
  {
    path: ROUTES.BLUEPRINT_DETAIL,
    requires: ROLE_REQUIREMENT.ENDPOINT_MANAGER,
    element: BlueprintDetail,
    title: ROUTE_TITLES.BLUEPRINT_DETAIL,
  },
  {
    path: ROUTES.CERTIFICATES,
    requires: ROLE_REQUIREMENT.BASE_STATION_MANAGER,
    element: Certificates,
    title: ROUTE_TITLES.CERTIFICATES,
  },
  // Users and Roles tabbed page
  {
    path: ROUTES.USERS,
    requires: ROLE_REQUIREMENT.TENANT_MANAGER,
    element: UsersAndRoles,
    title: ROUTE_TITLES.USERS,
  },
  {
    path: ROUTES.USER_DETAIL,
    requires: ROLE_REQUIREMENT.ADMIN,
    element: UserDetail,
    title: ROUTE_TITLES.USER_DETAIL,
  },
  {
    path: ROUTES.USER_PASSWORD,
    requires: ROLE_REQUIREMENT.ADMIN,
    element: UserPassword,
    title: ROUTE_TITLES.USER_DETAIL, // Intentionally reuses USER_DETAIL title
  },
  // API Keys management
  {
    path: ROUTES.API_KEYS,
    requires: ROLE_REQUIREMENT.ADMIN,
    element: ApiKeys,
    title: ROUTE_TITLES.API_KEYS,
  },
  // Organization management routes (ECE only)
  {
    path: ROUTES.ORGANIZATIONS,
    requires: ROLE_REQUIREMENT.ADMIN,
    element: Organizations,
    title: ROUTE_TITLES.ORGANIZATIONS,
    featureFlag: FEATURE_FLAG.ENTERPRISE_ORGANIZATIONS,
  },
  {
    path: ROUTES.ORGANIZATION_DETAIL,
    requires: ROLE_REQUIREMENT.ADMIN,
    element: OrganizationDetail,
    title: ROUTE_TITLES.ORGANIZATION_DETAIL,
    featureFlag: FEATURE_FLAG.ENTERPRISE_ORGANIZATIONS,
  },
  // Organization Users (ECE only)
  {
    path: ROUTES.ORGANIZATION_USERS,
    requires: ROLE_REQUIREMENT.TENANT_MANAGER,
    element: OrganizationUsersPage,
    title: ROUTE_TITLES.ORGANIZATION_USERS,
    featureFlag: FEATURE_FLAG.ENTERPRISE_ORGANIZATIONS,
  },
  {
    path: ROUTES.LOGIN,
    element: Login,
    title: ROUTE_TITLES.LOGIN,
  },
  {
    path: ROUTES.REGISTER,
    element: Register,
    title: ROUTE_TITLES.REGISTER,
  },
  {
    path: ROUTES.AUTH_CALLBACK,
    element: AuthCallback,
    title: ROUTE_TITLES.AUTH_CALLBACK,
  },
  // Self-service password change
  {
    path: ROUTES.MY_PASSWORD,
    element: MyPassword,
    title: ROUTE_TITLES.MY_PASSWORD,
  },
];
