# Testground test plans for datahop ipfs-lite

## Running
We run tests on docker. There are few configured tests that require more than 10 minutes to complete. 
For that we need to configure testground `~/.env.toml` with the following line.
```
[daemon.scheduler]
task_timeout_min          = 60
```

### Tests
``` 
testground run single --plan=test-plans/datahop --testcase=connection --runner=local:docker --builder=docker:go --instances=10
```