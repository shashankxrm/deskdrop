import express from 'express';
import cors from 'cors';
import dotenv from 'dotenv';
import { createServer } from 'http';
import { Server } from 'socket.io';
import mongoose from 'mongoose';
import { fileURLToPath } from 'url';
import { dirname, join } from 'path';
import { connectRedis } from './services/redisService.js';
import { setupSocketIO } from './socket/socketHandler.js';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

// Import routes
import linksRouter from './routes/links.js';
import authRouter from './routes/auth.js';
import devicesRouter from './routes/devices.js';

// Load environment variables
dotenv.config();

const app = express();
const httpServer = createServer(app);

// Configure Socket.IO
const io = new Server(httpServer, {
  cors: {
    origin: '*',
    methods: ['GET', 'POST']
  }
});

// Store io instance in app for use in routes
app.set('io', io);

// Setup Socket.IO handlers
setupSocketIO(io);

// Middleware
app.use(cors());
app.use(express.json());
app.use(express.urlencoded({ extended: true }));

// Serve static files (for web auth page)
app.use(express.static(join(__dirname, '../public')));

// Serve auth page at root
app.get('/', (req, res) => {
  res.sendFile(join(__dirname, '../public/auth.html'));
});

// Health check endpoint
app.get('/api/health', (req, res) => {
  res.json({
    status: 'ok',
    timestamp: new Date().toISOString(),
    service: 'DeskDrop Backend'
  });
});

// API Routes
app.use('/api/links', linksRouter);
app.use('/api/auth', authRouter);
app.use('/api/devices', devicesRouter);

// Connect to MongoDB
const connectMongoDB = async () => {
  try {
    const mongoUri = process.env.MONGODB_URI || 'mongodb://localhost:27017/deskdrop';
    await mongoose.connect(mongoUri);
    console.log('MongoDB connected successfully');
  } catch (error) {
    console.error('MongoDB connection error:', error);
    process.exit(1);
  }
};

// Connect to Redis
const initializeServices = async () => {
  try {
    await connectMongoDB();
    await connectRedis();
  } catch (error) {
    console.error('Failed to initialize services:', error);
    process.exit(1);
  }
};

// Start server
const PORT = process.env.PORT || 3000;

initializeServices().then(() => {
  httpServer.listen(PORT, () => {
    console.log(`DeskDrop Backend server running on port ${PORT}`);
    console.log(`Health check: http://localhost:${PORT}/api/health`);
  });
});

// Graceful shutdown
process.on('SIGTERM', async () => {
  console.log('SIGTERM received, shutting down gracefully');
  httpServer.close(() => {
    console.log('HTTP server closed');
    process.exit(0);
  });
});

