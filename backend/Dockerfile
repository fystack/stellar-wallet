# Backend API. sqlite driver is modernc.org/sqlite (pure Go), so CGO is off
# and the final image can be a distroless static base.
FROM golang:1.26@sha256:672c944cfd07599b5f1bb179869270cdc94c6254e41db4d431259693cb286b82 AS build
WORKDIR /src
ENV CGO_ENABLED=0
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN mkdir /data && go build -trimpath -ldflags="-s -w" -o /out/stellar-wallet-backend .

FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab
WORKDIR /app
COPY --from=build /out/stellar-wallet-backend /app/stellar-wallet-backend
# Owned by nonroot (65532) so the backend_data named volume inherits it.
COPY --from=build --chown=65532:65532 /data /data
# config.yaml + event_initiator.key + wallet.db are provided at runtime via
# compose mounts/volumes.
EXPOSE 8090
ENTRYPOINT ["/app/stellar-wallet-backend"]
