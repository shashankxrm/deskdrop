import mongoose from 'mongoose';

const credentialSchema = new mongoose.Schema({
  credentialID: {
    type: Buffer,
    required: true
  },
  credentialPublicKey: {
    type: Buffer,
    required: true
  },
  counter: {
    type: Number,
    default: 0
  },
  deviceName: {
    type: String,
    default: 'Unknown Device'
  },
  deviceType: {
    type: String,
    enum: ['desktop', 'android', 'web'],
    default: 'desktop'
  },
  platform: {
    type: String // 'macos', 'windows', 'linux', 'android', 'ios'
  },
  createdAt: {
    type: Date,
    default: Date.now
  },
  lastUsed: {
    type: Date,
    default: Date.now
  }
}, { _id: true });

const refreshTokenSchema = new mongoose.Schema({
  token: {
    type: String,
    required: true
  },
  expiresAt: {
    type: Date,
    required: true
  },
  createdAt: {
    type: Date,
    default: Date.now
  }
}, { _id: true });

const userSchema = new mongoose.Schema({
  userId: {
    type: String,
    required: true,
    unique: true,
    index: true
  },
  email: {
    type: String,
    required: true,
    unique: true,
    index: true
  },
  credentials: {
    type: [credentialSchema],
    default: []
  },
  refreshTokens: {
    type: [refreshTokenSchema],
    default: []
  },
  createdAt: {
    type: Date,
    default: Date.now
  }
});

export const User = mongoose.model('User', userSchema);

