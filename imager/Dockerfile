FROM ubuntu:24.04

RUN apt-get update && apt-get install -y \
    btrfs-progs \
    dosfstools \
    parted \
    qemu-utils \
    qemu-user-static \
    wget \
    wpasupplicant \
    xz-utils \
    rsync \
    util-linux \
    openssl \
    && rm -rf /var/lib/apt/lists/*

RUN which wpa_passphrase

WORKDIR /work
COPY build.sh /work/build.sh
COPY build-orangepi.sh /work/build-orangepi.sh
COPY lib /work/lib
RUN chmod +x /work/build.sh /work/build-orangepi.sh

# Default to OrangePi build; the Makefile overrides at run-time by appending
# /work/build.sh when TARGET=rpi.
CMD ["/work/build-orangepi.sh"]