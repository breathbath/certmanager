FROM golang:1.24-alpine AS build-env

RUN mkdir -p /build

WORKDIR /build
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .

ARG version
# Build
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
        -o /build/certmanager \
        -ldflags \
        "-X git.vrsal.cc/alex/iros/pkg/cmd.Version=$version" \
        main.go

# -------------
# Image creation stage

FROM alpine:latest
RUN apk --no-cache add ca-certificates shadow

WORKDIR /app

RUN addgroup -g 1002 admin
RUN adduser -D -u 1002 -G admin admin
USER admin

COPY --from=build-env /build/certmanager /app/

CMD /app/certmanager
