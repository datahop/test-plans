# Testground test plans for datahop ipfs-lite

## Running
``` 
testground run single --plan=test-plans/datahop --testcase=connection --runner=local:docker --builder=docker:go --instances=10
```