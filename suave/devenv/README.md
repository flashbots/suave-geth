## Run the devnet:
`docker-compose up --build --force-recreate`

## Genesis info
Execution node's address: 0xb5feafbdd752ad52afb7e1bd2e40432a485bbb7f (private key: 6c45335a22461ccdb978b78ab61b238bad2fae4544fb55c14eb096c875ccfc52)
Pre-funded private key: 0x784a372aac67e9da69be6e3d1125205700f0149ab3a166f19a607e58501ec899, Address: 0x96C609E6A635E6D9568641d9F9F4e8F805967149

## Monitoring redis
docker exec -it devenv_redis_1 redis-cli
> auth default MTIzNDU2NzgK
> MONITOR

## Suave-cli
```
docker exec -it devenv_suave-cli_1 suavecli deployMevShareContract -privkey 784a372aac67e9da69be6e3d1125205700f0149ab3a166f19a607e58501ec899 -suave_rpc=http://suave-mevm-1:8545
docker exec -it devenv_suave-cli_1 suavecli sendBundle -ex_node_addr 0xb5feafbdd752ad52afb7e1bd2e40432a485bbb7f -goerli_rpc=http://suave-enabled-chain:8545 -privkey 784a372aac67e9da69be6e3d1125205700f0149ab3a166f19a607e58501ec899 -suave_rpc=http://suave-mevm-1:8545
```
