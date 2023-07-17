# Installing

* [Docker](#docker-container)
* [Docker Compose](#docker-compose)
* [Build From Source](#building-from-source)
* [Systemd Service](#run-as-a-systemd-service-on-ubuntu)

Contributions and corrections are welcomed here. Would be nice to add a section on Akash deployments too.

## Docker Container

```shell
mkdir tenderduty && cd tenderduty
docker run --rm ghcr.io/blockpane/tenderduty:latest -example-config >config.yml
# edit config.yml and add chains, notification methods etc.
docker run -d --name tenderduty -p "8888:8888" -p "28686:28686" --restart unless-stopped -v $(pwd)/config.yml:/var/lib/tenderduty/config.yml ghcr.io/blockpane/tenderduty:latest
docker logs -f --tail 20 tenderduty
```

## Docker Compose

```shell
mkdir tenderduty && cd tenderduty

cat > docker-compose.yml << EOF
---
version: '3.2'
services:

  v2:
    image: ghcr.io/blockpane/tenderduty:latest
    command: ""
    ports:
      - "8888:8888" # Dashboard
      - "28686:28686" # Prometheus exporter
    volumes:
      - home:/var/lib/tenderduty
      - ./config.yml:/var/lib/tenderduty/config.yml
      - ./chains.d:/var/lib/tenderduty/chains.d/
    logging:
      driver: "json-file"
      options:
        max-size: "20m"
        max-file: "10"
    restart: unless-stopped

volumes:
  home:
EOF

docker-compose pull
docker run --rm ghcr.io/blockpane/tenderduty:latest -example-config >config.yml

# Edit the config.yml file, and then start the container
docker-compose up -d
docker-compose logs -f --tail 20
```

## Building from source

*Note: building tenderduty requires go v1.18 or later*

### Installing Go

If you intend to build from source, you will need to install Go. There are many choices on how to do this. **The most common method is to use the official installation instructions at [go.dev](https://go.dev/doc/install),** but there are a couple of shortcuts that can also be used:

Ubuntu:
```shell
sudo apt-get install -y snapd
sudo snap install go --classic
```

MacOS: using [Homebrew](https://brew.sh)
```shell
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
brew install go
```

### Building

Because of design choices made by golang devs, it's not possible to use 'go install' remotely because tendermint uses replace directives in the go.mod file. It's necessary to clone the repo and build manually.

```
git clone https://github.com/blockpane/tenderduty
cd tenderduty
cp example-config.yml config.yml
# edit config.yml with your favorite editor
go get ./...
go build -ldflags '-s -w' -trimpath -o ~/go/bin/tenderduty main.go
```

## Run as a systemd service on Ubuntu

First, create a new user

```shell
sudo addgroup --system tenderduty 
sudo adduser --ingroup tenderduty --system --home /var/lib/tenderduty tenderduty
```

Install Go: see [the instructions above](#installing-go).

Install the binaries

```shell
sudo -su tenderduty
cd ~
echo 'export PATH=$PATH:~/go/bin' >> .bashrc
. .bashrc
git clone https://github.com/blockpane/tenderduty
cd tenderduty
go install
cp example-config.yml ../config.yml
cd ..
# Edit the config.yml with your editor of choice
exit
```

Now create and enable the service

```shell
# Create the service file
sudo tee /etc/systemd/system/tenderduty.service << EOF
[Unit]
Description=Tenderduty
After=network.target
ConditionPathExists=/var/lib/tenderduty/go/bin/tenderduty

[Service]
Type=simple
Restart=always
RestartSec=5
TimeoutSec=180

User=tenderduty
WorkingDirectory=/var/lib/tenderduty
ExecStart=/var/lib/tenderduty/go/bin/tenderduty

# there may be a large number of network connections if a lot of chains
LimitNOFILE=infinity

# extra process isolation
NoNewPrivileges=true
ProtectSystem=strict
RestrictSUIDSGID=true
LockPersonality=true
PrivateUsers=true
PrivateDevices=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF

# Enable and start the service
sudo systemctl daemon-reload
sudo systemctl enable tenderduty
sudo systemctl start tenderduty

# and to watch the logs, press CTRL-C to stop watching
sudo journalctl -fu tenderduty

```
