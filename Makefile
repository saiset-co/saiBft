up:
	docker-compose -f ./microservices/docker-compose.yml up -d

down:
	docker-compose -f ./microservices/docker-compose.yml down --remove-orphans

build:
	make service
	make docker

service:
	## bft
	cd ./src/saiBft && go build -o ../../microservices/saiBft/build/sai-bft
	cp ./src/saiBft/build/config.yml ./microservices/saiBft/build/config.yml
	## saiStorage
	cd ./src/saiStorage && go build -o ../../microservices/saiStorage/build/sai-storage
	cp ./src/saiStorage/config/config.json ./microservices/saiStorage/build/config.json
	

docker:
	docker-compose -f ./microservices/docker-compose.yml up -d --build

logs:
	docker-compose -f ./microservices/docker-compose.yml logs -f

logn:
	docker-compose -f ./microservices/docker-compose.yml logs -f sai-consensus

sh:
	docker-compose -f ./microservices/docker-compose.yml exec sai-consensus sh

run:
	docker-compose -f ./microservices/docker-compose.yml run --rm sai-consensus
