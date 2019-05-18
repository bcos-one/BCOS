build:
	docker build -t bcosone/bcos:v1.0.0 ../

rpc:
	docker run -d --name bcos -p 9545:9545 -p 30303:30303 bcosone/bcos:v1.0.0

bash:
	docker run -it --name bcos bcosone/bcos:v1.0.0 sh

kill:
	docker rm -f bcos

init:
	docker exec -it bcos /bin/entrypoint.sh init

start:
	docker exec -it bcos /bin/entrypoint.sh start

stop:
	docker exec -it bcos /bin/entrypoint.sh stop

test:
	curl -i -H 'content-type: application/json' -X POST --data '{"jsonrpc":"2.0","method":"eth_getBalance","params":["0x2831a3d42008a643eaa84d3547f6f77bfaa9a731", "latest"],"id":1}' localhost:9545
