#!/bin/bash

echo "🔨 Building WebRTC Social Server..."


echo "📦 Downloading dependencies..."
go mod download

echo "🏗️  Compiling..."
go build -ldflags "-s -w " -trimpath -buildvcs=false -o server main.go

if [ $? -eq 0 ]; then
    echo "✅ Build successful!"
    echo ""
    echo "Run with: ./server serve"
else
    echo "❌ Build failed!"
    exit 1
fi
