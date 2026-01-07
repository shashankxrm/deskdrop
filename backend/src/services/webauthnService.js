import { 
  generateRegistrationOptions, 
  verifyRegistrationResponse,
  generateAuthenticationOptions,
  verifyAuthenticationResponse
} from '@simplewebauthn/server';
import { User } from '../models/User.js';

// Get WebAuthn configuration from environment
const rpName = process.env.WEBAUTHN_RP_NAME || 'DeskDrop';
const rpID = process.env.WEBAUTHN_RP_ID || 'localhost';
const origin = process.env.WEBAUTHN_ORIGIN || `http://${rpID}:3000`;

/**
 * Generate registration challenge for new passkey
 */
export async function generateRegistrationChallenge(userId, userName, existingCredentials = [], clientType = 'web') {
  try {
    // For Electron clients, use localhost as rpID
    const effectiveRPID = (clientType === 'electron' || clientType === 'desktop') ? 'localhost' : rpID;
    
    const options = await generateRegistrationOptions({
      rpName,
      rpID: effectiveRPID,
      userName,
      userID: Buffer.from(userId),
      userDisplayName: userName,
      attestationType: 'none',
      excludeCredentials: existingCredentials.map(cred => ({
        id: Buffer.from(cred.credentialID),
        type: 'public-key',
        transports: ['usb', 'nfc', 'ble', 'internal']
      })),
      authenticatorSelection: {
        authenticatorAttachment: 'platform', // Prefer platform authenticators (Touch ID, Face ID, etc.)
        userVerification: 'required',
        residentKey: 'required'
      },
      supportedAlgorithmIDs: [-7, -257] // ES256, RS256
    });
    
    return options;
  } catch (error) {
    console.error('Error generating registration challenge:', error);
    throw new Error('Failed to generate registration challenge');
  }
}

/**
 * Verify registration response and store credential
 */
export async function verifyRegistration(userId, response, challenge, deviceName, deviceType, platform) {
  try {
    // Get user's existing credentials
    const user = await User.findOne({ userId });
    if (!user) {
      throw new Error('User not found');
    }
    
    const existingCredentials = user.credentials || [];
    
    // Verify the registration response
    // For browser-based auth, try multiple origin/rpID combinations
    console.log('Verifying registration with:', {
      rpID,
      origin,
      challengeLength: challenge?.length
    });
    
    // Try different origin/rpID combinations
    const verificationAttempts = [
      { origin: origin, rpID: rpID, name: 'configured values' },
      { origin: 'http://localhost:3000', rpID: 'localhost', name: 'localhost:3000' },
      { origin: 'http://localhost', rpID: 'localhost', name: 'localhost' },
    ];
    
    let verification;
    let lastError;
    
    for (const attempt of verificationAttempts) {
      try {
        console.log(`Trying verification with ${attempt.name}...`);
        verification = await verifyRegistrationResponse({
          response,
          expectedChallenge: challenge,
          expectedOrigin: attempt.origin,
          expectedRPID: attempt.rpID,
          requireUserVerification: false
        });
        console.log(`Verification successful with ${attempt.name}`);
        break;
      } catch (error) {
        lastError = error;
        console.log(`Verification failed with ${attempt.name}:`, error.message);
        continue;
      }
    }
    
    if (!verification) {
      console.error('All verification attempts failed. Last error:', lastError);
      throw new Error(`Verification failed: ${lastError?.message || 'Unknown error'}`);
    }
    
    if (verification.verified && verification.registrationInfo) {
      // Store new credential
      // credentialID and credentialPublicKey might already be Buffers
      const credentialID = Buffer.isBuffer(verification.registrationInfo.credentialID) 
        ? verification.registrationInfo.credentialID 
        : Buffer.from(verification.registrationInfo.credentialID);
      
      const credentialPublicKey = Buffer.isBuffer(verification.registrationInfo.credentialPublicKey)
        ? verification.registrationInfo.credentialPublicKey
        : Buffer.from(verification.registrationInfo.credentialPublicKey);
      
      const newCredential = {
        credentialID: credentialID,
        credentialPublicKey: credentialPublicKey,
        counter: verification.registrationInfo.counter || 0,
        deviceName: deviceName || 'Unknown Device',
        deviceType: deviceType || 'desktop',
        platform: platform,
        createdAt: new Date(),
        lastUsed: new Date()
      };
      
      // Add credential to user
      user.credentials.push(newCredential);
      await user.save();
      
      return { 
        verified: true, 
        credential: newCredential 
      };
    }
    
    return { 
      verified: false, 
      error: 'Verification failed' 
    };
  } catch (error) {
    console.error('Error verifying registration:', error);
    throw error;
  }
}

/**
 * Generate authentication challenge for existing passkey
 */
export async function generateAuthenticationChallenge(userId, clientType = 'web') {
  try {
    const user = await User.findOne({ userId });
    if (!user || !user.credentials || user.credentials.length === 0) {
      throw new Error('No credentials found for user');
    }
    
    // For Electron clients, use localhost as rpID
    const effectiveRPID = (clientType === 'electron' || clientType === 'desktop') ? 'localhost' : rpID;
    
    const options = await generateAuthenticationOptions({
      rpID: effectiveRPID,
      allowCredentials: user.credentials.map(cred => ({
        id: Buffer.from(cred.credentialID),
        type: 'public-key',
        transports: ['usb', 'nfc', 'ble', 'internal']
      })),
      userVerification: 'preferred'
    });
    
    return options;
  } catch (error) {
    console.error('Error generating authentication challenge:', error);
    throw error;
  }
}

/**
 * Verify authentication response
 */
export async function verifyAuthentication(userId, response, challenge) {
  try {
    const user = await User.findOne({ userId });
    if (!user) {
      throw new Error('User not found');
    }
    
    // Find the credential being used
    const credentialID = Buffer.from(response.id, 'base64url');
    const credential = user.credentials.find(
      cred => cred.credentialID.equals(credentialID)
    );
    
    if (!credential) {
      throw new Error('Credential not found');
    }
    
    // Verify the authentication response
    // For Electron clients, try localhost first with multiple origin patterns
    const localhostOrigins = [
      'http://localhost',
      'http://localhost:3000',
      'http://localhost:8080',
      'http://127.0.0.1',
      'http://127.0.0.1:3000'
    ];
    
    let verification;
    let verified = false;
    
    // Try each localhost origin pattern
    for (const localOrigin of localhostOrigins) {
      try {
        verification = await verifyAuthenticationResponse({
          response,
          expectedChallenge: challenge,
          expectedOrigin: localOrigin,
          expectedRPID: 'localhost',
          credential: {
            id: credential.credentialID,
            publicKey: credential.credentialPublicKey,
            counter: credential.counter
          },
          requireUserVerification: false
        });
        console.log(`Authentication verification successful with ${localOrigin}`);
        verified = true;
        break;
      } catch (error) {
        console.log(`Verification with ${localOrigin} failed:`, error.message);
        continue;
      }
    }
    
    // If localhost didn't work, try with configured values
    if (!verified) {
      verification = await verifyAuthenticationResponse({
        response,
        expectedChallenge: challenge,
        expectedOrigin: origin,
        expectedRPID: rpID,
        credential: {
          id: credential.credentialID,
          publicKey: credential.credentialPublicKey,
          counter: credential.counter
        },
        requireUserVerification: false
      });
    }
    
    if (verification.verified && verification.authenticationInfo) {
      // Update credential counter and last used
      credential.counter = verification.authenticationInfo.newCounter;
      credential.lastUsed = new Date();
      await user.save();
      
      return { 
        verified: true,
        credential: credential
      };
    }
    
    return { 
      verified: false, 
      error: 'Authentication failed' 
    };
  } catch (error) {
    console.error('Error verifying authentication:', error);
    throw error;
  }
}

