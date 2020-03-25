#!/usr/bin/env python3
import json, os

from audiapi.API import API
from audiapi.Services import LogonService

credentials = 'audi.json'

if __name__ == '__main__':
    api = API()
    logon_service = LogonService(api)
    if not logon_service.restore_token():
        # We need to login
        if os.path.isfile(credentials):
            with open(credentials) as data_file:
                data = json.load(data_file)
            logon_service.login(data['user'], data['pass'])
        else:
            user = os.environ['AUDI_USER']
            password = os.environ['AUDI_PASS']
            logon_service.login(user, password)
