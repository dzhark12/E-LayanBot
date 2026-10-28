FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/elayandesa ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/elayandesa-bot ./cmd/bot

FROM alpine:3.22
RUN apk add --no-cache ca-certificates texlive texmf-dist-latexextra texmf-dist-fontsrecommended && mkdir -p /var/lib/elayan/private && chown -R 10001:10001 /var/lib/elayan
WORKDIR /app
COPY --from=build /out/elayandesa /app/elayandesa
COPY --from=build /out/elayandesa-bot /app/elayandesa-bot
EXPOSE 8080
USER 10001:10001
ENTRYPOINT ["/app/elayandesa"]
