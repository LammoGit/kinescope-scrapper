from cli import CommandLineParser
from connection import SiteConnection
from options import Options 

def main():
    option_parser = CommandLineParser()

    options = option_parser.parse()

    print(options)

if __name__ == "__main__":
    main()
