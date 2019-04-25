from charms.reactive import when

from charmhelpers.core import unitdata, hookenv


@when('kafka.ready')
def kafka_read(kafka):
    hookenv.log(kafka.kafkas())
    unitdata.kv().set('sisyphus.kafkas', kafka.kafkas())
