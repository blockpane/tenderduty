# TenderDuty v2

![dashboard screenshot](docs/dash.png)
TenderDuty v2 is complete rewrite of the original tenderduty graciously sponsored by the [Osmosis Grants Program](https://grants.osmosis.zone/). This new version adds a web dashboard, prometheus exporter, telegram and discord notifications, multi-chain support, more granular alerting, and more types of alerts.

Documentation will be provided soon. The example-config.yml file is well-commented.

30 second quickstart for beta testers:

```
git clone https://github.com/blockpane/tenderduty
cd tenderduty
git checkout feature/v2
cp example-config.yml config.yml
# edit config.yml
go get ./...
go install
~/go/bin/tenderduty
```

if you'd prefer to containerize, you can also 
```
docker-compose up -d 
docker-compose logs -f
```