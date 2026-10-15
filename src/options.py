from pydantic import Model
from typing import Optional

class Options(Model):
    range_start: int = 0
    range_end: Optional[int] = None
    base_url: str = "https://kinescope.io"
    output: str = "./downloads"
    threads: int = 4
    timeout: float = 35.0
    retries: int = 3



