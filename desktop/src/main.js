import { app, BrowserWindow, Notification, shell, ipcMain, dialog } from 'electron';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';
import { setupSocketService } from './services/socketService.js';
import { setupClipboardService } from './services/clipboardService.js';
import { setupTrayManager } from './utils/trayManager.js';
import { initAuthService } from './services/authService.js';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

let mainWindow = null;
let authWindow = null;
let tokenWindow = null;
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
}

async function showPairingInstructions(backendUrl) {
  // Show dialog with options
  const result = await dialog.showMessageBox({
    type: 'info',
    title: 'DeskDrop - Pairing Required',
    message: 'Pairing Token Required',
    detail: 'To connect your desktop app, you need a pairing token.\n\n1. Click "Open Auth Page" to authenticate with WebAuthn\n2. After authentication, copy the pairing token\n3. Paste it when prompted',
    buttons: ['Open Auth Page', 'I Have a Token', 'Cancel'],
    defaultId: 0,
    cancelId: 2
  });

  if (result.response === 2) {
    // Cancel
    return;
  }

  if (result.response === 0) {
    // Open auth page in external browser
    const authUrl = `${backendUrl}/auth.html`;
    shell.openExternal(authUrl);
    
    // Wait a moment, then show token input
    setTimeout(() => {
      promptForPairingToken(backendUrl);
    }, 1500);
  } else {
    // User has token, prompt for it directly
    promptForPairingToken(backendUrl);
  }
}

function promptForPairingToken(backendUrl) {
  // Create a simple input window for pairing token
  tokenWindow = new BrowserWindow({
    width: 500,
    height: 220,
    show: false,
    resizable: false,
    modal: true,
    webPreferences: {
      nodeIntegration: true,
      contextIsolation: false,
    },
  });

  const tokenHtml = `
    <!DOCTYPE html>
    <html>
    <head>
      <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
          font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
          padding: 30px;
          background: #f5f5f5;
        }
        h2 {
          margin-bottom: 15px;
          color: #333;
          font-size: 18px;
        }
        input {
          width: 100%;
          padding: 12px;
          font-size: 14px;
          border: 2px solid #ddd;
          border-radius: 6px;
          margin-bottom: 15px;
          box-sizing: border-box;
        }
        input:focus {
          outline: none;
          border-color: #667eea;
        }
        .buttons {
          display: flex;
          gap: 10px;
          justify-content: flex-end;
        }
        button {
          padding: 10px 20px;
          border: none;
          border-radius: 6px;
          cursor: pointer;
          font-size: 14px;
          font-weight: 500;
        }
        .connect {
          background: #667eea;
          color: white;
        }
        .connect:hover {
          background: #5568d3;
        }
        .cancel {
          background: #ddd;
          color: #333;
        }
        .cancel:hover {
          background: #ccc;
        }
        .error {
          color: #c33;
          font-size: 12px;
          margin-bottom: 10px;
          display: none;
        }
      </style>
    </head>
    <body>
      <h2>Enter Pairing Token</h2>
      <div class="error" id="errorMsg"></div>
      <input type="text" id="tokenInput" placeholder="Paste your pairing token here..." autofocus>
      <div class="buttons">
        <button class="cancel" onclick="cancel()">Cancel</button>
        <button class="connect" onclick="connect()">Connect</button>
      </div>
      <script>
        const { ipcRenderer } = require('electron');
        
        function cancel() {
          ipcRenderer.send('pairing-token-cancelled');
        }
        
        function connect() {
          const token = document.getElementById('tokenInput').value.trim();
          if (!token) {
            const errorDiv = document.getElementById('errorMsg');
            errorDiv.textContent = 'Please enter a pairing token';
            errorDiv.style.display = 'block';
            return;
          }
          ipcRenderer.send('pairing-token-submitted', token);
        }
        
        document.getElementById('tokenInput').addEventListener('keypress', (e) => {
          if (e.key === 'Enter') {
            connect();
          }
        });
      </script>
    </body>
    </html>
  `;

  tokenWindow.loadURL(`data:text/html;charset=utf-8,${encodeURIComponent(tokenHtml)}`);
  tokenWindow.show();
  tokenWindow.focus();
}

async function savePairingToken(token, backendUrl) {
  const { readFileSync, writeFileSync, existsSync } = await import('fs');
  const userDataPath = app.getPath('userData');
  const configPath = join(userDataPath, 'config.json');
  
  let config = {};
  if (existsSync(configPath)) {
    config = JSON.parse(readFileSync(configPath, 'utf8'));
  }
  
  config.pairingToken = token;
  config.serverUrl = backendUrl;
  
  writeFileSync(configPath, JSON.stringify(config, null, 2));
  console.log('Pairing token saved');
}

async function initializeServices() {
  clipboardService = setupClipboardService();

  // Tray MUST be initialized successfully
  trayManager = setupTrayManager({
    onQuit: () => app.quit(),
    onReconnect: () => socketService?.reconnect(),
  });

  // Initialize auth service with backend URL
  const backendUrl = process.env.BACKEND_URL || 'http://localhost:3000';
  initAuthService(backendUrl);

  // Check if user has pairing token
  const { readFileSync, existsSync } = await import('fs');
  const userDataPath = app.getPath('userData');
  const configPath = join(userDataPath, 'config.json');
  
  let hasPairingToken = false;
  if (existsSync(configPath)) {
    try {
      const config = JSON.parse(readFileSync(configPath, 'utf8'));
      hasPairingToken = !!config.pairingToken;
    } catch (error) {
      console.error('Error reading config:', error);
    }
  }

  if (!hasPairingToken) {
    // Show pairing instructions and get token from user
    await showPairingInstructions(backendUrl);
    // After pairing, initializeSocketService will be called from IPC handler
    return; // Don't initialize socket service until paired
  }

  // User has pairing token, initialize socket service
  await initializeSocketService();
}

async function initializeSocketService() {
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

// IPC handlers for pairing token
ipcMain.on('pairing-token-submitted', async (event, token) => {
  if (tokenWindow) {
    tokenWindow.close();
    tokenWindow = null;
  }
  
  const backendUrl = process.env.BACKEND_URL || 'http://localhost:3000';
  
  // Save token to config
  await savePairingToken(token, backendUrl);
  
  // Initialize socket service with new token
  try {
    await initializeSocketService();
    
    if (Notification.isSupported()) {
      new Notification({
        title: 'DeskDrop',
        body: 'Pairing successful! Connected to backend.',
        icon: join(__dirname, '../assets/trayTemplate.png'),
      }).show();
    }
  } catch (error) {
    console.error('Failed to connect after pairing:', error);
    dialog.showErrorBox('Connection Error', 'Failed to connect to backend. Please check your pairing token and try again.');
  }
});

ipcMain.on('pairing-token-cancelled', () => {
  if (tokenWindow) {
    tokenWindow.close();
    tokenWindow = null;
  }
});

app.whenReady().then(async () => {
  // ✅ macOS-only: hide dock AFTER app is ready
  if (process.platform === 'darwin') {
    app.dock.hide();
  }

  createWindow();

  try {
    await initializeServices();
  } catch (err) {
    console.error('Failed to initialize services:', err);
    // Show auth window if initialization fails
    const backendUrl = process.env.BACKEND_URL || 'http://localhost:3000';
    if (!authWindow && !tokenWindow) {
      showPairingInstructions(backendUrl);
    }
  }
});

app.on('window-all-closed', () => {
  // tray-only app → never quit automatically
});

app.on('before-quit', () => {
  socketService?.disconnect();
});
