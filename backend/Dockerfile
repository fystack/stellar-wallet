# Backend API. sqlite driver is modernc.org/sqlite (pure Go), so CGO is off
# and the final image can be a distroless static base.
FROM golang:1.26 AS build
WORKDIR /src
ENV CGO_ENABLED=0
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /out/stellar-wallet-backend .

FROM gcr.io/distroless/static-debian12
WORKDIR /app
COPY --from=build /out/stellar-wallet-backend /app/stellar-wallet-backend
# config.yaml + event_initiator.key + wallet.db are provided at runtime via
# compose mounts/volumes.
EXPOSE 8090
ENTRYPOINT ["/app/stellar-wallet-backend"]
