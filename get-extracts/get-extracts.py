# Created by Brian Turley
# Solutions Engineer
# Ookla
# 4/22/16
# Updated 1/18/17
# Tested in Python 2.7 on OS X 10.10.5, Ubuntu 14, and Windows 10

# This Python script queries a list of available data extract files from Speedtest Intelligence,
# determines what data sets are available, and then downloads the most recent version of each.
# By default the files are stored in the directory where the script is running, but modifying
# the storageDir variable will allow you to specify a directory.

import urllib2
import json
import os
import base64
import sys


theurl = 'https://intelligence.speedtest.net/extracts'

#Please replace MyUserName and MyDataPassword below with your
#organization's API key.

username = 'MyApiKey'
password = 'MyApiSecret'



#By default, the script stores the extract files in the directory where the script is running
#To specify a storage directory, change this value to a string represting the directory where
#the files should be stored.
#Example: storageDir = '/data/ookla/extracts'

storageDir = os.getcwd()

opener = urllib2.build_opener()
urllib2.install_opener(opener)

#request json list of files
opener.addheaders = [('Accept', 'application/json')]

#setup authentication
base64string = base64.b64encode('%s:%s' % (username, password))
opener.addheaders = [('Authorization', 'Basic %s' % base64string)]

#If login page is returned, raise error message.
result = urllib2.urlopen(theurl)
if result.info().type == "text/html":
    print "Authentication error.\nPlease verify that the API key is correct."
    sys.exit()

try:
    response = result.read()
except urllib2.HTTPError, err:
   if err.code == 500:
       print "Error: The account associated with this API key does not have access to data extracts.\nPlease contact your technical account manager to enable data extracts for this account."
       sys.exit()
try:
    json_data = json.loads(response)
except ValueError, e:
    print e
    sys.exit()

files = {}

#loop through list of files to find the most recent of each extract type
for x in json_data:
    #exclude directories and column header files
    if x['type'] == 'file' and x['name'].find('headers') == -1:

        #set the extract type by according to the file name prefix
        if '_20' in x['name']:
            sep = x['name'].index('_20')
            ftype = x['name'][:sep]

            if ftype in files:
                #if this data set already exist, check if this file is newer.
                if x['mtime'] > files[ftype]['age']:
                    files[ftype]['name'] = x['name']
                    files[ftype]['url'] = x['url']
                    files[ftype]['age'] = x['mtime']
            else:
                #If no other files of this data set exist, and this file.
                files[ftype] = {}
                files[ftype]['name'] = x['name']
                files[ftype]['url'] = x['url']
                files[ftype]['age'] = x['mtime']

    #loop through sub directories
    if x['type'] == 'dir':
        subdir = theurl + x['url']

        subresult = urllib2.urlopen(subdir)
        subjson = json.loads(subresult.read())

        for y in subjson:
            #exclude directories and column header files
            if y['type'] == 'file' and y['name'].find('headers') == -1:

                #set the extract type by according to the file name prefix
                if '_20' in y['name']:
                    sep = y['name'].index('_20')
                    ftype = y['name'][:sep]

                    if ftype in files:
                        #if this data set already exist, check if this file is newer.
                        if y['mtime'] > files[ftype]['age']:
                            files[ftype]['name'] = y['name']
                            files[ftype]['url'] = y['url']
                            files[ftype]['age'] = y['mtime']
                    else:
                        #If no other files of this data set exist, and this file.
                        files[ftype] = {}
                        files[ftype]['name'] = y['name']
                        files[ftype]['url'] = y['url']
                        files[ftype]['age'] = y['mtime']


if len(files) == 0:
    print "No data extract files found. If this is an error, please contact your technical account manager."

#download most recent files
for key, value in files.iteritems():
    fname = value['name']
    furl = value['url']
    u = urllib2.urlopen(furl)
    flocation = storageDir + '/' + fname
    f = open(flocation, 'wb')
    meta = u.info()
    file_size = int(meta.getheaders("Content-Length")[0])
    print "Downloading: %s Bytes: %s" % (fname, file_size)

    file_size_dl = 0
    block_sz = 8192
    while True:
        buffer = u.read(block_sz)
        if not buffer:
            break

        file_size_dl += len(buffer)
        f.write(buffer)
        status = r"%10d  [%3.2f%%]" % (file_size_dl, file_size_dl * 100. / file_size)
        status = status + chr(8)*(len(status)+1)
        print status,

