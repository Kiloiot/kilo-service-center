import { validateEui } from "@utils/eui";
import { GEO_BOUNDS } from "@constants/app";
import {
  VAL_BS_EUI_FORMAT,
  VAL_BS_NAME_REQUIRED,
  VAL_LAT_LON_PAIR,
  VAL_LATITUDE_RANGE,
  VAL_LONGITUDE_RANGE,
} from "@constants/messages";

export interface CommissioningFormData {
  name: string;
  eui: string;
  latitude: string;
  longitude: string;
  altitude: string;
}

export type CommissioningErrors = Partial<
  Record<keyof CommissioningFormData | "general", string>
>;

export interface CertificateData {
  bsEui: string;
  serviceCenterUrl: string;
  downloadUrls: {
    caCert: string;
    clientCert: string;
    privateKey: string;
  };
  expiresAt: string;
}

export interface CommissionPayload {
  eui: string;
  name: string;
  latitude?: number;
  longitude?: number;
  altitude?: number;
}

export const EMPTY_COMMISSIONING_FORM: CommissioningFormData = {
  name: "",
  eui: "",
  latitude: "",
  longitude: "",
  altitude: "",
};

export function buildCommissionPayload(
  formData: CommissioningFormData,
): CommissionPayload {
  const payload: CommissionPayload = {
    eui: formData.eui,
    name: formData.name.trim(),
  };
  if (formData.latitude.trim() && formData.longitude.trim()) {
    payload.latitude = parseFloat(formData.latitude);
    payload.longitude = parseFloat(formData.longitude);
    if (formData.altitude.trim()) {
      payload.altitude = parseFloat(formData.altitude);
    }
  }
  return payload;
}

/** Latitude and longitude are optional but must be given as a pair. */
export function validateCommissioningForm(
  formData: CommissioningFormData,
): CommissioningErrors {
  const errors: CommissioningErrors = {};
  if (!formData.name.trim()) {
    errors.name = VAL_BS_NAME_REQUIRED;
  }
  if (validateEui(formData.eui) !== null) {
    errors.eui = VAL_BS_EUI_FORMAT;
  }
  const hasLat = formData.latitude.trim() !== "";
  const hasLng = formData.longitude.trim() !== "";
  if (hasLat !== hasLng) {
    errors.latitude = VAL_LAT_LON_PAIR;
    errors.longitude = VAL_LAT_LON_PAIR;
  }
  if (hasLat) {
    const lat = parseFloat(formData.latitude);
    if (
      isNaN(lat) ||
      lat < GEO_BOUNDS.LATITUDE_MIN ||
      lat > GEO_BOUNDS.LATITUDE_MAX
    ) {
      errors.latitude = VAL_LATITUDE_RANGE;
    }
  }
  if (hasLng) {
    const lng = parseFloat(formData.longitude);
    if (
      isNaN(lng) ||
      lng < GEO_BOUNDS.LONGITUDE_MIN ||
      lng > GEO_BOUNDS.LONGITUDE_MAX
    ) {
      errors.longitude = VAL_LONGITUDE_RANGE;
    }
  }
  return errors;
}
