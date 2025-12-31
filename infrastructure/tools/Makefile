#!/bin/bash
create_topic:
	./kafka-topics.sh --bootstrap-server localhost:9092 --topic service_registry --partitions 1 --replication-factor 1  --create
list_topic:
	./kafka-topics.sh --bootstrap-server localhost:9092 --list