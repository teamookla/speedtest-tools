
# Ookla
# Updated 5/30/18

# This Python script queries a list of available data extract files from Speedtest Intelligence,
# determines what data sets are available, and then downloads the most recent version of each.
# By default the files are stored in the directory where the script is running, but modifying
# the storageDir variable will allow you to specify a directory.

try: # Python3
    import urllib.request as compatible_urllib
except ImportError: # Python 2
    import urllib2 as compatible_urllib

import json
import os
import base64
import sys

extracts_url = 'https://intelligence.speedtest.net/extracts'

# Please replace MyApiKey and MyApiSecret below with your organization's API key.
username = 'my_api_key'
password = 'my_api_secret'

# By default, the script stores the extract files in the directory where the script is running
# To specify a storage directory, change this value to a string represting the directory where
# the files should be stored.
# Example: storageDir = '/data/ookla/extracts'
storageDir = os.getcwd()

opener = compatible_urllib.build_opener()
compatible_urllib.install_opener(opener)
opener.addheaders = [('Accept', 'application/json')]

# setup authentication
login_credentials = '%s:%s' % (username, password)
base64string = base64.b64encode(login_credentials.encode('utf-8')).decode('ascii')
opener.addheaders = [('Authorization', 'Basic %s' % base64string)]

# makes request for files
response = compatible_urllib.urlopen(extracts_url).read()
try:
    content = json.loads(response)
except ValueError as err:
    print(err)
    sys.exit()

#############################################################
# loop through contents, sort through files and directories
def sort_files_and_directories(contents):
    for entry in contents:
        if entry['type'] == 'file' and entry['name'].find('headers') == -1 and '_20' in entry['name']:
            filter(entry)
        elif entry['type'] == 'dir':
            subdir = extracts_url + entry['url']
            files = json.loads(compatible_urllib.urlopen(subdir).read())
            sort_files_and_directories(files)

# determine if file should be downloaded - check for new datasets and most current file for exisiting datasets
def filter(file):
    # identify the dataset by the file name prefix
    dataset = file['name'][:file['name'].index('_20')]
    if dataset not in files or dataset in files and file['mtime'] > files[dataset]['age']:
        files[dataset] = {'name': file['name'], 'url': file['url'], 'age': file['mtime']}

def download(files):
    if len(files):
        for data_set, file in files.items():
            response = compatible_urllib.urlopen(file['url'])
            flocation = storageDir + '/' + file['name']
            print(("Downloading: %s" % (file['name'])))
            with open(flocation, 'wb') as content:
                content.write(response.read())
        return
    print("No data extract files found. If this is an error, please contact your technical account manager.")

#############################################################
files = {}
sort_files_and_directories(content)
download(files)
