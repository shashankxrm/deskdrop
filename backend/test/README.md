# Backend Tests

This directory contains all backend tests using Vitest.

## Setup

Install dependencies:
```bash
npm install
```

## Running Tests

```bash
# Run tests once
npm test

# Run tests in watch mode
npm run test:watch

# Run tests with coverage
npm run test:coverage
```

## Test Structure

- `helpers.js` - Test utilities and helper functions
- `setup.js` - Test environment setup/teardown
- `services/` - Unit tests for services
- `routes/` - Integration tests for API routes

## Test Coverage Goals

- **Lines**: 80%+
- **Functions**: 80%+
- **Branches**: 80%+
- **Statements**: 80%+

## Writing Tests

### Example Unit Test

```javascript
import { describe, it, expect, beforeEach } from 'vitest';
import { myFunction } from '../../src/services/myService.js';
import { cleanupTestData } from '../helpers.js';

describe('myService', () => {
  beforeEach(async () => {
    await cleanupTestData();
  });

  it('should do something', async () => {
    const result = await myFunction();
    expect(result).toBe(expected);
  });
});
```

### Example Integration Test

```javascript
import { describe, it, expect, beforeEach } from 'vitest';
import express from 'express';
import request from 'supertest';
import myRouter from '../../src/routes/myRoute.js';

describe('POST /api/myroute', () => {
  let app;

  beforeEach(() => {
    app = express();
    app.use(express.json());
    app.use('/api/myroute', myRouter);
  });

  it('should handle request', async () => {
    const response = await request(app)
      .post('/api/myroute')
      .send({ data: 'test' });

    expect(response.status).toBe(200);
  });
});
```

## Mocking

- **Redis**: Redis operations are mocked in tests
- **Socket.IO**: Use `createMockSocketIO()` helper
- **Database**: Uses in-memory MongoDB (MongoMemoryServer)

## Notes

- Tests use in-memory MongoDB, so no real database connection needed
- Redis is mocked, so no Redis server needed for tests
- All test data is cleaned up between tests

