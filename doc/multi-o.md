# Multi Orchestrator Setup

This document describes the various ways an orchestrator could run multiple nodes and separate concerns. 

## Unlocking Account and Connecting to Ethereum

Unlocking your Ethereum account and connecting to a JSON-RPC provider are necessary steps to run the Livepeer node unless it's in standalone transcoder mode. 

For brevity we'll exclude these flags in the examples by `<...ETH SETUP...>` but these should always be required. If no Ethereum account is available the Livepeer node will create one for you.

```
livepeer
    -network <mainnet|rinkeby> \
    -ethUrl <JSON_RPC_PROVIDER> \
    -ethKeystorePath <PATH_TO_KEY> \
    -ethAcctAddr <ETHEREUM_ADDRESS> \
    -ethPassword <ETHEREUM_ACCOUNT_PASSWORD (string|path)> \
    -ethController <CONTROLLER_CONTRACT_ADDRESS (required if '-network' not provided)>
```

## Single Orchestrator with Redeemer

This allows an orchestrator running a single on-chain node to use a seperate Ethereum account to redeem winning tickets on-chain and pay for that transaction while the recipient is still orchestrator node's Ethereum address.

The orchestrator node will still be responsible for initialising rounds and calling `reward`. 

1. Start the Redeemer 

```shell
livepeer
    <...ETH SETUP...\
    -redeemer=true \ 
    -httpAddr <REDEEMER_HTTP_ADDR (host):port \ 
    -ethOrchAddr <ORCHESTRATOR_ON_CHAIN_ETH_ADDR (also the recipient address)>
```

2. Start the Orchestrator


```shell
livepeer
    <...ETH SETUP...> \
    -orchestrator=true -transcoder=true \
    -initializeRound=true \
    -httpAddr <ORCH_HTTP_ADDR (host):port \
    -pricePerUnit <PRICE (wei/pixel if '-pixelsPerUnit' is not set)
```


## Blockchain Service Node with Multiple Orchestrators

In this setup a node started with the keys for the on-chain registered address will be responsible for all transactions. 

In order to use multiple orchestrators with this setup, the cluster of orchestrator nodes responsible for transcoding would have to be behind a load balancer, **the URI of the load balancer will be the on-chain registered Service URI.**

1. Start the Blockchain Service Node 

```shell
livepeer
    <...ETH SETUP...\
    -orchestrator= true \ 
    -redeemer=true \ 
    -httpAddr <REDEEMER_HTTP_ADDR (host):port \
    -initializeRound=true 
```

2. Start an Orchestrator 
```shell
livepeer
    <...ETH SETUP...> \
    -orchestrator=true -transcoder=true \
    -ethOrchAddr <ORCHESTRATOR_ON_CHAIN_ETH_ADDR (also the recipient address)> \
    -redeemerAddr <REDEEMER_HTTP_ADDR> \
    -httpAddr <ORCH_HTTP_ADDR (host):port \
    -pricePerUnit <PRICE (wei/pixel if '-pixelsPerUnit' is not set)
```

## Redeemer + RewardService & RoundInitializer + Multiple Orchestrators

This is the "coldest" setup possible for on-chain registered addresses. The keys for the on-chain registered address will be responsible for initialising rounds and calling reward, since reward can not be called on behalf of another address in the current version of the protocol.

To use this setup with multiple orchestrators a load balancer is required as described above. 

1. Start the Redeemer

```shell
livepeer
    <...ETH SETUP...\
    -redeemer=true \ 
    -httpAddr <REDEEMER_HTTP_ADDR (host):port \ 
    -ethOrchAddr <ORCHESTRATOR_ON_CHAIN_ETH_ADDR (also the recipient address)>
```

2. Start the RewardService & RoundInitializer 

In the current go-livepeer version (0.5.8) this node type will still run a transcoder server. To avoid this node also transcoding avoid making it accessible through the load balancer of your choice. 

```shell
livepeer
    <...ETH SETUP...> \
    -orchestrator=true -transcoder=true\
    -initializeRound=true \
    -pricePerUnit <PRICE (wei/pixel if '-pixelsPerUnit' is not set)
```

3. Start an Orchestrator 

The main difference is to use the `-ethOrchAddr` flag and specify the Ethereum address of the RewardService & RoundInitializer node. 

```shell
livepeer
    <...ETH SETUP...> \
    -orchestrator=true \
    -ethOrchAddr <ORCHESTRATOR_ON_CHAIN_ETH_ADDR (also the recipient address)>
    -redeemerAddr <REDEEMER_HTTP_ADDR> \
    -pricePerUnit <PRICE (wei/pixel if '-pixelsPerUnit' is not set)
```