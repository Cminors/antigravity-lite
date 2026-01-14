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
server:
  port: 8045
  host: "0.0.0.0"
  log_level: "info"
  lan_access: true

proxy:
  timeout: 120
  max_retries: 3
  auto_rotate: true
  stream_enabled: true
  schedule_mode: "balance"
  max_wait_time: 60

storage:
  db_path: "./data/antigravity.db"

routes:
  - pattern: "gpt-4*"
    target: "gemini-3-pro-high"
  - pattern: "gpt-4o*"
    target: "gemini-3-flash"
  - pattern: "gpt-3.5*"
    target: "gemini-2.5-flash"
  - pattern: "claude-3-haiku-*"
    target: "gemini-2.5-flash-lite"
  - pattern: "claude-3-5-sonnet-*"
    target: "claude-sonnet-4-5"
  - pattern: "claude-3-opus-*"
    target: "claude-opus-4-5-thinking"
EOF
fi

echo ""
echo "🔨 Building and starting..."

# Use docker compose v2 if available, fallback to v1
if docker compose version &> /dev/null; then
    docker compose up -d --build
elif command -v docker-compose &> /dev/null; then
    docker-compose up -d --build
else
    echo "❌ Neither 'docker compose' nor 'docker-compose' found. Please install Docker Compose."
    exit 1
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
