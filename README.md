# nexus-agent

A lightweight, single-binary service monitor with Slack alerts.

Checks 79 services in 25 milliseconds. Sends state-change alerts to Slack. No databases. No dashboards. Just one binary.

## Install

Download the binary for your platform from the Releases page, or:

    curl -L https://github.com/YOURNAME/nexus-agent/releases/latest/download/nexus-agent-linux-amd64 -o nexus-agent
    chmod +x nexus-agent
    sudo mv nexus-agent /usr/local/bin/

## Usage

One-shot check:

    nexus-agent --config services.json

Continuous watch mode (30 second interval):

    nexus-agent --watch --config services.json

Custom interval:

    nexus-agent --watch --interval 10s

## Config

Create a services.json file:

    [
      {"name": "HTTP_Server", "port": "4221", "protocol": "tcp"},
      {"name": "DNS_Server",  "port": "2053", "protocol": "udp"},
      {"name": "Redis",       "port": "6380", "protocol": "tcp"}
    ]

Protocol is either tcp or udp.

## Slack alerts

Set your Slack webhook as an environment variable:

    export NEXUS_SLACK_WEBHOOK="https://hooks.slack.com/services/..."

Alerts only fire on state changes:
- Service goes down - alert sent
- Service recovers - alert sent
- Service stays down - no repeated alerts (no spam)

## License

MIT
