# =============================================================================
# imager Makefile
# =============================================================================
# Targets:
#   make build                 — RPi 5 golden image (default)
#   make TARGET=opi build      — OrangePi (Armbian) golden image (skeleton)
#   make sd-card-list          — list disks for flashing
#   make sd-card-test DISK=N   — test SD card with 100 MB dd
#   make sd-card-flash DISK=N  — flash golden.img to /dev/diskN
#   make sd-card-flash-opi DISK=N — flash golden-opi.img to /dev/diskN
# =============================================================================

TARGET ?= rpi
BUILD_SCRIPT_rpi := build.sh
BUILD_SCRIPT_opi := build-orangepi.sh
BUILD_SCRIPT := $(BUILD_SCRIPT_$(TARGET))

ifndef BUILD_SCRIPT
$(error Unknown TARGET=$(TARGET). Supported: rpi, opi)
endif

# Image filename per target so RPi + OPi builds don't clobber each other.
IMG_FILE_rpi := golden.img
IMG_FILE_opi := golden-opi.img
IMG_FILE := $(IMG_FILE_$(TARGET))

build:
	# Build the Docker build environment (shared for both targets)
	docker build -t pi-builder .

	# Run it — needs --privileged for losetup/mount.
	# BUILD_SCRIPT controls which builder runs inside the container.
	docker run --rm \
	--privileged \
	-v ./output:/output \
	-v ./input:/input \
	-e BUILD_SCRIPT=$(BUILD_SCRIPT) \
	pi-builder /work/$(BUILD_SCRIPT)

sd-card-list:
	diskutil list

sd-card-test:
ifndef DISK
	$(error DISK is required. Usage: make sd-card-test DISK=19)
endif
	@echo "==> Testing SD card write (100 MB of zeros)..."
	sudo launchctl unload /System/Library/LaunchDaemons/com.apple.diskarbitrationd.plist 2>/dev/null || true
	diskutil unmountDisk /dev/disk$(DISK) 2>/dev/null || true
	sleep 1
	sudo dd if=/dev/zero of=/dev/rdisk$(DISK) bs=1m count=1000 status=progress; \
	DD_EXIT=$$?; \
	sudo launchctl load /System/Library/LaunchDaemons/com.apple.diskarbitrationd.plist 2>/dev/null || true; \
	if [ $$DD_EXIT -eq 0 ]; then echo "==> SD card OK — hardware is fine"; else echo "==> SD card FAILED — hardware issue (bad card or reader)"; fi; \
	exit $$DD_EXIT

sd-card-flash:
ifndef DISK
	$(error DISK is required. Usage: make sd-card-flash DISK=4)
endif
	@echo "==> Flashing ./output/$(IMG_FILE) to /dev/disk$(DISK)..."
	diskutil unmountDisk force /dev/disk$(DISK)
	sudo dd if=./output/$(IMG_FILE) of=/dev/disk$(DISK) bs=64m conv=fsync status=progress
	sync
	diskutil eject /dev/disk$(DISK) 2>/dev/null || true
	@echo "==> Flash complete"

sd-card-flash-opi:
	$(MAKE) sd-card-flash TARGET=opi DISK=$(DISK)
