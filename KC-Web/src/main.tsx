import React from "react";
import ReactDOM from "react-dom/client";

import { ErrorBoundary } from "@components/common/ErrorBoundary";
import { APP_ROOT_ELEMENT_ID } from "@constants/app";
import { APP_ERRORS } from "@constants/messages";
import { KCThemeProvider } from "@theme/ThemeContext";

import App from "./App";

import "./index.css";

const rootElement = document.getElementById(APP_ROOT_ELEMENT_ID);

if (!rootElement) {
  throw new Error(APP_ERRORS.ROOT_ELEMENT_MISSING);
}

const root = ReactDOM.createRoot(rootElement);

root.render(
  <React.StrictMode>
    <ErrorBoundary boundary="root">
      <KCThemeProvider>
        <App />
      </KCThemeProvider>
    </ErrorBoundary>
  </React.StrictMode>,
);
