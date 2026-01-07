import { app } from 'electron';
import { readFileSync, writeFileSync, existsSync } from 'fs';
import { join } from 'path';

let backendUrl = null;

/**
 * Initialize auth service with backend URL
 */
export function initAuthService(serverUrl) {
  backendUrl = serverUrl;
}

/**
 * Get backend URL
 */
export function getBackendUrl() {
  return backendUrl;
}
