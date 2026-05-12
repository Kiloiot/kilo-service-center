import React, {
  createContext,
  type ReactNode,
  useContext,
  useEffect,
  useState,
} from "react";

import { api } from "@services/api";
import { logger } from "@utils/logger";

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

export const SystemProvider: React.FC<{ children: ReactNode }> = ({
  children,
}) => {
  const [versionInfo, setVersionInfo] = useState<VersionInfo | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  const fetchVersion = async () => {
    try {
      setLoading(true);
      setError(null);
      const version = await api.getVersion();
      setVersionInfo(version);
    } catch (err) {
      logger.error("Failed to fetch version info:", err);
      setError(
        err instanceof Error ? err.message : "Failed to load version info",
      );
      // Set fallback version info using build-time disclosure constants
      setVersionInfo({
        version: "unknown",
        buildTime: new Date().toISOString(),
        gitCommit: "unknown",
        gitBranch: "unknown",
        buildUser: "unknown",
        goVersion: "unknown",
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
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchVersion();
  }, []);

  const refreshVersion = async () => {
    await fetchVersion();
  };

  return (
    <SystemContext.Provider
      value={{
        versionInfo,
        loading,
        error,
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
    throw new Error("useSystem must be used within SystemProvider");
  }

  return context;
};
