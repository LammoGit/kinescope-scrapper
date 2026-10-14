import requests
import time
import paramiko
from options import Options

class SiteConnection:

    def __init__(self, options):
        self.options = options

    def _create_session(self):
        session = requests.Session()
        session.headers.update({
            'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)'
        })
        return session

    def site_connect(self, url):
        session = self._create_session()
        for attempt in range(self.options.retries + 1):
            try:
                response = session.get(url, timeout=options.timeout)
                if response.status_code == 200:
                    return response.text, 'checking'
                if response.status_code == 429:
                    time.sleep(timeout)
                    if attempt > options.retries:
                        return None, 'rate_limited'
                if response.status_code in (401, 403):
                    return None, 'forbidden'
                if response.status_code == 404:
                    return None, 'not_found'
            except requests.RequestException:
                return None, 'failed'

