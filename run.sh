#!/bin/bash

NAME=server
DEBUG=0


for i in "$@"
do
case $i in
    -d|--debug)
    echo "🔨 Running WebRTC Social Server..."
    echo "📦 Update dependencies..."
    go mod tidy
    if [ $? -eq 0 ]; then
        echo "✅ Dependencies updated!"
        echo ""
    else
        echo "❌ Update dependencies failed!"
        exit 1
    fi
    go run . serve
    exit
    ;;
    -b|--build)
    echo "🔨 Building WebRTC Social Server..."
    echo "📦 Downloading dependencies..."
    go mod download

    echo "🏗️  Compiling..."
    go build -ldflags "-s -w" -trimpath -buildvcs=false -o $NAME main.go

    if [ $? -eq 0 ]; then
        echo "✅ Build successful!"
        echo ""
    else
        echo "❌ Build failed!"
        exit 1
    fi
    ;;
esac
done


if [ ! -f "$NAME" ]; then
    echo "❌ server binary \"$NAME\" not found!"
    exit 2
fi


echo "🚀 Starting WebRTC Social Server..."
echo ""
echo "📡 Server will be available at:"
echo "   - API: http://localhost:8090/api/"
echo "   - Admin: http://localhost:8090/_/"
echo ""

./$NAME serve
