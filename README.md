# Monitrix 📡

A real-time network monitoring tool built in Go that continuously tests network connectivity by pinging multiple remote hosts and provides a beautiful web dashboard for visualization.

I created this tool originally because the constant network issues I have been with the Malaysia broadband internet provider `UniFi`. I need a way to track downtime history and retain logs as a prove to them since they always do not honor the SLAs.

![screenshot](screenshots/1.png?raw=true "Screenshot")

## Features

✅ **Continuous Network Monitoring**: Pings multiple hosts (Google, Apple, Cloudflare, GitHub) every 30 seconds  
✅ **Persistent Logging**: All results saved to local JSON log files  
✅ **Web Dashboard**: Real-time visualization with dark theme  
✅ **Downtime Detection**: Automatically tracks and highlights connectivity issues  
✅ **Time Range Selection**: Filter data by custom date/time ranges  
✅ **Statistics**: Per-host uptime percentages, latency metrics, and failure counts  
✅ **Auto-refresh**: Dashboard updates every 30 seconds


## Prerequisites

- Go 1.24 or higher
- Network connectivity to test hosts


## Usage

### Docker Deployment 🐳


```bash
# 1. Copy and configure environment variables
cp .env.example .env

# 2. Edit .env to set your hosts and interval

# 3. Start with Docker Compose
docker-compose up -d

# 4. View logs
docker-compose logs -f monitrix

# 5. Access dashboard at http://localhost:8080
```

### Access Dashboard

Once running, open your browser to:
```
http://localhost:8080
```

### What You'll See

The application will:
1. Start monitoring network connectivity to configured hosts
2. Log results to `data/network_monitor_YYYY-MM-DD.jsonl`
3. Print ping results to console
4. Serve the web dashboard on port 8080

### Configuration

**Via Environment Variables (Recommended):**

| Variable | Default | Description |
|----------|---------|-------------|
| `MONITOR_HOSTS` | `google.com,rexlow.com,github.com` | Comma-separated hosts |
| `MONITOR_INTERVAL` | `30` | Check interval in seconds |
| `WEB_ADDR` | `0.0.0.0:8080` | Web server address |

## Development

### Project Structure

- `internal/monitor/ping.go`: Core ping/connectivity testing logic
- `internal/storage/file.go`: File-based logging system
- `internal/api/server.go`: HTTP API and statistics calculation
- `cmd/monitrix/main.go`: Application orchestration
- `web/index.html`: Single-page dashboard application

## License

MIT License - Feel free to modify and use as needed!

---

Built with ❤️ using Go and vanilla JavaScript
