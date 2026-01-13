#!/bin/bash
# ===========================================
# Antigravity Lite - Quick Deploy Script
# One-click deployment for Linux servers
# ===========================================

set -e

echo "🚀 Antigravity Lite Quick Deploy"
echo "================================"

# Check Docker
if ! command -v docker &> /dev/null; then
    echo "❌ Docker not installed. Installing..."
    curl -fsSL https://get.docker.com | sh
    sudo systemctl enable docker
    sudo systemctl start docker
fi

if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    echo "📦 Installing Docker Compose..."
    sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
    sudo chmod +x /usr/local/bin/docker-compose
fi

# Create directories
mkdir -p data

# Create default config if not exists
if [ ! -f config.yaml ]; then
    echo "📝 Creating default config..."
    cat > config.yaml << 'EOF'
proxy:
  port: 8045
  host: "0.0.0.0"
  enable_auth: false
  api_keys: []

accounts:
  db_path: "./data/accounts.db"
  auto_refresh: true
  refresh_interval: 30

routing:
  default_model: "gemini-2.0-flash"
  model_mapping: {}
EOF
fi

echo ""
echo "🔨 Building and starting..."

# Use docker compose v2 if available
if docker compose version &> /dev/null; then
    docker compose up -d --build
else
    docker-compose up -d --build
fi

echo ""
echo "✅ Deployment complete!"
echo ""
echo "📊 Access Web UI:  http://$(hostname -I | awk '{print $1}'):8045"
echo "📡 OpenAI API:     http://$(hostname -I | awk '{print $1}'):8045/v1/chat/completions"
echo "📡 Claude API:     http://$(hostname -I | awk '{print $1}'):8045/v1/messages"
echo ""
echo "🔑 Add accounts via web UI or use OAuth authorization"
echo ""
echo "📋 Useful commands:"
echo "   View logs:      docker logs -f antigravity-lite"
echo "   Stop:           docker-compose down"
echo "   Restart:        docker-compose restart"
