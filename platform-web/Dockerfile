# Build Stage
FROM oven/bun:1-alpine AS builder

WORKDIR /app

# Copy package files
COPY package.json package-lock.json ./

# Install dependencies
RUN bun install

# Copy source code
COPY . .

# Build with baked-in API URL
ENV VITE_API_URL=/api
RUN bun run build

# Serve Stage
FROM nginx:alpine

# Copy build artifacts
COPY --from=builder /app/dist /usr/share/nginx/html

# Copy custom nginx config
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
