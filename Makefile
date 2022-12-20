up:
	docker-compose -f ./microservices/docker-compose.yml up -d

down:
	docker-compose -f ./microservices/docker-compose.yml down --remove-orphans

build:
	make service
	make docker

service:
#	cd ./src/saiStorage && go mod tidy && go build -o ../../microservices/saiStorage/build/sai-storage
	cd ./src/saiBft && go mod tidy && go build -o ../../microservices/saiBft/build/sai-bft
#	cd ./src/saiVM1 && go mod tidy && go build -o ../../microservices/saiVM1/build/sai-vm1
#	cd ./src/saiBTC && go mod tidy && go build -o ../../microservices/saiBtc/build/sai-btc
#	cd ./src/saiP2pProxy && go mod tidy && go build -o ../../microservices/saiP2pProxy/build/sai-p2p
#	cp ./src/saiP2pProxy/config.yml ./microservices/saiP2pProxy/build/config.yml
#	cp ./src/saiBft/config.yml ./microservices/saiBft/build/config.yml
#	cp ./src/saiVM1/config.yml ./microservices/saiVM1/build/config.yml
#	cp ./src/saiBft/btc_keys.json ./microservices/saiBft/build/btc_keys.json
#	cp ./src/saiBTC/saibtc.config ./microservices/saiBtc/build/saibtc.config
#	cp ./src/saiStorage/config.json ./microservices/saiStorage/build/config.json
	

docker:
	docker-compose -f ./microservices/docker-compose.yml up -d --build

log:
	docker-compose -f ./microservices/docker-compose.yml logs -f

logp:
	docker-compose -f ./microservices/docker-compose.yml logs -f sai-p2p

logpr:
	docker-compose -f ./microservices/docker-compose.yml logs -f sai-p2p-proxy

logb:
	docker-compose -f ./microservices/docker-compose.yml logs -f sai-bft

loga:
	docker-compose -f ./microservices/docker-compose.yml logs -f sai-auth

logs:
	docker-compose -f ./microservices/docker-compose.yml logs -f sai-storage

logc:
	docker-compose -f ./microservices/docker-compose.yml logs -f sai-contract-explorer

logv:
	docker-compose -f ./microservices/docker-compose.yml logs -f sai-vm1

sha:
	docker-compose -f ./microservices/docker-compose.yml run --rm sai-auth sh

shv:
	docker-compose -f ./microservices/docker-compose.yml run --rm sai-vm1 sh

shb:
	docker-compose -f ./microservices/docker-compose.yml run --rm sai-bft sh

shs:
	docker-compose -f ./microservices/docker-compose.yml run --rm sai-storage sh

shc:
	docker-compose -f ./microservices/docker-compose.yml run --rm sai-contract-explorer sh
