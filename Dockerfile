FROM golang:1.22-alpine AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /nexus-agent .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /nexus-agent /usr/local/bin/nexus-agent
ENTRYPOINT ["nexus-agent"]
