FROM golang:1.16-alpine
RUN go env -w GO111MODULE=on
RUN apk add --no-cache git

# Set working directory for docker image
WORKDIR /deltadb

# Copy go.mod and go.sum
COPY go.mod .
COPY go.sum .

# Copy go code and shell scripts
COPY app app/
COPY scripts scripts/
COPY config config/

# Install go module dependencies
RUN go mod tidy

RUN rm -rf /deltadb/app/builds

# Run build script
RUN sh scripts/build.sh

RUN chmod 755 /deltadb/scripts/*
RUN chmod 755 /deltadb/app/builds/*

RUN /usr/bin/crontab /deltadb/config/cron.txt

CMD ["sh", "scripts/start.sh"]