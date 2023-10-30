import json
import argparse
import os

parser = argparse.ArgumentParser()
parser.add_argument("--redis-folder", default="../../redis/src/commands")
args = parser.parse_args()
directory = args.redis_folder

go_file = """
package main

var readOnlyCommands = map[string]bool{
"""


for command_filename in os.listdir(directory):
    f = os.path.join(directory, command_filename)
    # checking if it is a file
    if os.path.isfile(f):
        with open(f, "r") as json_fd:
            command_json = json.load(json_fd)
            for command_name, command_details in command_json.items():
                if "command_flags" in command_details:
                    command_flags = command_details["command_flags"]
                    if "READONLY" in command_flags and "NOSCRIPT" not in command_flags:
                        go_file_newline = f'"{command_name}":true,\n'
                        go_file += go_file_newline
go_file += "}"

print(go_file)
