# syntax=docker/dockerfile:1

FROM node:22-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .

# Client DSN is embedded for browser Sentry; source-map upload uses auth token at build time.
ARG PUBLIC_SENTRY_DSN=
ARG SENTRY_ORG=
ARG SENTRY_PROJECT=humbleshare-ui
ARG SENTRY_AUTH_TOKEN=
ARG SENTRY_RELEASE=
ENV PUBLIC_SENTRY_DSN=$PUBLIC_SENTRY_DSN \
    SENTRY_ORG=$SENTRY_ORG \
    SENTRY_PROJECT=$SENTRY_PROJECT \
    SENTRY_AUTH_TOKEN=$SENTRY_AUTH_TOKEN \
    SENTRY_RELEASE=$SENTRY_RELEASE

RUN npm run build

FROM node:22-alpine
WORKDIR /app
ENV NODE_ENV=production
COPY --from=builder /app/build ./build
COPY --from=builder /app/package.json ./
COPY --from=builder /app/package-lock.json ./
COPY --from=builder /app/node_modules ./node_modules
USER node
EXPOSE 3000
ENV PORT=3000
ENV HOST=0.0.0.0
CMD ["node", "build"]
