package vault

// UnsealScript returns sciript contents to auto unseal vault.
func UnsealScript() string {
	return `#!/usr/bin/python3

import base64
import json
from os import path
import subprocess
import sys
import time

import requests


ETCDCTL = '/usr/local/bin/etcdctl'
KEY_UNSEAL = 'boot/vault-unseal-key'


def wait_vault():
    while True:
        try:
            requests.head('https://127.0.0.1:8200/v1/sys/health')
            return
        except requests.ConnectionError:
            time.sleep(1)


def get_unseal_key() ->str:
    crt = '/etc/etcd/backup.crt'
    key = '/etc/etcd/backup.key'
    if not path.exists(crt):
        crt = '/etc/vault/etcd.crt'
        key = '/etc/vault/etcd.key'

    env = {'ETCDCTL_API': '3'}
    p = subprocess.run(
        [ETCDCTL, '-w', 'json', '--cert='+crt, '--key='+key, 'get', KEY_UNSEAL],
        env=env, check=True, stdout=subprocess.PIPE)
    j = json.loads(p.stdout)
    if 'kvs' not in j:
        return ''

    return base64.b64decode(j['kvs'][0]['value']).decode('utf-8')


def main():
    wait_vault()

    unseal_key = get_unseal_key()

    if unseal_key == '':
        print('no unseal key in etcd')
        return

    r = requests.put('https://127.0.0.1:8200/v1/sys/unseal',
                     json={'key': unseal_key})
    if r.status_code != 200:
        sys.exit('unseal failed with {}'.format(r.status_code))

    print('vault unsealed')


if __name__ == '__main__':
    main()
`
}
