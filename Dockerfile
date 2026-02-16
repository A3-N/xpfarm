# Base image
# Using latest to ensure support for modern toolchains (e.g. subfinder needs 1.24+)
FROM golang:1.25.5-bookworm

# Set working directory
WORKDIR /app

# Install System Dependencies
# nmap: Network scanner
# chromium: Headless browser for Katana/Gowitness
# libpcap-dev: Required for Naabu
# git: For go install
RUN apt-get update && apt-get install -y \
    nmap \
    chromium \
    libpcap-dev \
    git \
    && rm -rf /var/lib/apt/lists/*

# Pre-install ProjectDiscovery Tools (and others)
# This saves time on container startup and ensures tools are ready.
RUN go install -v github.com/projectdiscovery/subfinder/v2/cmd/subfinder@latest
RUN go install -v github.com/projectdiscovery/naabu/v2/cmd/naabu@latest
RUN go install -v github.com/projectdiscovery/httpx/cmd/httpx@latest
RUN go install -v github.com/projectdiscovery/katana/cmd/katana@latest
RUN go install -v github.com/projectdiscovery/uncover/cmd/uncover@latest
RUN go install -v github.com/projectdiscovery/urlfinder/cmd/urlfinder@latest
RUN go install -v github.com/projectdiscovery/nuclei/v3/cmd/nuclei@latest
RUN go install -v github.com/projectdiscovery/cvemap/cmd/vulnx@latest
RUN go install -v github.com/sensepost/gowitness@latest
RUN go install -v github.com/projectdiscovery/wappalyzergo/cmd/update-fingerprints@latest

# Copy Go Modules files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy Source Code
COPY . .

# Build the Application
RUN go build -o xpfarm main.go

# Expose Port
EXPOSE 8888

# Environment Variables
# Ensure Go bin is in PATH (it usually is in golang images, but explicit is good)
ENV PATH="/go/bin:${PATH}"

# Run the application
CMD ["./xpfarm"]
