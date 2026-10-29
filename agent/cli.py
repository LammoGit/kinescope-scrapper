import argparse
import os


def parser_interval(value: str):
    s = value.strip().replace(" ", "")
    if s.isdigit():
        return (0, int(s))
    if s.startswith("[") and s.endswith("]"):
        args = s[1:-1].split(",")
        if len(args) == 2:
            start, end = args[0], args[1]
            if start.isdigit() and end == "":
                return (int(start), None)
            if start.isdigit and end.isdigit:
                return (int(start), int(end))
    raise argparse.ArgumentTypeError(f"Неверный формат диапазона: {value}")


class CommandLineInterface:
    def menu(self, options, count):
        os.system("cls" if os.name == "nt" else "clear")
        print(f"""KINESCOPE PARSER AGENT v0.40.0
Total downloaded videos: {count}
{"-" * 47}
""")
        print(f""" :: SERVICE
    1. Parse One Link 
    2. Parse Range Of Links

 :: OPTIONS
    3. Agent Name\t[{options.agent_name}]
    4. Base Url\t\t[{options.base_url}]
    5. Urls Range\t[{options.range_start}, {options.range_end}]
    6. Storage Path\t[{options.storage_path}]
    7. Threads Amount\t[{options.threads}]
    8. Timeout Time\t[{options.timeout}]
    9. Retries Amout\t[{options.retries}]

{"-" * 47}
    0. Exit
""")

    def start(self, agent_name):
        os.system("cls" if os.name == "nt" else "clear")
        print(f"""{agent_name.upper()} STARTED WORKING ON TASK:""")

    def change_option(self, options, option_to_change):
        while True:
            os.system("cls" if os.name == "nt" else "clear")
            if option_to_change == "borders":
                option_type = "[A,B]"
            else:
                option_type = type(getattr(options, option_to_change))
            print(f"""You are changing {option_to_change} new variable must be {option_type}.
If you want to cancle change just press Enter with null input field or write 0.""")
            new_variable = input("Input new variable: ")
            if new_variable == "" or new_variable == 0:
                return None
            if option_to_change == "borders":
                try:
                    start, end = parser_interval(new_variable)
                    options.range_start = start
                    options.range_end = end
                    return options
                except TypeError:
                    continue
            else:
                try:
                    new_variable = option_type(new_variable)
                    setattr(options, option_to_change, new_variable)
                    return options
                except TypeError:
                    continue


class CommandLineParser:
    def __init__(self):
        self.parser = argparse.ArgumentParser(description="Агент для парсинга сайтов")
        self._add_arguments()

    def _add_arguments(self):
        self.parser.add_argument(
            "-a",
            "--agent_name",
            type=str,
            required=False,
            default=argparse.SUPPRESS,
            help="Изменить имя агента",
        )
        self.parser.add_argument(
            "-u",
            "--base_url",
            type=str,
            required=False,
            default=argparse.SUPPRESS,
            help="Изменить url сайта",
        )
        self.parser.add_argument(
            "-b",
            "--borders",
            type=parser_interval,
            required=False,
            default=argparse.SUPPRESS,
            help="Границы id для парсинга. Форматы [A,B],A,[A,]",
        )
        self.parser.add_argument(
            "-r",
            "--retries",
            type=int,
            required=False,
            default=argparse.SUPPRESS,
            help="Установить число попыток",
        )
        self.parser.add_argument(
            "-t",
            "--threads",
            type=int,
            required=False,
            default=argparse.SUPPRESS,
            help="Установить число потоков",
        )
        self.parser.add_argument(
            "-s",
            "--storage_path",
            type=str,
            required=False,
            default=argparse.SUPPRESS,
            help="Установить путь для каталога установки",
        )
        self.parser.add_argument(
            "-T",
            "--timeout",
            type=float,
            required=False,
            default=argparse.SUPPRESS,
            help="Установить время таймаута между запросами",
        )

    def parse(self, args=None):
        parsed_opt = self.parser.parse_args(args)
        args_dict = vars(parsed_opt)
        interval_tuple = args_dict.pop("borders", None)
        if interval_tuple is not None:
            args_dict["range_start"] = interval_tuple[0]
            args_dict["range_end"] = interval_tuple[1]
        else:
            pass
        path = args_dict.pop("storage_path", None)
        if path is not None:
            if path.endswith("/"):
                args_dict["storage_path"] = path
            else:
                args_dict["storage_path"] = path + "/"
        return args_dict
