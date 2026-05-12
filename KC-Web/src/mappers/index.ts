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
export {
  base64ToHex,
  formatEUI,
  formatUnixNs,
  normalizeStatus,
  parseISODate,
  unixNsToDate,
} from "./common.mapper";

// Base station mappers
export { mapBaseStation, mapBaseStationList } from "./base-station.mapper";

// Endpoint mappers
export {
  deriveActivityStatus,
  deriveAttachState,
  mapEndpoint,
} from "./endpoint.mapper";

// Event and certificate mappers
export {
  deriveCertificateStatus,
  mapCertificate,
  mapCertificateList,
  mapEvent,
  mapEventList,
} from "./event.mapper";

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
export {
  mapAuthTokens,
  mapLoginResponse,
  mapUserMembership,
  mapUserProfile,
  type RawUserProfile,
} from "./auth.mapper";
