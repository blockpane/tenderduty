# Installing

This documentation is incomplete. But will cover Docker, Systemd, and hopefully Akash options. For now ...

if you'd prefer to containerize and not build locally, you can:

```
mkdir tenderduty && cd tenderduty
docker run --rm ghcr.io/blockpane/tenderduty:latest -example-config >config.yml
# edit config.yml and add chains, notification methods etc.
docker run -d --name tenderduty -p "8888:8888" -p "28686:28686" --restart unless-stopped -v $(pwd)/config.yml:/var/lib/tenderduty/config.yml ghcr.io/blockpane/tenderduty:latest
docker logs -f --tail 20 tenderduty
```

Or if building from source:

```
git clone https://github.com/blockpane/tenderduty
cd tenderduty
git checkout release/v2
cp example-config.yml config.yml
# edit config.yml
go get ./...
go install
~/go/bin/tenderduty
```
