# GGUIX Linux runtime harness. Native linux/arm64 on Apple Silicon (no emulation).
# Build:  docker build -t gguix-linux -f linux/Dockerfile linux
FROM ubuntu:24.04
ENV DEBIAN_FRONTEND=noninteractive LANG=C.UTF-8
ARG GO_VERSION=1.27.3

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates curl git jq xz-utils unzip python3 python3-gi python3-dbus rsync file procps psmisc sudo \
    build-essential pkg-config cmake \
    # GL / X11 / Wayland dev (Fyne, Cogent Core: glfw/vulkan/wgpu)
    libgl1-mesa-dev libegl1-mesa-dev libgles2-mesa-dev xorg-dev libwayland-dev libxkbcommon-dev libxkbcommon-x11-dev \
    libvulkan-dev mesa-vulkan-drivers libgl1-mesa-dri libglx-mesa0 \
    # GTK3 + GTK4 + WebKitGTK (webview_go, glaze, Wails v3 default is gtk4/webkitgtk-6.0)
    libgtk-3-dev libwebkit2gtk-4.1-dev libgtk-4-dev libwebkitgtk-6.0-dev libayatana-appindicator3-dev \
    # Qt6 for MIQT and MIQT+WebEngine
    qt6-base-dev qt6-webengine-dev qt6-webengine-dev-tools qt6-webchannel-dev qt6-wayland libqt6opengl6-dev \
    qt6-svg-dev \
    # session tooling
    xvfb openbox xdotool x11-utils x11-xserver-utils imagemagick xclip xsel wl-clipboard wtype libxkbcommon-tools zenity \
    weston sway grim dbus dbus-x11 dbus-user-session dunst libnotify-bin xdg-utils \
    xdg-desktop-portal xdg-desktop-portal-gtk xdg-desktop-portal-gnome shared-mime-info \
    fonts-dejavu-core fonts-noto-core adwaita-icon-theme librsvg2-common \
    libnss3 libasound2t64 at-spi2-core \
    && rm -rf /var/lib/apt/lists/*

# Go: official tarball, linux/arm64
RUN curl -fsSL https://go.dev/dl/go${GO_VERSION}.linux-arm64.tar.gz | tar -C /usr/local -xz
# Node: latest LTS, linux-arm64 tarball
RUN set -e; v=$(curl -fsSL https://nodejs.org/dist/index.json | jq -r '[.[]|select(.lts!=false)][0].version'); \
    curl -fsSL https://nodejs.org/dist/$v/node-$v-linux-arm64.tar.xz | tar -C /usr/local --strip-components=1 -xJ; node -v; npm -v
# Tailwind standalone for linux-arm64 (the repo's bin/tools one is a macOS binary)
RUN curl -fsSL -o /usr/local/bin/tailwindcss https://github.com/tailwindlabs/tailwindcss/releases/download/v4.3.5/tailwindcss-linux-arm64 && chmod +x /usr/local/bin/tailwindcss

# webview_go pins pkg-config name webkit2gtk-4.0, which Ubuntu 24.04 no longer ships (only 4.1).
# CONTAINER WORKAROUND: alias the 4.0 pkg-config name to 4.1 so webview_go can compile.
RUN sed 's/^Name:.*/Name: webkit2gtk-4.0 (alias of 4.1)/' /usr/lib/aarch64-linux-gnu/pkgconfig/webkit2gtk-4.1.pc > /usr/lib/aarch64-linux-gnu/pkgconfig/webkit2gtk-4.0.pc && \
    sed 's/^Name:.*/Name: webkit2gtk-web-extension-4.0 (alias)/' /usr/lib/aarch64-linux-gnu/pkgconfig/webkit2gtk-web-extension-4.1.pc > /usr/lib/aarch64-linux-gnu/pkgconfig/webkit2gtk-web-extension-4.0.pc

ENV PATH=/usr/local/go/bin:/root/go/bin:$PATH \
    GOPATH=/root/go GOMODCACHE=/gomod GOCACHE=/gocache GOTOOLCHAIN=local GOFLAGS=-mod=mod \
    GOPROXY=https://proxy.golang.org,direct \
    LIBGL_ALWAYS_SOFTWARE=1 GALLIUM_DRIVER=llvmpipe \
    NO_AT_BRIDGE=1 XDG_RUNTIME_DIR=/run/user/0 HOME=/root
RUN mkdir -p /run/user/0 && chmod 700 /run/user/0 /root && mkdir -p /gomod /gocache /work /repo
WORKDIR /work
# Scripts are used from the bind-mounted repo: /repo/linux/run.sh, lib.sh, prep.sh, wlinput.py
