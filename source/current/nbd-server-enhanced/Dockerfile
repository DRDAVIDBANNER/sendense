FROM ubuntu:20.04

# Prevent interactive prompts during package installation
ENV DEBIAN_FRONTEND=noninteractive
ENV TZ=UTC

# Install NBDX build dependencies
RUN apt-get update && apt-get install -y \
    build-essential \
    pkg-config \
    libglib2.0-dev \
    autotools-dev \
    autoconf \
    automake \
    libtool \
    git \
    wget \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Create build directory
WORKDIR /build

# Download NBD server source (version 3.24 - matching system version)
RUN wget http://archive.ubuntu.com/ubuntu/pool/universe/n/nbd/nbd_3.24.orig.tar.xz && \
    tar -xf nbd_3.24.orig.tar.xz

# Copy our enhanced source and patch
COPY nbd-server.c /build/nbd-3.24/
COPY nbd-server-cache-flush.patch /build/

# Apply MigrateKit cache flush enhancement patch
WORKDIR /build/nbd-3.24
RUN patch -p1 < /build/nbd-server-cache-flush.patch

# Build NBDX (enhanced NBD server)
RUN ./autogen.sh && \
    ./configure --enable-syslog --prefix=/usr && \
    make nbd-server

# Create output directory and copy NBDX binary
RUN mkdir -p /output && \
    cp nbd-server /output/nbdx && \
    chmod +x /output/nbdx

# Add version identification
RUN echo "NBDX v1.0.0-migratekit-enhanced (built $(date))" > /output/nbdx-version.txt

# Display build success
RUN echo "🎉 NBDX build completed successfully!" && \
    ls -la /output/nbdx && \
    /output/nbdx --version || true

WORKDIR /output
CMD ["echo", "NBDX build complete - enhanced NBD server ready for deployment"]
