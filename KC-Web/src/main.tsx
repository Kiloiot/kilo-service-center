import React from 'react';
import ReactDOM from 'react-dom/client';

import { ErrorBoundary } from '@components/common/ErrorBoundary';
import { KCThemeProvider } from '@theme/ThemeContext';

import App from './App';

import './index.css';

const rootElement = document.getElementById('root');

if (!rootElement) {
  throw new Error('Failed to find the root element');
}

const root = ReactDOM.createRoot(rootElement);

root.render(
  <React.StrictMode>
    <ErrorBoundary boundary="root">
      <KCThemeProvider>
        <App />
      </KCThemeProvider>
    </ErrorBoundary>
  </React.StrictMode>
);
