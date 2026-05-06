# =============================================================================
# Stage 1: Build
# =============================================================================
FROM node:22-alpine AS builder

WORKDIR /build

RUN corepack enable && corepack prepare pnpm@10.33.3 --activate

COPY pnpm-lock.yaml pnpm-workspace.yaml package.json ./
COPY web/package.json ./web/package.json
RUN pnpm install --frozen-lockfile

COPY web ./web
WORKDIR /build/web
RUN pnpm run build

# =============================================================================
# Stage 2: Serve with Nginx
# =============================================================================
FROM nginx:alpine

# Remove default config
RUN rm /etc/nginx/conf.d/default.conf

COPY web/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /build/web/dist /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
