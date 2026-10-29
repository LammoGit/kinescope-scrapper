from cli import CommandLineParser, CommandLineInterface
from connection import SiteConnection
from options import Options
from parser import KinescopeParser
from itertools import count
import threading
import time

console_mode = threading.Event()
stop_flag = threading.Event()


def waiting_for_connection():
    while not stop_flag.is_set():
        if console_mode.is_set():
            time.sleep(0.1)
            continue


def wait_for_enter(parser, connection, options):
    input()
    console_mode.set()
    interface = CommandLineInterface()
    interface.menu(options, connection.get_count())
    console_loop(interface, parser, connection, options)


def console_loop(interface, parser, connection, options):
    while not stop_flag.is_set():
        try:
            cmd = input("\nSelect option(0-9): ").strip()
        except EOFError:
            break

        match cmd:
            case "0":
                stop_flag.set()
                break
            case "1":
                interface.start(options.agent_name)
                get_video(parser, options, connection, options.base_url)
                print("Waiting for Enter to continue...")
                input()
                interface.menu(options, connection.get_count())
            case "2":
                interface.start(options.agent_name)
                if options.range_end is not None:
                    for i in range(options.range_start, options.range_end):
                        url = options.base_url + "/" + str(i)
                        get_video(parser, options, connection, url)
                else:
                    it = count(options.range_start)
                    for i in it:
                        url = options.base_url + "/" + str(i)
                        get_video(parser, options, connection, url)
                print("Waiting for Enter to continue...")
                input()
                interface.menu(options, connection.get_count())
            case "3":
                new_options = interface.change_option(options, "agent_name")
                if new_options is not None:
                    options = new_options
                interface.menu(options, connection.get_count())
            case "4":
                new_options = interface.change_option(options, "base_url")
                if new_options is not None:
                    options = new_options
                interface.menu(options, connection.get_count())
            case "5":
                new_options = interface.change_option(options, "borders")
                if new_options is not None:
                    options = new_options
                interface.menu(options, connection.get_count())
            case "6":
                new_options = interface.change_option(options, "storage_path")
                if new_options is not None:
                    options = new_options
                interface.menu(options, connection.get_count())
            case "7":
                new_options = interface.change_option(options, "threads")
                if new_options is not None:
                    options = new_options
                interface.menu(options, connection.get_count())
            case "8":
                new_options = interface.change_option(options, "timeout")
                if new_options is not None:
                    options = new_options
                interface.menu(options, connection.get_count())
            case "9":
                new_options = interface.change_option(options, "retries")
                if new_options is not None:
                    options = new_options
                interface.menu(options, connection.get_count())
            case _:
                interface.menu(options, connection.get_count())


def get_video(parser, options, connection, url):
    search_result = parser.search_for_metadata(connection, url, options.agent_name)
    if search_result:
        connection.download_metadata(url[21:], search_result[1], search_result[2])
        connection.download_media(search_result[3], url[21:], "thumbnail.jpg")
        parser.download_video(
            connection, search_result[0], url[21:], options.agent_name
        )


def main():
    option_parser = CommandLineParser()
    args_dict = option_parser.parse()
    options = Options(**args_dict)

    parser = KinescopeParser()
    connection = SiteConnection(options)

    if "base_url" in args_dict:
        get_video(parser, options, connection, options.base_url)
    elif (
        "base_url" not in args_dict
        and "range_start" in args_dict
        and "range_end" in args_dict
    ):
        for id in range(options.range_start, options.range_end):
            url = options.base_url + "/" + str(id)
            print(f"[{options.agent_name} status: started search for {url}]")
            get_video(parser, options, connection, url)
    else:
        print("Waiting for connection, for manual use press Enter...")
        connection_thread = threading.Thread(target=waiting_for_connection, daemon=True)
        connection_thread.start()
        console_thread = threading.Thread(
            target=wait_for_enter, args=(parser, connection, options), daemon=True
        )
        console_thread.start()

        try:
            while not stop_flag.is_set():
                time.sleep(0.2)
        except KeyboardInterrupt:
            stop_flag.set()


if __name__ == "__main__":
    main()
