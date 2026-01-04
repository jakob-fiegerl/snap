#!/bin/bash

set -e

echo "📸 Snap Installer"
echo "=================="
echo ""

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Error: Go is not installed"
    echo "Please install Go first: https://go.dev/doc/install"
    exit 1
fi

echo "✓ Go is installed"

# Build the binary
echo "🔨 Building snap..."
go mod tidy
go build -o snap

if [ ! -f "snap" ]; then
    echo "❌ Error: Build failed"
    exit 1
fi

echo "✓ Build successful"

# Create ~/bin directory if it doesn't exist
if [ ! -d "$HOME/bin" ]; then
    echo "📁 Creating ~/bin directory..."
    mkdir -p "$HOME/bin"
fi

# Copy the binary to ~/bin
echo "📦 Installing snap to ~/bin..."
cp snap "$HOME/bin/snap"
chmod +x "$HOME/bin/snap"

echo "✓ Snap installed to ~/bin/snap"
echo ""

# Check if ~/bin is in PATH
if [[ ":$PATH:" != *":$HOME/bin:"* ]]; then
    echo "⚠️  Warning: ~/bin is not in your PATH"
    echo ""
    echo "Add this line to your shell configuration file:"
    echo ""
    
    # Detect shell
    if [ -n "$BASH_VERSION" ]; then
        echo "  echo 'export PATH=\"\$HOME/bin:\$PATH\"' >> ~/.bashrc"
        echo "  source ~/.bashrc"
    elif [ -n "$ZSH_VERSION" ]; then
        echo "  echo 'export PATH=\"\$HOME/bin:\$PATH\"' >> ~/.zshrc"
        echo "  source ~/.zshrc"
    else
        echo "  export PATH=\"\$HOME/bin:\$PATH\""
    fi
    echo ""
else
    echo "✓ ~/bin is already in your PATH"
fi

echo ""
echo "🎉 Installation complete!"
echo ""
echo "Usage:"
echo "  snap save              Save changes with AI-generated commit message"
echo "  snap save --seed 123   Use a custom seed"
echo "  snap help              Show help"
echo ""
echo "Make sure Ollama is running with phi4 model installed:"
echo "  ollama serve"
echo "  ollama pull phi4"
