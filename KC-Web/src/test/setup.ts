/**
 * Vitest Test Setup
 *
 * This file runs before each test file and configures the testing environment.
 */

import { cleanup } from "@testing-library/react";
import { afterEach, vi } from "vitest";

import "@testing-library/jest-dom";

// Pass-through mock: loads actual proto constructors from @kilocenter/grpc-stubs (pre-bundled CJS)
// CJS→ESM interop may wrap all exports under `default`, so we unwrap first.
vi.mock("@services/grpc/kilocenter_pb", async () => {
  const actual = await vi.importActual<Record<string, unknown>>(
    "@kilocenter/grpc-stubs/kilocenter_pb",
  );
  const exports = (actual.default ?? actual) as Record<string, unknown>;
  return {
    ...exports,
    default: exports,
  };
});

vi.mock("@services/grpc/kilocenter_pb_service", () => {
  // Create a mock class for KiloCenterServiceClient
  class MockKiloCenterServiceClient {
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    constructor(_host: string) {}
    login = vi.fn();
    logout = vi.fn();
    getAuthSettings = vi.fn();
    listBaseStations = vi.fn();
    getBaseStation = vi.fn();
    createBaseStation = vi.fn();
    updateBaseStation = vi.fn();
    deleteBaseStation = vi.fn();
    listEndPoints = vi.fn();
    getEndPoint = vi.fn();
    createEndPoint = vi.fn();
    updateEndPoint = vi.fn();
    deleteEndPoint = vi.fn();
    attachEndPoint = vi.fn();
    detachEndPoint = vi.fn();
    listEvents = vi.fn();
    getVersion = vi.fn();
    getProfile = vi.fn();
    refreshTokens = vi.fn();
    changePassword = vi.fn();
    exchangeOIDC = vi.fn();
    exchangeOAuth2 = vi.fn();
    listUsers = vi.fn();
    getUser = vi.fn();
    createUser = vi.fn();
    updateUser = vi.fn();
    deleteUser = vi.fn();
    updateUserPassword = vi.fn();
    listOrganizations = vi.fn();
    getOrganization = vi.fn();
    createOrganization = vi.fn();
    updateOrganization = vi.fn();
    deleteOrganization = vi.fn();
    listOrganizationUsers = vi.fn();
    getOrganizationUser = vi.fn();
    addOrganizationUser = vi.fn();
    updateOrganizationUser = vi.fn();
    removeOrganizationUser = vi.fn();
    generateCertificate = vi.fn();
    getServerCertificateStatus = vi.fn();
    getAnalyticsOverview = vi.fn();
    getAlertSummary = vi.fn();
    listBaseStationMessages = vi.fn();
    getBaseStationMessage = vi.fn();
    getBaseStationMessageStats = vi.fn();
    sendULTransmit = vi.fn();
    listManufacturers = vi.fn();
    getManufacturer = vi.fn();
    createManufacturer = vi.fn();
    updateManufacturer = vi.fn();
    deleteManufacturer = vi.fn();
    listDeviceModels = vi.fn();
    getDeviceModel = vi.fn();
    createDeviceModel = vi.fn();
    updateDeviceModel = vi.fn();
    deleteDeviceModel = vi.fn();
    listBlueprints = vi.fn();
    getBlueprint = vi.fn();
    createBlueprint = vi.fn();
    updateBlueprint = vi.fn();
    deleteBlueprint = vi.fn();
    setDefaultBlueprint = vi.fn();
    submitBlueprintToRegistry = vi.fn();
    getReleaseInfo = vi.fn();
    registerAccount = vi.fn();
    listBaseStationActivity = vi.fn();
    listEndpointActivity = vi.fn();
  }

  return {
    default: {},
    KiloCenterService: {
      serviceName: "kilocenter.api.v1.KiloCenterService",
      StreamMessages: {},
      StreamEvents: {},
      StreamBaseStationMessages: {},
    },
    KiloCenterServiceClient: MockKiloCenterServiceClient,
  };
});

// Cleanup after each test case (e.g., clearing jsdom)
afterEach(() => {
  cleanup();
});

// Mock window.matchMedia for components that use media queries
Object.defineProperty(window, "matchMedia", {
  writable: true,
  value: (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  }),
});

// Mock ResizeObserver for MUI components
global.ResizeObserver = class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
};
