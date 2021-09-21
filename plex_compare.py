# Ryan Strom

# exports as a CSV what movies are in the first DB and not in the second
# inputs are the file paths to each of the CSV files.

def plex_compare(db1, db2, file, exclude_file=""):

    import utils

    space_needed = 0
    if exclude_file != "":
        exclude_file = (open(exclude_file, 'r', encoding='utf-8')).readlines()
        for line in exclude_file:
            exclude_file[exclude_file.index(line)] = line.replace("\n", "")

    # Import both CSV's into list of dictionaries
    db1 = utils.import_csv(db1)
    db2 = utils.import_csv(db2)

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
            dict1[dict["Title"]] = dict
        for dict in db2:
            dict2[dict["Title"]] = dict

        for movie in dict1:
            if movie not in dict2 and movie not in exclude_file:
                list.append(dict1[movie])
    elif media_type == "tv":
        for dict in db1:
            print(dict["Episode Title"])
        for dict in db1:
            dict1[dict["Episode Title"]] = dict
        for dict in db2:
            dict2[dict["Episode Title"]] = dict

    # Iterated through the dictionaries to print required file space
    for movie in list:
        if movie["Part Size as Bytes"].isdecimal():
            space_needed += int(movie["Part Size as Bytes"])
    print(utils.human_readable(space_needed))

    utils.export_csv(list, file)

# Examples:
#plex_compare("C:/Strom/KentLibrary.csv", "C:/Strom/ryanlibrary.csv", "C:/strom/ryan_no_have.csv", exclude_file="C:/strom/test.txt")
#plex_compare("C:/Strom/ryanlibrary.csv", "C:/strom/kentlibrary.csv", "C:/strom/kent_no_have.csv")
#plex_compare("C:/Strom/testing/test1.csv", "C:/Strom/testing/test2.csv", "C:/strom/testing/test_no_have.csv")


# Returns list of items from specified Plex library
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
    requests.post(url, headers=headers_dict, data=json)


def download_diff(url, token, out_file):
    # Functional: Sept 19 2021

    import requests

    headers_dict = {"token": token}

    out_file = open(out_file, 'w', encoding='utf-8')

    out_file.write(requests.get(url, headers=headers_dict).json())

    out_file.close()


def new_user(username, url):
    import requests
    import string
    import random
    import json

    N = 20
    token = ''.join(random.SystemRandom().choice(
        string.ascii_uppercase + string.digits) for _ in range(N))

    headers_dict = {'Content-type': 'application/json', 'Accept': 'text/plain'}

    data_json = {'username': username, 'access_token': token}
    data_json = json.dumps(data_json)
    requests.post(url, headers=headers_dict, data=data_json)


'''
data = transcribe_data_json(get_library('http://localhost:32400',
                                        '4CX8sBFPjAVSfJWohux5', "Movies"))

sync_data(data, "http://10.0.1.2:8081/sync", '4CX8sBFPjAVSfJWohux5')
'''
new_user('test', 'http://10.0.1.2:8081/newuser')

'''
download_diff('http://localhost:8081/diff',
              '4CX8sBFPjAVSfJWohux5', "C:/strom/test.json")
'''
