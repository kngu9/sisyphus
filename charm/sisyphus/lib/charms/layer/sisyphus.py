import os
import re
import socket
import signal

from pathlib import Path
from base64 import b64encode

from charmhelpers.core import hookenv
from charmhelpers.core.templating import render


SISYPHUS_SNAP = 'sisyphus'
SISYPHUS_COMMON = '/var/snap/{}/common'.format(SISYPHUS_SNAP)
SISYPHUS_CERT = os.path.join(
    SISYPHUS_COMMON,
    'client.crt',
)
SISYPHUS_KEY = os.path.join(
    SISYPHUS_COMMON,
    'client.key',
)
SISYPHUS_CA_CERT = os.path.join(
    SISYPHUS_COMMON,
    'ca.crt',
)

class Sisyphus(object):
    @staticmethod
    def install(cfg=hookenv.config()['config']):
        render(
            None,
            os.path.join(SISYPHUS_COMMON, "config.yaml"),
            config_template=cfg,
            context={},
            perms=0o644
        )

    def __init__(self):
        self.lock_path = os.path.join(
            os.sep, 'var', 'run', 'sisyphus', 'pid'
        )

        if not os.path.isfile(self.lock_path):
            os.makedirs(os.path.dirname(self.lock_path))

            open(self.lock_path, 'w+').close()
            os.chmod(self.lock_path, 0o644)

    def get_pid(self):
        buff = None

        with open(self.lock_path, 'r') as f:
            buff = f.read()

        if not buff:
            return -1

        return int(buff)
    
    def lock_pid(self, pid):
        with open(self.lock_path, 'w+') as f:
            f.write(str(pid))

    def is_running(self):
        pid = self.get_pid()
        return os.path.exists(os.path.join('proc', str(pid)))

    def terminate(self, signal=signal.SIGTERM):
        pid = self.get_pid()

        if pid != -1 and self.is_running():
            os.kill(pid, signal)
    
    def clear_pid(self):
        with open(self.lock_path, 'w+') as f:
            f.write('')


def resolve_private_address(addr):
    IP_pat = re.compile(r'\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}')
    contains_IP_pat = re.compile(r'\d{1,3}[-.]\d{1,3}[-.]\d{1,3}[-.]\d{1,3}')
    if IP_pat.match(addr):
        return addr  # already IP
    try:
        ip = socket.gethostbyname(addr)
        return ip
    except socket.error as e:
        hookenv.log(
            'Unable to resolve private IP: %s (will attempt to guess)' %
            addr,
            hookenv.ERROR
        )
        hookenv.log('%s' % e, hookenv.ERROR)
        contained = contains_IP_pat.search(addr)
        if not contained:
            raise ValueError(
                'Unable to resolve private-address: {}'.format(addr)
            )
        return contained.groups(0).replace('-', '.')
