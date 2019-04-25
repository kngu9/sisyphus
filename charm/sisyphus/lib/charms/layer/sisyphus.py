import os
import shutil
import re
import socket

from pathlib import Path
from base64 import b64encode

from charmhelpers.core import hookenv, host
from charmhelpers.core.templating import render

from charms.reactive.relations import RelationBase

from charms.layer import snap

SISYPHUS_SNAP = 'sisyphus'
SISYPHUS_COMMON = '/var/snap/{}/common'.format(SISYPHUS_SNAP)


class Sisyphus(object):
    @staticmethod
    def install(cfg=hookenv.config()['config']):
        '''
        Generates client-ssl.properties and server.properties with the current
        system state.
        '''
        render(
            None,
            os.path.join(SISYPHUS_COMMON, "config.yaml"),
            config_template=cfg,
            perms=0o644
        )


def keystore_password():
    path = os.path.join(
        SISYPHUS_COMMON,
        'keystore.secret'
    )
    if not os.path.isfile(path):
        with os.fdopen(
                os.open(path, os.O_WRONLY | os.O_CREAT, 0o440),
                'wb') as f:
            token = b64encode(os.urandom(32))
            f.write(token)
            password = token.decode('ascii')
    else:
        password = Path(path).read_text().rstrip()
    return password


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
