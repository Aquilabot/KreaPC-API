FROM golang:1.22 as modules
COPY go.mod go.sum /modules/
WORKDIR /modules
RUN go mod download

# Stage 2: Build
FROM golang:1.22 as builder
COPY --from=modules /go/pkg /go/pkg
COPY . /workdir
WORKDIR /workdir
RUN PWGO_VER=$(grep -oE "playwright-go v\S+" /workdir/go.mod | sed 's/playwright-go //g') \
    && go install github.com/playwright-community/playwright-go/cmd/playwright@${PWGO_VER}
RUN GOOS=linux GOARCH=amd64 go build -o /bin/kreapc

# Stage 3: Final
FROM ubuntu:jammy
COPY --from=builder /go/bin/playwright /bin/kreapc /
RUN apt-get update && apt-get install -y ca-certificates tzdata \
    && /playwright install --with-deps \
    && rm -rf /var/lib/apt/lists/*
EXPOSE 4321
CMD ["/kreapc"]