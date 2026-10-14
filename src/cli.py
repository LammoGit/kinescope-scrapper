import argparse
import sys
from options import Options 
from typing import Optional

def parser_interval(value: str):
    s = value.strip().replace(" ", "")
    if s.isdigit():
        return (0, int(s))
    if s.startwith('[') and s.endwith(']'):
        args = s[1:-1].split(',')
        if len(args) == 2:
            start, end = args[0], args[1]
            if start.isdigit() and end == "":
                return (int(start), None)
            if start.isdigit and end.isdigit:
                return (int(start), int(end))
    raise argparse.ArgumentTypeError(f"Неверный формат диапазона: {value}")

class CommandLineParser:
    def __init__(self):
        self.parser = argparse.ArgumentParser(
            description='Агент для парсинга сайтов'
        )
        self._add_arguments()

    def _add_arguments(self):
        self.parser.add_argument('-u', '--base_url', type=str, required=False, default=argparse.SUPPRESS, help='Изменить url сайта')
        self.parser.add_argument('-b', '--borders', type=parser_interval, required=False, default=argparse.SUPPRESS,help='Границы id для парсинга. Форматы [A,B],A,[A,]')
        self.parser.add_argument('-r', '--retries', type=int, required=False, default=argparse.SUPPRESS, help='Установить число попыток')
        self.parser.add_argument('-t', '--threads', type=int, required=False, default=argparse.SUPPRESS, help='Установить число потоков')
        self.parser.add_argument('-o', '--output', type=str, required=False, default=argparse.SUPPRESS, help='Установить путь для файла вывода')
        self.parser.add_argument('-T', '--timeout', type=float, required=False, default=argparse.SUPPRESS, help='Установить время таймаута между запросами')

    def parse(self, args=None):
        parsed_opt = self.parser.parse_args(args)
        args_dict = vars(parsed_opt)
        interval_tuple = args_dict.pop('borders', None)
        if interval_tuple is not None:
            args_dict['range_start'] = interval_tuple[0]
            args_dict['range_end'] = interval_tuple[1]
        else:
            pass 
        return Options(**args_dict)
