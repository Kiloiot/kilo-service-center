/* eslint-disable react-refresh/only-export-components */
/**
 * Test Utilities
 *
 * Custom render function that wraps components with all necessary providers.
 * Use this instead of @testing-library/react's render for components that
 * depend on context providers.
 *
 * This ensures tests exercise the real provider stack (Theme, Org, FeatureFlag, Router)
 * rather than mocks, providing confidence that the component integration works correctly.
 */

import type { ReactElement, ReactNode } from "react";
import { MemoryRouter } from "react-router-dom";

import { render, type RenderOptions } from "@testing-library/react";

import { FeatureFlagProvider } from "@contexts/FeatureFlagContext";
import { OrganizationProvider } from "@contexts/OrganizationContext";
import { SystemProvider } from "@contexts/SystemContext";
import { KCThemeProvider } from "@theme/ThemeContext";

interface ProvidersProps {
  children: ReactNode;
  initialRoute?: string;
}

/**
 * Wrapper component that includes all application providers.
 * Uses MemoryRouter for controlled routing in tests.
 */
function AllProviders({ children, initialRoute = "/" }: ProvidersProps) {
  return (
    <MemoryRouter initialEntries={[initialRoute]}>
      <KCThemeProvider>
        <SystemProvider>
          <OrganizationProvider>
            <FeatureFlagProvider>{children}</FeatureFlagProvider>
          </OrganizationProvider>
        </SystemProvider>
      </KCThemeProvider>
    </MemoryRouter>
  );
}

interface CustomRenderOptions extends Omit<RenderOptions, "wrapper"> {
  initialRoute?: string;
}

/**
 * Custom render function that wraps the component with all providers.
 * Supports setting initial route for testing route-dependent components.
 *
 * @example
 * import { renderWithProviders, screen } from '@/test/utils';
 *
 * test('renders component', () => {
 *   renderWithProviders(<MyComponent />);
 *   expect(screen.getByText('Hello')).toBeInTheDocument();
 * });
 *
 * test('renders with specific route', () => {
 *   renderWithProviders(<App />, { initialRoute: '/base-stations' });
 *   expect(screen.getByTestId('base-stations-page')).toBeInTheDocument();
 * });
 */
function renderWithProviders(
  ui: ReactElement,
  options: CustomRenderOptions = {},
) {
  const { initialRoute, ...renderOptions } = options;
  return render(ui, {
    wrapper: ({ children }) => (
      <AllProviders initialRoute={initialRoute}>{children}</AllProviders>
    ),
    ...renderOptions,
  });
}

// Re-export everything from @testing-library/react
export * from "@testing-library/react";
export { userEvent } from "@testing-library/user-event";

// Export custom render as both named and default
export { renderWithProviders };
export { renderWithProviders as render };
