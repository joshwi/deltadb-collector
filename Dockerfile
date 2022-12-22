# Pull alpine image for golang
FROM golang:1.16-alpine
# Enable go modules
RUN go env -w GO111MODULE=on
# Add git to docker image
RUN apk add --no-cache git
# Set working directory for docker image
WORKDIR /root
# Copy files to directory
COPY . .
# Install go module dependencies
RUN go mod tidy
# Remove any existing go binaries
RUN rm -rf /root/app/builds
# Run build script
RUN sh scripts/build.sh
# Change permissions for scripts
RUN chmod 755 /root/scripts/*
RUN chmod 755 /root/app/builds/*
# Load the scheduled scripts into crontab
RUN /usr/bin/crontab /root/app/cron/cron.txt
# Start cron
CMD ["sh", "scripts/start.sh"]