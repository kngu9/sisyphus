from charms.layer.sisyphus import Sisyphus

from charms.reactive import helpers, when, when_not, set_flag, clear_flag

from charmhelpers.core import hookenv


def install():
    cfg = hookenv.config()['config']
    if not cfg:
        clear_flag('sisyphus.configured')
        hookenv.status_set('blocked', 'waiting for configuration')
        return

    if helpers.data_changed('sisyphus.config', cfg):
        Sisyphus.install(cfg=cfg)

    set_flag('sisyphus.configured')
    hookenv.status_set('waiting', 'waiting for further commands')


@when_not('sisyphus.installed')
def install_sisyphus():
    install()
    set_flag('sisyphus.installed')


@when('config.changed')
def config_changed():
    install()
