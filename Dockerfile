# Stage 1: Build CSS
FROM oven/bun:1 AS css
WORKDIR /app
COPY package.json bun.lock* ./
RUN bun install --frozen-lockfile
COPY input.css ./
COPY templates/ ./templates/
COPY index.html* ./
RUN bun run css:build

# Stage 2: Build Go binary
FROM golang:1.25 AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=css /app/output.css ./output.css
RUN CGO_ENABLED=0 go build -o wingmodels ./cmd/wingmodels

# Stage 3: Runtime
FROM gcr.io/distroless/static-debian12
COPY --from=build /app/wingmodels /wingmodels
COPY --from=build /app/build/ /build/
COPY --from=build /app/output.css /output.css
COPY --from=build /app/public/ /public/
COPY --from=build /app/templates/ /templates/
COPY --from=build /app/index.html* /
ENTRYPOINT ["/wingmodels", "serve"]
