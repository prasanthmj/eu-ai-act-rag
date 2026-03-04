#!/bin/bash
set -e

PIDFILE=".server.pid"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

start_server() {
    local mode="${1:-http}"

    if [ -f "$PIDFILE" ] && kill -0 "$(cat "$PIDFILE")" 2>/dev/null; then
        echo "Server already running (PID $(cat "$PIDFILE")). Stop it first: ./run.sh stop"
        exit 1
    fi

    if [ ! -f .env ]; then
        echo "Error: .env file not found. Create it with OPENAI_API_KEY."
        exit 1
    fi

    source .env
    echo "Starting $mode server..."
    go run . --mode "$mode" &
    echo $! > "$PIDFILE"
    sleep 2

    if kill -0 "$(cat "$PIDFILE")" 2>/dev/null; then
        echo "Server started (PID $(cat "$PIDFILE"), mode=$mode)"
    else
        echo "Server failed to start. Check logs."
        rm -f "$PIDFILE"
        exit 1
    fi
}

stop_server() {
    if [ ! -f "$PIDFILE" ]; then
        echo "No server running (no PID file)."
        return
    fi

    local pid
    pid=$(cat "$PIDFILE")
    if kill -0 "$pid" 2>/dev/null; then
        kill "$pid"
        echo "Server stopped (PID $pid)"
    else
        echo "Server was not running (stale PID file)."
    fi
    rm -f "$PIDFILE"
}

status_server() {
    if [ -f "$PIDFILE" ] && kill -0 "$(cat "$PIDFILE")" 2>/dev/null; then
        echo "Server running (PID $(cat "$PIDFILE"))"
    else
        echo "Server not running"
        rm -f "$PIDFILE" 2>/dev/null
    fi
}

case "${1:-}" in
    start)
        start_server "${2:-http}"
        ;;
    stop)
        stop_server
        ;;
    status)
        status_server
        ;;
    restart)
        stop_server
        sleep 1
        start_server "${2:-http}"
        ;;
    *)
        echo "Usage: ./run.sh {start|stop|status|restart} [http|mcp]"
        echo ""
        echo "Examples:"
        echo "  ./run.sh start http    Start HTTP server on :8080"
        echo "  ./run.sh start mcp     Start MCP server on stdio"
        echo "  ./run.sh stop          Stop the running server"
        echo "  ./run.sh status        Check if server is running"
        echo "  ./run.sh restart http  Restart the server"
        exit 1
        ;;
esac
