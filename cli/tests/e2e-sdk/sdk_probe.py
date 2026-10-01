"""The Python SDK, gzip on by default: one track to /v1/batch.

Usage: python sdk_probe.py DATA_PLANE_URL
"""

import sys

from rudderstack.analytics.client import Client

if len(sys.argv) != 2:
    sys.exit("usage: python sdk_probe.py DATA_PLANE_URL")

c = Client("python", host=sys.argv[1], sync_mode=True)
c.track(user_id="u1", event="SDK Probe", properties={"sdk": "python", "n": 1})
c.flush()
print("python sent")
