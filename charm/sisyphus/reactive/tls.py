import os

from charms.layer import tls_client
from charms.layer.sisyphus import SISYPHUS_COMMON

from charms.reactive import when


@when('certificates.available')
def send_data():
    # Request a server cert with this information.
    tls_client.request_client_cert(
        'system:snap-sisyphus',
        crt_path=os.path.join(
            SISYPHUS_COMMON,
            'client.crt',
        ),
        key_path=os.path.join(
            SISYPHUS_COMMON,
            'client.key'
        )
    )
