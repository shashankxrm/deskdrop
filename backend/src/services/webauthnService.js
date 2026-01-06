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
export async function generateRegistrationChallenge(userId, userName, existingCredentials = []) {
  try {
    const options = await generateRegistrationOptions({
      rpName,
      rpID,
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
        userVerification: 'preferred',
        requireResidentKey: false
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
    const verification = await verifyRegistrationResponse({
      response,
      expectedChallenge: challenge,
      expectedOrigin: origin,
      expectedRPID: rpID,
      requireUserVerification: false
    });
    
    if (verification.verified && verification.registrationInfo) {
      // Store new credential
      const newCredential = {
        credentialID: Buffer.from(verification.registrationInfo.credentialID),
        credentialPublicKey: Buffer.from(verification.registrationInfo.credentialPublicKey),
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
export async function generateAuthenticationChallenge(userId) {
  try {
    const user = await User.findOne({ userId });
    if (!user || !user.credentials || user.credentials.length === 0) {
      throw new Error('No credentials found for user');
    }
    
    const options = await generateAuthenticationOptions({
      rpID,
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
    const verification = await verifyAuthenticationResponse({
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

