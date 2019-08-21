/*

Package etcdutil helps implementation of the common specification to
take etcd connection parameters.

To read parameters from YAML (or JSON):

    import (
        "github.com/cybozu-go/etcdutil"
        "sigs.k8s.io/yaml"
    )

    func main() {
        cfg := etcdutil.NewConfig("/key-prefix/")
        err := yaml.Unmarshal(data, cfg)
        if err != nil {
            panic(err)
        }

        etcd, err := etcdutil.NewClient(cfg)
        if err != nil {
            panic(err)
        }
        defer etcd.Close()

        // use etcd; it is a etcd *clientv3.Client object.
    }

To read parameters from command-line flags:

    import (
        "flag"

        "github.com/cybozu-go/etcdutil"
    )

    func main() {
        cfg := etcdutil.NewConfig("/key-prefix/")
        cfg.AddFlags(flag.CommandLine)
        flag.Parse()

        etcd, err := etcdutil.NewClient(cfg)
        if err != nil {
            panic(err)
        }
        defer etcd.Close()

        // use etcd; it is a etcd *clientv3.Client object.
    }

*/
package etcdutil
