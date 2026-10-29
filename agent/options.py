from pydantic import BaseModel
from pathlib import Path
from typing import Optional


class Options(BaseModel):
    agent_name: str = "worker_1"
    range_start: int = 0
    range_end: Optional[int] = None
    base_url: str = "https://kinescope.io"
    storage_path: str = str(Path(__file__).parent)[:-3] + "/downloads/"
    threads: int = 4
    timeout: float = 120
    retries: int = 3
