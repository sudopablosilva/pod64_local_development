# Multi-stage build for React frontend and Node.js backend
FROM node:22-alpine AS frontend-build

# Set working directory for frontend
WORKDIR /app/frontend

# Copy frontend package files
COPY frontend/package*.json ./

# Install frontend dependencies with security audit
RUN npm ci --omit=dev && (npm audit fix --audit-level=high || echo "Some vulnerabilities require breaking changes - continuing build")

# Copy frontend source code
COPY frontend/ ./

# Build the React application
RUN npm run build

# Production stage
FROM node:22-alpine AS production

# Set working directory
WORKDIR /app

# Install dumb-init for proper signal handling
RUN apk add --no-cache dumb-init

# Create non-root user
RUN addgroup -g 1001 -S nodejs && \
    adduser -S dashboard -u 1001

# Copy package files
COPY package*.json ./

# Install production dependencies with security audit
RUN npm ci --omit=dev && npm audit fix --audit-level=high && npm cache clean --force

# Copy server source code
COPY server.js ./

# Copy built frontend from previous stage
COPY --from=frontend-build /app/frontend/build ./frontend/build

# Change ownership to non-root user
RUN chown -R dashboard:nodejs /app
USER dashboard

# Expose port
EXPOSE 3000

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD node -e "require('http').get('http://localhost:3000/health', (res) => { \
    process.exit(res.statusCode === 200 ? 0 : 1) \
  }).on('error', () => process.exit(1))"

# Start the application with dumb-init
ENTRYPOINT ["dumb-init", "--"]
CMD ["node", "server.js"]
