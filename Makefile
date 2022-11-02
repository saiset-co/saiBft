up:
	docker-compose -f ./microservices/docker-compose.yml up -d

down:
	docker-compose -f ./microservices/docker-compose.yml down --remove-orphans

build:
	make service
	make docker

service:
	cd ./src/saiStorage && go mod tidy && go build -o ../../microservices/saiStorage/build/sai-storage
	cd ./src/saiBft && go mod tidy && go build -o ../../microservices/saiBft/build/sai-bft
	cd ./src/saiBTC && go mod tidy && go build -o ../../microservices/saiBtc/build/sai-btc	
	cd ./src/saiP2p && go mod tidy && go build -o ../../microservices/saiP2p/build/sai-p2p	
	cp ./src/saiBft/build/config.yml ./microservices/saiBft/build/build/config.yml
	cp ./src/saiBTC/saibtc.config ./microservices/saiBtc/build/saibtc.config
	cp ./src/saiStorage/config.json ./microservices/saiStorage/build/config.json
	


docker:
	docker-compose -f ./microservices/docker-compose.yml up -d --build

log:
	docker-compose -f ./microservices/docker-compose.yml logs -f

loga:
	docker-compose -f ./microservices/docker-compose.yml logs -f sai-auth

logs:
	docker-compose -f ./microservices/docker-compose.yml logs -f sai-storage

logc:
	docker-compose -f ./microservices/docker-compose.yml logs -f sai-contract-explorer

sha:
	docker-compose -f ./microservices/docker-compose.yml run --rm sai-auth sh

shs:
	docker-compose -f ./microservices/docker-compose.yml run --rm sai-storage sh

shc:
	docker-compose -f ./microservices/docker-compose.yml run --rm sai-contract-explorer sh
