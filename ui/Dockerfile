# Build the Vite app, then serve it statically via nginx. nginx also reverse-
# proxies /api and /health to the backend so the browser talks to a single
# origin (no CORS, SSE works).
FROM node:22-alpine AS build
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci
COPY . .
# Empty base → api.ts uses relative URLs, which nginx proxies to the backend.
ENV VITE_API_BASE=""
RUN npm run build

FROM nginx:1.27-alpine
COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
