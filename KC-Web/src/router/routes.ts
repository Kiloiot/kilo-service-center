/**
 * Route Configuration
 *
 * Centralized route definitions with lazy loading support.
 * All routes are wrapped with FeatureProtectedRoute.
 */

import { lazy } from "react";

import type { FeatureFlagName } from "@contexts/FeatureFlagContext";
import { ROUTE_TITLES, ROUTES } from "@constants/app";

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
    element: Dashboard,
    title: ROUTE_TITLES.DASHBOARD,
    featureFlag: "scaci_dashboard",
  },
  {
    path: ROUTES.BASE_STATIONS,
    element: BaseStations,
    title: ROUTE_TITLES.BASE_STATIONS,
  },
  {
    path: ROUTES.BASE_STATION_DETAIL,
    element: BaseStations,
    title: ROUTE_TITLES.BASE_STATION_DETAIL,
  },
  {
    path: ROUTES.ENDPOINTS,
    element: EndPoints,
    title: ROUTE_TITLES.ENDPOINTS,
  },
  {
    path: ROUTES.ENDPOINT_DETAIL,
    element: EndPoints,
    title: ROUTE_TITLES.ENDPOINT_DETAIL,
  },
  // Blueprint feature routes
  {
    path: ROUTES.BLUEPRINTS,
    element: Blueprints,
    title: ROUTE_TITLES.BLUEPRINTS,
  },
  {
    path: ROUTES.BLUEPRINT_MANUFACTURER,
    element: Blueprints,
    title: ROUTE_TITLES.BLUEPRINT_MANUFACTURER,
  },
  {
    path: ROUTES.BLUEPRINT_MODEL,
    element: Blueprints,
    title: ROUTE_TITLES.BLUEPRINT_MODEL,
  },
  {
    path: ROUTES.BLUEPRINT_MODEL_NEW,
    element: AddDeviceModel,
    title: ROUTE_TITLES.BLUEPRINT_MODEL_NEW,
  },
  {
    path: ROUTES.BLUEPRINT_DETAIL,
    element: BlueprintDetail,
    title: ROUTE_TITLES.BLUEPRINT_DETAIL,
  },
  {
    path: ROUTES.CERTIFICATES,
    element: Certificates,
    title: ROUTE_TITLES.CERTIFICATES,
    featureFlag: "certificate_generation",
  },
  // Users and Roles tabbed page
  {
    path: ROUTES.USERS,
    element: UsersAndRoles,
    title: ROUTE_TITLES.USERS,
  },
  {
    path: ROUTES.USER_DETAIL,
    element: UserDetail,
    title: ROUTE_TITLES.USER_DETAIL,
  },
  {
    path: ROUTES.USER_PASSWORD,
    element: UserPassword,
    title: ROUTE_TITLES.USER_DETAIL, // Intentionally reuses USER_DETAIL title
  },
  // API Keys management
  {
    path: ROUTES.API_KEYS,
    element: ApiKeys,
    title: ROUTE_TITLES.API_KEYS,
  },
  // Organization management routes (ECE only)
  {
    path: ROUTES.ORGANIZATIONS,
    element: Organizations,
    title: ROUTE_TITLES.ORGANIZATIONS,
    featureFlag: "enterprise_organizations",
  },
  {
    path: ROUTES.ORGANIZATION_DETAIL,
    element: OrganizationDetail,
    title: ROUTE_TITLES.ORGANIZATION_DETAIL,
    featureFlag: "enterprise_organizations",
  },
  // Organization Users (ECE only)
  {
    path: ROUTES.ORGANIZATION_USERS,
    element: OrganizationUsersPage,
    title: ROUTE_TITLES.ORGANIZATION_USERS,
    featureFlag: "enterprise_organizations",
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

/**
 * Get route by path
 */
export const getRouteByPath = (path: string): RouteConfig | undefined => {
  return routes.find((route) => route.path === path);
};

/**
 * Get route title by path
 */
export const getRouteTitleByPath = (path: string): string => {
  const route = getRouteByPath(path);
  return route?.title || ROUTE_TITLES.DEFAULT;
};
