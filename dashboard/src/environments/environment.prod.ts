export const environment = {
  production: true,
  apiBaseUrl: window.location.origin,
  wsBaseUrl: `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}`,
  enableDebugLogs: false,
  version: '1.0.0'
};