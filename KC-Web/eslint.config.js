// Re-export the shared kc-web-dev contributor harness ESLint config so flat-config
// base-path resolution treats KC-Web/ as its own base (ESLint 9 ignores files
// located outside the config file's directory). Plugins resolve relative to the
// kc-web-dev import path, which is where they're installed.
export { default } from '../../kc-web-dev/eslint.config.js';
