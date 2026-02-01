#!/bin/bash
VOSK_VERSION="0.3.45"
BUILD_DIR="build/windows"

# Download Vosk if not present
if [ ! -d "vosk-win64-${VOSK_VERSION}" ]; then
    wget https://github.com/alphacep/vosk-api/releases/download/v${VOSK_VERSION}/vosk-win64-${VOSK_VERSION}.zip
    unzip vosk-win64-${VOSK_VERSION}.zip
fi

# Build
export CGO_CFLAGS="-I${PWD}/vosk-win64-${VOSK_VERSION}"
export CGO_LDFLAGS="-L${PWD}/vosk-win64-${VOSK_VERSION} -lvosk"
GOOS=windows GOARCH=amd64 go build -o ${BUILD_DIR}/vosk-to-resonite.exe ./cmd/vosk-to-resonite

# Copy DLLs
cp vosk-win64-${VOSK_VERSION}/*.dll ${BUILD_DIR}/

echo "Build complete in ${BUILD_DIR}/"