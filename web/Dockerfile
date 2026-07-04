# syntax=docker/dockerfile:1.7

# Builder: install deps (cached layer) and run vite build → ./dist
FROM node:22-alpine AS builder

WORKDIR /src

# Cache deps first — package-lock.json is the source of truth, so `npm ci`
# is reproducible and faster than `npm install` in CI.
COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund

# Copy the rest of the source and build the production bundle.
COPY . .
RUN npm run build

# Runtime: nginx-unprivileged binds 8080 as non-root, matching the
# backend Dockerfile's non-root stance.
FROM nginxinc/nginx-unprivileged:1.27-alpine

# SPA-aware nginx config: try_files falls back to index.html so future
# client-side routes resolve on direct hits / refresh.
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Static assets from builder stage. Default nginx html dir.
COPY --from=builder /src/dist /usr/share/nginx/html

EXPOSE 8080
