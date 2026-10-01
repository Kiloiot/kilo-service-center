import React, {
  createContext,
  type ReactNode,
  useCallback,
  useContext,
  useEffect,
  useMemo,
} from "react";

import { useVersionInfo } from "@hooks";

import { getErrorMessage } from "@utils/error-message";
import { logger } from "@utils/logger";
import { APP_ERRORS, LOG_MESSAGES, VERSION_INFO } from "@constants/messages";

interface VersionInfo {
  version: string;
  buildTime: string;
  gitCommit: string;
  gitBranch: string;
  buildUser: string;
  goVersion: string;
  schemaVersion: number;
  artifacts: Record<string, string>;
  isProduction: boolean;
  scEui?: string;
  scVendor?: string;
  scModel?: string;
  scName?: string;
  scSwVersion?: string;
  edition?: string;
  editionCode?: string;
  licenseId?: string;
  licenseUrl?: string;
  sourceUrl?: string;
  documentationUrl?: string;
  homepageUrl?: string;
  trademarkNotice?: string;
}

interface SystemContextValue {
  versionInfo: VersionInfo | null;
  loading: boolean;
  error: string | null;
  refreshVersion: () => Promise<void>;
}

const SystemContext = createContext<SystemContextValue | undefined>(undefined);

const fallbackVersionInfo = (): VersionInfo => ({
  version: VERSION_INFO.UNKNOWN,
  buildTime: new Date().toISOString(),
  gitCommit: VERSION_INFO.UNKNOWN,
  gitBranch: VERSION_INFO.UNKNOWN,
  buildUser: VERSION_INFO.UNKNOWN,
  goVersion: VERSION_INFO.UNKNOWN,
  schemaVersion: 0,
  artifacts: {},
  isProduction: false,
  edition: __APP_EDITION__,
  licenseId: __LICENSE_ID__,
  licenseUrl: __LICENSE_URL__,
  sourceUrl: __SOURCE_URL__,
  documentationUrl: __DOCS_URL__,
  homepageUrl: __HOMEPAGE_URL__,
  trademarkNotice: __TRADEMARK_NOTICE__,
});

export const SystemProvider: React.FC<{ children: ReactNode }> = ({
  children,
}) => {
  const { data, isLoading, error, refetch } = useVersionInfo();

  useEffect(() => {
    if (error) logger.error(LOG_MESSAGES.VERSION_FETCH_FAILED, error);
  }, [error]);

  // Build-time disclosure constants stand in when the backend is unreachable.
  const versionInfo = useMemo<VersionInfo | null>(
    () => data ?? (error ? fallbackVersionInfo() : null),
    [data, error],
  );

  const refreshVersion = useCallback(async () => {
    await refetch();
  }, [refetch]);

  return (
    <SystemContext.Provider
      value={{
        versionInfo,
        loading: isLoading,
        error: error ? getErrorMessage(error, VERSION_INFO.LOAD_FAILED) : null,
        refreshVersion,
      }}
    >
      {children}
    </SystemContext.Provider>
  );
};

// eslint-disable-next-line react-refresh/only-export-components
export const useSystem = (): SystemContextValue => {
  const context = useContext(SystemContext);

  if (!context) {
    throw new Error(APP_ERRORS.SYSTEM_CONTEXT_REQUIRED);
  }

  return context;
};
