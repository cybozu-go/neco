module github.com/cybozu-go/neco/pkg/necogcp-functions

go 1.13

replace github.com/cybozu-go/neco => ../../

require (
	cloud.google.com/go/pubsub v1.6.1
	github.com/cybozu-go/log v1.6.0
	github.com/cybozu-go/neco v0.0.0-20200804040141-29ac5b698130
	github.com/cybozu-go/well v1.10.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/spf13/cobra v1.0.0
)
