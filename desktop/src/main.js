import { app, BrowserWindow, Notification, shell } from 'electron';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';
import { setupSocketService } from './services/socketService.js';
import { setupClipboardService } from './services/clipboardService.js';
import { setupTrayManager } from './utils/trayManager.js';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

let mainWindow = null;
let socketService = null;
let clipboardService = null;
let trayManager = null;

function createWindow() {
  mainWindow = new BrowserWindow({
    width: 400,
    height: 300,
    show: false, // tray-only app
    webPreferences: {
      nodeIntegration: true,
      contextIsolation: false,
    },
  });

  // Optional settings UI later
  // mainWindow.loadFile('src/index.html');
}

async function initializeServices() {
  clipboardService = setupClipboardService();

  // Tray MUST be initialized successfully
  trayManager = setupTrayManager({
    onQuit: () => app.quit(),
    onReconnect: () => socketService?.reconnect(),
  });

  socketService = await setupSocketService({
    onLinkReceived: handleLinkReceived,
    onConnectionStatusChange: (status, message) => {
      trayManager?.updateConnectionStatus(status, message);
    },
  });
}

function handleLinkReceived(linkData) {
  const { url } = linkData;

  clipboardService?.writeText(url);

  if (Notification.isSupported()) {
    new Notification({
      title: 'DeskDrop',
      body: 'Link copied to clipboard',
      icon: join(__dirname, '../assets/trayTemplate.png'),
    }).show();
  }

  shell.openExternal(url);
}

app.whenReady().then(async () => {
  // ✅ macOS-only: hide dock AFTER app is ready
  if (process.platform === 'darwin') {
    app.dock.hide();
  }

  createWindow();

  try {
    await initializeServices();
  } catch (err) {
    console.error('Failed to initialize tray/services:', err);
    app.quit(); // ❗ avoid ghost app
  }
});

app.on('window-all-closed', () => {
  // tray-only app → never quit automatically
});

app.on('before-quit', () => {
  socketService?.disconnect();
});
