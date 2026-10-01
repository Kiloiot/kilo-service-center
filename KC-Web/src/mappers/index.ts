/**
 * Mappers Barrel Export
 *
 * MIOTY type transformation utilities.
 * Import via '@mappers' alias (to be configured in tsconfig).
 *
 * @example
 * import { mapBaseStation, mapEndpoint } from '@mappers';
 */

// Common utilities

// Base station mappers
export { mapBaseStation } from "./base-station.mapper";
export { mapBaseStationLocation } from "./base-station-location.mapper";

// Endpoint mappers
export { mapEndpoint } from "./endpoint.mapper";

// Event mappers
export { mapEventLogEntry } from "./event-log.mapper";

// Uplink mappers
export { mapStationUplink, mapUplink } from "./uplink.mapper";

// User mappers
export { mapUser, mapUserList } from "./user.mapper";

// Organization mappers
export {
  mapOrganization,
  mapOrganizationList,
  mapOrgUser,
  mapOrgUserList,
} from "./organization.mapper";

// Auth mappers (snake_case → camelCase for login response)
export { mapLoginResponse, mapUserProfile } from "./auth.mapper";

// Pagination
export { mapListPage } from "./pagination.mapper";

// SCACI control plane
export { mapScaciSession, mapScaciStatus } from "./scaci.mapper";

// Alerts
export { mapAlert, mapAlertSummary } from "./alert.mapper";

// Base station operations
export { mapBaseStationAvailability } from "./base-station-operations.mapper";
