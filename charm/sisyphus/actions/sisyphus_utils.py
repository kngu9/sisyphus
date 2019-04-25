import sys

from charmhelpers.core import hookenv


def fail(msg):
    hookenv.action_set({'outcome': 'failure'})
    hookenv.action_fail(msg)
    sys.exit()
