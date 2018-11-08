HOST = http://localhost
PEER_PORT = 12380
CLIENT_PORT = 12379
ETCD_DIR = /tmp/neco-etcd/

clean:
	-rm -r $(ETCD_DIR)

start-etcd: clean
	etcd --data-dir $(ETCD_DIR) \
		--initial-cluster default=$(HOST):$(PEER_PORT) \
		--listen-peer-urls $(HOST):$(PEER_PORT) \
		--initial-advertise-peer-urls $(HOST):$(PEER_PORT) \
		--listen-client-urls $(HOST):$(CLIENT_PORT) \
		--advertise-client-urls $(HOST):$(CLIENT_PORT)

test:
	go test -v -count=1 -race -mod=vendor ./...


.PHONY:	start-etcd clean
