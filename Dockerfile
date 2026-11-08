FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/elayandesa ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/elayandesa-bot ./cmd/bot

FROM alpine:3.22
# Dipecah per paket + retry agar tahan koneksi lambat/putus; layer yang sukses tersimpan di cache.
RUN for i in 1 2 3 4 5 6; do apk add --no-cache ca-certificates && break || sleep 5; done
RUN for i in 1 2 3 4 5 6; do apk add --no-cache texlive && break || sleep 5; done
RUN for i in 1 2 3 4 5 6; do apk add --no-cache texmf-dist-fontsrecommended && break || sleep 5; done
RUN for i in 1 2 3 4 5 6; do apk add --no-cache texmf-dist-latexextra && break || sleep 5; done
RUN mkdir -p /var/lib/elayan/private && chown -R 10001:10001 /var/lib/elayan
WORKDIR /app
COPY --from=build /out/elayandesa /app/elayandesa
COPY --from=build /out/elayandesa-bot /app/elayandesa-bot
EXPOSE 8080
USER 10001:10001
ENTRYPOINT ["/app/elayandesa"]