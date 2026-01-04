import { app, Tray, Menu, nativeImage } from 'electron';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';
import { existsSync } from 'fs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

let tray = null;
let connectionStatus = 'disconnected';
let reconnectCallback = null;

export const setupTrayManager = (callbacks) => {
  reconnectCallback = callbacks?.onReconnect || null;

  // macOS menu-bar tray icon (MUST be template image)
  const iconPath = join(__dirname, '../../assets/trayTemplate.png');

  if (!existsSync(iconPath)) {
    throw new Error(
      'Tray icon not found. Expected: assets/trayTemplate.png (monochrome, transparent)'
    );
  }

  const trayIcon = nativeImage
    .createFromPath(iconPath)
    .resize({ width: 16 });

  // CRITICAL for macOS menu bar visibility
  trayIcon.setTemplateImage(true);

  tray = new Tray(trayIcon);
  tray.setToolTip('DeskDrop');

  updateMenu();

  return {
    updateConnectionStatus: (status, message) => {
      connectionStatus = status;
      updateMenu(message);
    },
  };
};

function updateMenu(message) {
  if (!tray) return;

  const statusText = getStatusText(connectionStatus, message);

  const contextMenu = Menu.buildFromTemplate([
    {
      label: 'DeskDrop',
      enabled: false,
    },
    {
      type: 'separator',
    },
    {
      label: `Status: ${statusText}`,
      enabled: false,
    },
    {
      type: 'separator',
    },
    {
      label: 'Reconnect',
      enabled: connectionStatus !== 'connected',
      click: () => {
        if (reconnectCallback) {
          reconnectCallback();
          connectionStatus = 'connecting';
          updateMenu('Reconnecting...');
        }
      },
    },
    {
      type: 'separator',
    },
    {
      label: 'Quit',
      click: () => {
        app.quit();
      },
    },
  ]);

  tray.setContextMenu(contextMenu);
  tray.setToolTip(`DeskDrop — ${statusText}`);
}

function getStatusText(status, message) {
  switch (status) {
    case 'connected':
      return 'Connected';
    case 'connecting':
      return message || 'Connecting…';
    case 'disconnected':
      return message || 'Disconnected';
    case 'error':
      return message || 'Error';
    default:
      return 'Unknown';
  }
}
