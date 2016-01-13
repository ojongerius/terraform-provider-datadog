#!/usr/bin/env python3

import re
import os
import sys
import logging
from datadog import initialize, api


class migrateDatadogProvider:
    """
    """
    def __init__(self):
        if 'DATADOG_API_KEY' not in os.environ or 'DATADOG_APP_KEY' not in os.environ:
            print("Need both DATADOG_API_KEY and DATADOG_APP_KEY to be set, bailing")
            sys.exit(2)
        self.api_key = os.environ.get('DATADOG_API_KEY')
        self.app_key = os.environ.get('DATADOG_APP_KEY')
        self.delete_list = []

    def convert_state(self):
        """
        Write new statefile, find monitors that can be removed
        """
        with open('terraform.tfstate.new', 'w') as new:
            with open('terraform.tfstate', 'rb') as old:
                for line in old:
                    res = re.match(b'.*"(([0-9]+)__([0-9]+))".*', line)
                    if res:
                        if res.group(3) not in self.delete_list:
                            self.delete_list.append(res.group(3))
                        line = line.replace(res.group(1), res.group(2))
                    new.write(line.decode('utf-8'))

        print("Done writing terraform.tfstate.new")

    def delete_monitors(self):
        """
        Delete monitors found by convert_state
        """
        options = {
            'api_key': self.api_key,
            'app_key': self.app_key
        }

        initialize(**options)
        logging.basicConfig()

        [api.Monitor.delete(x) for x in self.delete_list]


def main():

    migrator = migrateDatadogProvider()

    print("Converting terraform.tfstate")
    migrator.convert_state()

    print("Removing left over monitors")
    migrator.delete_monitors()

    print("Done with state conversion and monitor removal, you will need to:\n * Inspect and move terraform.tfstate.new to terrafrom.tfsate\n * Run terraform plan and apply to finish the migration.")

if __name__ == '__main__':
    main()
