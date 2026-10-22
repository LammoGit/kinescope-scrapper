from pydantic import BaseModel
from typing import Optional


class Options(BaseModel):
    agent_name: str = "worker_1"
    range_start: int = 0
    range_end: Optional[int] = None
    base_url: str = "https://kinescope.io"
    storage_path: str = "../downloads/"
    threads: int = 4
    timeout: float = 35.0
    retries: int = 3
