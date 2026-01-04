import { app, BrowserWindow, Tray, Menu, nativeImage, Notification, shell } from 'electron';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';
import { setupSocketService } from './services/socketService.js';
import { setupClipboardService } from './services/clipboardService.js';
import { setupTrayManager } from './utils/trayManager.js';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

let mainWindow = null;
let tray = null;
let socketService = null;
let clipboardService = null;
let trayManager = null;

// Keep app running in background
app.dock?.hide(); // Hide from dock on macOS

function createWindow() {
  // Create a hidden window (we'll use tray instead)
  mainWindow = new BrowserWindow({
    width: 400,
    height: 300,
    show: false, // Don't show window
    webPreferences: {
      nodeIntegration: true,
      contextIsolation: false
    }
  });

  // Load a simple HTML page (optional, for settings)
  // mainWindow.loadFile('src/index.html');
  
  // Don't show window
  mainWindow.hide();
}

async function initializeServices() {
  // Initialize clipboard service
  clipboardService = setupClipboardService();
  
  // Initialize tray manager first
  trayManager = setupTrayManager({
    onQuit: () => {
      app.quit();
    },
    onReconnect: () => {
      if (socketService) {
        socketService.reconnect();
      }
    }
  });
  
  // Initialize socket service (async - loads config file)
  socketService = await setupSocketService({
    onLinkReceived: (linkData) => {
      handleLinkReceived(linkData);
    },
    onConnectionStatusChange: (status, message) => {
      if (trayManager) {
        trayManager.updateConnectionStatus(status, message);
      }
    }
  });
}

function handleLinkReceived(linkData) {
  const { url } = linkData;
  
  console.log('Link received:', url);
  
  // Copy to clipboard
  if (clipboardService) {
    clipboardService.writeText(url);
    console.log('Link copied to clipboard');
    
    // Show notification
    if (Notification.isSupported()) {
      const notification = new Notification({
        title: 'DeskDrop - Link Received',
        body: `Copied to clipboard: ${url.substring(0, 50)}...`,
        icon: join(__dirname, '../assets/icon.png')
      });
      notification.show();
    }
    
    // Optional: Auto-open in browser
    shell.openExternal(url);
    console.log('Link opened in browser');
  }
}

app.whenReady().then(async () => {
  createWindow();
  await initializeServices();
  
  app.on('activate', () => {
    // On macOS, re-create window when dock icon is clicked
    if (BrowserWindow.getAllWindows().length === 0) {
      createWindow();
    }
  });
});

app.on('window-all-closed', () => {
  // Don't quit on macOS when all windows are closed
  // Keep running in background (tray)
  if (process.platform !== 'darwin') {
    app.quit();
  }
});

app.on('before-quit', () => {
  // Cleanup
  if (socketService) {
    socketService.disconnect();
  }
});

