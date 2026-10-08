/**
 * Feature Flag Context
 *
 * Provides feature flag management for the application.
 * Flags are loaded from src/config/flags/default.json.
 *
 * Usage:
 *   const { isEnabled, flags, loading } = useFeatureFlags();
 *    */

import type { ReactNode } from "react";
import React, {
  createContext,
  useContext,
  useEffect,
  useRef,
  useState,
} from "react";

import { APP_ERRORS } from "@constants/messages";
import defaultFlags from "@config/flags/default.json";

// Type for flag names based on default.json keys
export type FeatureFlagName = keyof typeof defaultFlags;

// Feature flags record type
export type FeatureFlags = Record<FeatureFlagName, boolean>;

interface FeatureFlagContextValue {
  /** All feature flags */
  flags: FeatureFlags;
  /** Check if a specific flag is enabled */
  isEnabled: (flag: FeatureFlagName) => boolean;
  /** Loading state while flags are being loaded */
  loading: boolean;
}

const FeatureFlagContext = createContext<FeatureFlagContextValue | undefined>(
  undefined,
);

interface FeatureFlagProviderProps {
  children: ReactNode;
  /** Override default flags (useful for testing) */
  overrides?: Partial<FeatureFlags>;
}

/**
 * Feature Flag Provider
 *
 * Wraps the application to provide feature flag context.
 * Loads flags from default.json with optional overrides.
 */
export const FeatureFlagProvider: React.FC<FeatureFlagProviderProps> = ({
  children,
  overrides,
}) => {
  const [flags, setFlags] = useState<FeatureFlags>(defaultFlags);
  const [loading, setLoading] = useState(true);

  // Use ref to avoid re-render storm from default {} object
  const overridesRef = useRef<Partial<FeatureFlags> | undefined>(overrides);

  // Update ref if overrides prop changes (for testing scenarios)
  useEffect(() => {
    overridesRef.current = overrides;
  }, [overrides]);

  // Load flags ONCE on mount - no dependency on overrides to prevent infinite loop
  useEffect(() => {
    const loadFlags = async () => {
      try {
        // Simulate async loading (for future API integration)
        await Promise.resolve();
        setFlags({
          ...defaultFlags,
          ...(overridesRef.current || {}),
        });
      } finally {
        setLoading(false);
      }
    };

    loadFlags();
  }, []); // RUN ONCE - do not depend on overrides

  // Recompute flags when overrides change (e.g., edition info arrives asynchronously)
  useEffect(() => {
    if (overrides !== undefined) {
      setFlags((prev) => ({
        ...prev,
        ...overrides,
      }));
    }
  }, [overrides]);

  const isEnabled = (flag: FeatureFlagName): boolean => {
    return flags[flag] === true;
  };

  return (
    <FeatureFlagContext.Provider value={{ flags, isEnabled, loading }}>
      {children}
    </FeatureFlagContext.Provider>
  );
};

/**
 * Hook to access all feature flags
 *
 * @example
 * const { isEnabled, flags, loading } = useFeatureFlags();
 * if (isEnabled('enterprise_organizations')) { ... }
 */
// eslint-disable-next-line react-refresh/only-export-components
export const useFeatureFlags = (): FeatureFlagContextValue => {
  const context = useContext(FeatureFlagContext);

  if (!context) {
    throw new Error(APP_ERRORS.FEATURE_FLAGS_CONTEXT_REQUIRED);
  }

  return context;
};
