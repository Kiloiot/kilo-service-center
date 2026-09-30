/**
 * Concrete paths of parameterized routes.
 */

import { generatePath } from "react-router-dom";

import { normalizeEui } from "@utils/eui";
import { ROUTES } from "@constants/app";

/** A base station's detail page, keyed by its canonical EUI. */
export const baseStationDetailPath = (eui: string): string =>
  generatePath(ROUTES.BASE_STATION_DETAIL, { id: normalizeEui(eui) });

/** An end point's detail page, keyed by its EUI as the service returns it. */
export const endpointDetailPath = (eui: string): string =>
  generatePath(ROUTES.ENDPOINT_DETAIL, { id: eui });

export const userDetailPath = (id: string): string =>
  generatePath(ROUTES.USER_DETAIL, { id });

export const userPasswordPath = (id: string): string =>
  generatePath(ROUTES.USER_PASSWORD, { id });

export const organizationDetailPath = (id: string): string =>
  generatePath(ROUTES.ORGANIZATION_DETAIL, { id });

export const organizationUsersPath = (id: string): string =>
  generatePath(ROUTES.ORGANIZATION_USERS, { id });

export const blueprintDetailPath = (id: string): string =>
  generatePath(ROUTES.BLUEPRINT_DETAIL, { id });
