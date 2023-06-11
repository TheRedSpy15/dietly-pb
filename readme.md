# Dietly Pocketbase backend
[![Go](https://github.com/TheRedSpy15/dietly-pb/actions/workflows/go.yml/badge.svg)](https://github.com/TheRedSpy15/dietly-pb/actions/workflows/go.yml)

This is the custom implementation of Pocketbase for Dietly

### Run locally
`go run main.go serve --http=0.0.0.0:8090`

### Updating Pocketbase version (and other dependencies)
`go get -u all`

## Host with Fly.io
Currently we are using the free plan at fly.io

### Push to production server

- Requirements
    - [requires fly.io cli tools](https://fly.io/docs/hands-on/install-flyctl/)
    - Run flyctl `auth signup` to create a Fly.io account (email or GitHub).
    - Run flyctl `auth login` to login.
- deploy
    `flyctl deploy`