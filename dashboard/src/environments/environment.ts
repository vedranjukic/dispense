export const environment = {
  production: false,
  apiBaseUrl: window.location.origin,
  wsBaseUrl: `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}`,
  enableDebugLogs: true,
  version: '1.0.0'
};