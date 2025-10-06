export const API_BASE_URL = window.location.origin;

export const POLLING_INTERVALS = {
  SANDBOX_STATUS: 60000, // 60 seconds
  FILE_CHANGES: 10000,  // 10 seconds
  HEALTH_CHECK: 30000   // 30 seconds
} as const;

export const UI_CONSTANTS = {
  SIDEBAR_WIDTH: 300,
  RIGHT_PANEL_WIDTH: 250,
  TERMINAL_MIN_HEIGHT: 100,
  TERMINAL_MAX_HEIGHT: 600,
  TERMINAL_DEFAULT_HEIGHT: 200
} as const;

export const COLORS = {
  PRIMARY: '#2563eb',
  SUCCESS: '#059669',
  WARNING: '#d97706',
  ERROR: '#dc2626',
  NEUTRAL: '#6b7280'
} as const;

export const SANDBOX_STATES = {
  RUNNING: 'running',
  STOPPED: 'stopped',
  STARTING: 'starting',
  STOPPING: 'stopping',
  ERROR: 'error'
} as const;

export const LOG_TYPES = {
  STDOUT: 0,
  STDERR: 1,
  STATUS: 2,
  ERROR: 3
} as const;