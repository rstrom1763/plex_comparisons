# Ryan Strom

# exports as a CSV what movies are in the first DB and not in the second
# inputs are the file paths to each of the CSV files.

def plex_compare(db1, db2, file, exclude_file=""):

    import utils
    import json

    space_needed = 0
    if exclude_file != "":
        exclude_file = (open(exclude_file, 'r', encoding='utf-8')).readlines()
        for line in exclude_file:
            exclude_file[exclude_file.index(line)] = line.replace("\n", "")

    # Import both DB files into list of dictionaries
    if ".csv" in db1:
        db1 = utils.import_csv(db1)
    elif ".json" in db1:
        db1 = json.load(db1)
    else:
        print("Unknown data type")
        return
    if ".csv" in db2:
        db2 = utils.import_csv(db2)
    elif ".json" in db2:
        db2 = json.load(db2)
    else:
        print("Unknown data type")
        return

    dict1 = {}
    dict2 = {}
    list = []

    if "Season" in db1[0] and "Season" in db2[0]:
        media_type = "tv"
    elif "Season" not in db1[0] and "Season" not in db2[0]:
        media_type = "movie"
    else:
        print("Error: Mismatching DB Types")
        return

    if media_type == "movie":
        for dict in db1:
            dict1[dict["Title"].lower()] = dict
        for dict in db2:
            dict2[dict["Title"].lower()] = dict
        for movie in dict1:
            if movie not in dict2 and movie not in exclude_file:
                list.append(dict1[movie])
    elif media_type == "tv":
        for dict in db1:
            if dict['Series Title'].lower() not in dict1:
                dict1[dict["Series Title"].lower()] = {}
                dict1[dict['Series Title'].lower()][dict['Episode Title'].lower()] = dict
            elif dict['Episode Title'].lower() not in dict1[dict['Series Title'].lower()]:
                dict1[dict['Series Title'].lower()][dict['Episode Title'].lower()] = dict

        for dict in db2:
            if dict['Series Title'].lower() not in dict2:
                dict2[dict["Series Title"].lower()] = {}
                dict2[dict['Series Title'].lower()][dict['Episode Title'].lower()] = dict
            elif dict['Episode Title'].lower() not in dict2[dict['Series Title'].lower()]:
                dict2[dict['Series Title'].lower()][dict['Episode Title'].lower()] = dict
        
        for show in dict1.values():
            for episode in show.values():
                if episode['Series Title'].lower() not in dict2:
                    list.append(episode)
                else:
                    found = False
                    for show2 in dict2.values():
                        if found == True : break
                        for episode2 in show2.values():
                            if episode['Episode Title'].lower() in episode2['Episode Title'].lower() and episode['Series Title'].lower() in episode2['Series Title'].lower():
                                found = True
                                break
                    if found == False : list.append(episode)
                                
# Iterated through the dictionaries to print required file space
    for item in list:
        if item["Part Size as Bytes"].isdecimal():
            space_needed += int(item["Part Size as Bytes"])
    print(utils.human_readable(space_needed))

    utils.export_csv(list, file)


def get_library(url, token, library):

    from plexapi.server import PlexServer

    # print(type(test[0].locations))
    return ((PlexServer(url, token)).library.section(library)).search()


def transcribe_data_csv(files, out_file):

    import os
    import utils

    out_file = open(out_file, 'w', encoding="utf-8")

    out_file.write('"{}","{}","{}","{}","{}"\n'.format(
        "Title", "Year", "FileSize", "FilePath", "Duration"))

    for file in files:
        out_file.write('"{}","{}","{}","{}","{}"\n'.format(
            file.title, file.year, utils.human_readable(os.path.getsize(file.locations[0])), file.locations[0]), file.duration)

    out_file.close()


def transcribe_data_json(files):

    import os
    import utils
    import json

    dict = {}

    for file in files:

        file_size_human = utils.human_readable(
            os.path.getsize(file.locations[0]))

        file_size_bytes = os.path.getsize(file.locations[0])

        dict[file.title] = {"title": file.title, "year": file.year,
                            "file_size_human_readable": file_size_human, "file_size_bytes": file_size_bytes, "filepath": file.locations[0], "duration": file.duration}

    return json.dumps(dict)


def sync_data(json, url, token):

    import requests

    headers_dict = {"token": token,
                    'Content-type': 'application/json', 'Accept': 'text/plain'}
    requests.post(url, headers=headers_dict, data=json, verify=False)


def download_diff(url, token, out_file):
    # Functional: Sept 19 2021

    import requests

    headers_dict = {"token": token}

    out_file = open(out_file, 'w', encoding='utf-8')

    out_file.write(requests.get(
        url, headers=headers_dict, verify=False).json())

    out_file.close()


def new_user(username, url, password):
    import requests
    import string
    import random
    import json

    N = 20
    token = ''.join(random.SystemRandom().choice(
        string.ascii_letters + string.digits) for _ in range(N))

    headers_dict = {'Content-type': 'application/json',
                    'Accept': 'text/plain', 'access_token': token}

    data_json = {'username': username,
                 'access_token': token, 'password': password}

    data_json = json.dumps(data_json)
    result = requests.post(url, headers=headers_dict,
                           data=data_json, verify=False)

    if result.status_code != 200:
        print("Oh no")


# Examples:
'''
plex_compare("C:/Strom/KentLibrary.csv",
             "C:/Strom/ryanlibrary.csv", "C:/strom/ryan_no_have.csv")
plex_compare("C:/Strom/ryanLibrary.csv",
             "C:/Strom/kentlibrary.csv", "C:/strom/kent_no_have.csv")
'''
plex_compare("C:/Strom/kent_shows.csv", "C:/strom/ryan_shows.csv","C:/strom/ryan_shows_no_have.csv")
plex_compare("C:/Strom/ryan_shows.csv", "C:/strom/kent_shows.csv","C:/strom/kent_shows_no_have.csv")

# Returns list of items from specified Plex library

'''
data = transcribe_data_json(get_library('http://localhost:32400',
                                        '4CX8sBFPjAVSfJWohux5', "Movies"))
sync_data(data, "https://localhost:8081/sync", '4CX8sBFPjAVSfJWohux5')
'''
#new_user('test', 'https://plex.localdomain:8081/newuser', 'testPassword')

'''download_diff('https://plex:8081/diff',
              '4CX8sBFPjAVSfJWohux5', "C:/strom/test.json")'''
