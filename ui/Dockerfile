# Build the Vite app, then serve it statically via nginx. nginx also reverse-
# proxies /api and /health to the backend so the browser talks to a single
# origin (no CORS, SSE works).
FROM node:22-alpine@sha256:fe4f6d74e20370acb04ad889077d14d0f64a9c8ea6a42bc60cddf1e844699914 AS build
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci
COPY . .
# Empty base → api.ts uses relative URLs, which nginx proxies to the backend.
ENV VITE_API_BASE=""
RUN npm run build

# Distroless, non-root; listens on 8080.
FROM cgr.dev/chainguard/nginx@sha256:9d34c0a9f349eaf64ec75b3e8c09774fa234152a03cee94121e09e7220e2a560
COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 8080
