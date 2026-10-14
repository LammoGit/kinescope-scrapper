from dataclasses import dataclass
from typing import Optional

@dataclass
class Options:
    range_start: int = 0
    range_end: Optional[int] = None
    base_url: str = "https://kinescope.io"
    output: str = "./downloads"
    threads: int = 4
    timeout: float = 35.0
    retries: int = 3



