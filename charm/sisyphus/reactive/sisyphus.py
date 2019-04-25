from charms.layer.sisyphus import Sisyphus

from charms.reactive import helpers, when, when_not, set_flag, clear_flag

from charmhelpers.core import hookenv, unitdata


def install():
    cfg = hookenv.config()['config']
    if cfg:
        unitdata.kv().set('sisyphus.config', cfg)
        helpers.data_changed('sisyphus.config', cfg)

        Sisyphus.install(cfg=cfg)

        set_flag('sisyphus.configured')
        hookenv.status_set('idle', 'waiting for further commands')
    else:
        hookenv.status_set('blocked', 'waiting for configuration') 


@when_not('sisyphus.installed')
def install_sisyphus():
    install()
    set_flag('sisyphus.installed')


@when('config.changed')
def config_changed():
    install()