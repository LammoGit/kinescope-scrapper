import requests
import time
from options import Options
from pathlib import Path


class SiteConnection:
    def __init__(self, options):
        self.options = options
        self.video_count = 0

    def get_count(self):
        catalog = Path(self.options.storage_path)
        self.video_count = sum(1 for path in catalog.iterdir() if path.is_dir())
        return self.video_count

    def _create_session(self):
        session = requests.Session()
        session.headers.update(
            {"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"}
        )
        return session

    def site_connect(self, url):
        session = self._create_session()
        for attempt in range(self.options.retries + 1):
            try:
                response = session.get(url, timeout=(15, self.options.timeout))
                if response.status_code == 200:
                    return (
                        response.text,
                        f"[{self.options.agent_name} status]: checking for {url}",
                    )
                if response.status_code == 429:
                    time.sleep(15)
                    if attempt > self.options.retries:
                        return (
                            None,
                            f"[{self.options.agent_name} status]: rate_limited for {url}",
                        )
                if response.status_code in (401, 403):
                    return (
                        None,
                        f"[{self.options.agent_name} status]: forbidden for {url}",
                    )
                if response.status_code == 404:
                    return (
                        None,
                        f"[{self.options.agent_name} status]: not_found for {url}",
                    )
            except requests.exceptions.Timeout:
                return (
                    None,
                    f"[{self.options.agent_name} status]: failed - timeout fail for {url}",
                )
            except requests.exceptions.RequestException:
                return (
                    None,
                    f"[{self.options.agent_name} status]: failed - request fail for {url}",
                )

    def download_metadata(self, catalog, name, description):
        storage_path = Path(self.options.storage_path + catalog)
        storage_path.mkdir(parents=True, exist_ok=True)
        file_path = storage_path / "metadata.json"
        with file_path.open("w", encoding="utf-8") as f:
            f.write("{\n")
            f.write(name + ",\n")
            f.write(description + "\n")
            f.write("}")
        print(
            f"\r[{self.options.agent_name} status]: {file_path} downloaded successfuly"
        )

    def download_media(self, url, catalog, filename):
        result = requests.get(url, stream=True, timeout=(15, self.options.timeout))
        result.raise_for_status()
        total = int(result.headers.get("Content-Length", 0))
        downloaded = 0
        storage_path = Path(self.options.storage_path + catalog)
        storage_path.mkdir(parents=True, exist_ok=True)
        file_path = storage_path / filename
        with file_path.open("wb") as f:
            for chunk in result.iter_content(chunk_size=1024 * 1024):
                if chunk:
                    f.write(chunk)
                    downloaded += len(chunk)
                if total:
                    print(
                        f"\r[{self.options.agent_name} status]: downloading {file_path} {downloaded / total:.1%}",
                        end="",
                    )
            print(
                f"\n[{self.options.agent_name} status]: {catalog + '/' + filename} downloaded successfuly"
            )
