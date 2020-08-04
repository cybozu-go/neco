GCP_PROJECT ?= tapih-test
ZONE ?= us-west1-a
ACCOUNT_JSON_PATH ?= hoge
SERVICE_ACCOUNT_NAME ?= fuga

all:

deploy-function:
	gcloud functions deploy auto-dctest --project $(GCP_PROJECT) \
		--entry-point PubSubEntryPoint --runtime go113 \
		--trigger-topic autodctest-scheduler-events \
		--set-env-vars ACCOUNT_JSON_PATH=$(ACCOUNT_JSON_PATH),GCP_PROJECT=$(GCP_PROJECT),SERVICE_ACCOUNT_NAME=$(SERVICE_ACCOUNT_NAME),ZONE=$(ZONE)

deploy-create-scheduler:
	gcloud beta scheduler jobs create pubsub create-dctest \
		--project $(GCP_PROJECT) \
		--schedule '0 9 * * 1-5' \
		--topic autodctest-scheduler-events \
		--message-body '{"mode":"create", "namePrefix":"maneki", "num":2}' \
		--time-zone 'Asia/Tokyo' \
		--description 'automatically create dctest instance'

deploy-delete-scheduler:
	gcloud beta scheduler jobs create pubsub delete-dctest \
		--project $(GCP_PROJECT) \
		--schedule '0 20 * * *' \
		--topic autodctest-scheduler-events \
		--message-body '{"mode":"delete", "doForce":false}' \
		--time-zone 'Asia/Tokyo' \
		--description 'automatically delete dctest instances except for ones with skip-auto-delete label'

deploy-force-delete-scheduler:
	gcloud beta scheduler jobs create pubsub force-delete-dctest \
		--project $(GCP_PROJECT) \
		--schedule '0 23 * * *' \
		--topic autodctest-scheduler-events \
		--message-body '{"mode":"delete", "doForce":true}' \
		--time-zone 'Asia/Tokyo' \
		--description 'automatically delete dctest all instances'

.PHONY: \
	deploy-function \
	deploy-create-scheduler \
	deploy-delete-scheduler \
	deploy-force-delete-scheduler
