# DeskDrop Android App

Native Android app (Kotlin) that receives shared job links and sends them to the DeskDrop backend.

## Setup

1. Open the project in Android Studio
2. Update `ApiClient.kt` with your backend URL:
   - For emulator: `http://10.0.2.2:3000/api/` (default)
   - For physical device: `http://<your-computer-ip>:3000/api/`
3. Update dev token in `ApiClient.kt` or via SharedPreferences
4. Build and run the app

## Features

- **Share Target**: Appears in Android share menu for text/URL sharing
- **Link Submission**: Sends shared URLs to backend via HTTP POST
- **Simple UI**: Minimal confirmation screen with status feedback

## Configuration

### Backend URL
Edit `app/src/main/java/com/deskdrop/app/api/ApiClient.kt`:
```kotlin
private const val BASE_URL = "http://your-backend-url:3000/api/"
```

### Dev Token
The app uses a dev token for authentication. Set it in SharedPreferences or update the default in `ApiClient.kt`.

## Usage

1. Share a job link from any app (browser, LinkedIn, etc.)
2. Select "DeskDrop" from the share menu
3. Confirm the link and tap "Send to Desktop"
4. Link is sent to backend and delivered to desktop app

## Build

```bash
./gradlew assembleDebug
```

Install on device:
```bash
./gradlew installDebug
```

