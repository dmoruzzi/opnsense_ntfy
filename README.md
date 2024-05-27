# OPNSense NTFY

This application periodically checks an RSS feed for new updates and sends notifications using the [NTFY](https://docs.ntfy.sh) service.

## Features

- Fetches and parses RSS feeds.
- Sends notifications to configured ntfy servers.
- Supports multiple notification servers with different authentication tokens.
- Configurable refresh intervals.

## Configuration

The application is configured using a [config.yml](config.yml.example) file.