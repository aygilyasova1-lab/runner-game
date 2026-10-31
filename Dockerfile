FROM golang:1.26-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./server

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/server /server

WORKDIR /data
EXPOSE 50051
ENTRYPOINT ["/server"]