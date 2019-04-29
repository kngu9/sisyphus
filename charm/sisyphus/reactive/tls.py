import os
import shutil

from charmhelpers.core import hookenv

from charms.reactive import (when, remove_state,
                             when_not, set_state)

from charms.layer import tls_client
from charms.layer.sisyphus import (SISYPHUS_CA_CERT,
                                   SISYPHUS_CERT, SISYPHUS_KEY)


@when('certificates.available')
def send_data():
    # Request a server cert with this information.
    tls_client.request_client_cert(
        hookenv.service_name(),
        crt_path=SISYPHUS_CERT,
        key_path=SISYPHUS_KEY)


@when('tls_client.ca_installed')
@when_not('sisyphus.ca.certificate.saved')
def import_ca_crt_to_keystore():
    ca_path = '/usr/local/share/ca-certificates/{}.crt'.format(
        hookenv.service_name()
    )

    if os.path.isfile(ca_path):
        shutil.copyfile(ca_path, SISYPHUS_CA_CERT)
        remove_state('tls_client.ca_installed')
        set_state('sisyphus.ca.certificate.saved')
